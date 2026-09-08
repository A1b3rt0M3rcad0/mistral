package application

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

const craftCommandScope = "crafting.craft"

type InventoryRepository interface {
	Create(context.Context, inventory.Inventory) (persistence.Record[inventory.Inventory], error)
	Get(context.Context, string) (persistence.Record[inventory.Inventory], error)
	Save(context.Context, inventory.Inventory, persistence.Version) (persistence.Record[inventory.Inventory], error)
}

type CraftCommand struct {
	IdempotencyKey string
	CharacterID    string
	RecipeID       string
	Station        string
	Crafts         int
	Now            time.Time
}

type CraftCommandResult struct {
	Result   Result `json:"result"`
	Replayed bool   `json:"replayed"`
}

type PersistedCraftService struct {
	game        Service
	inventories InventoryRepository
	transactor  persistence.Transactor
	ledger      persistence.IdempotencyLedger
}

func NewPersistedCraftService(game Service, inventories InventoryRepository, transactor persistence.Transactor, ledger persistence.IdempotencyLedger) PersistedCraftService {
	return PersistedCraftService{game: game, inventories: inventories, transactor: transactor, ledger: ledger}
}

func (s PersistedCraftService) Execute(ctx context.Context, command CraftCommand) (CraftCommandResult, error) {
	if command.IdempotencyKey == "" || command.CharacterID == "" || command.RecipeID == "" || command.Station == "" {
		return CraftCommandResult{}, errors.New("idempotency key, character id, recipe id and station are required")
	}
	if command.Crafts <= 0 {
		return CraftCommandResult{}, errors.New("craft count must be positive")
	}
	if command.Now.IsZero() {
		return CraftCommandResult{}, errors.New("command time is required")
	}
	if s.inventories == nil || s.transactor == nil || s.ledger == nil {
		return CraftCommandResult{}, errors.New("persisted craft dependencies are required")
	}
	intent, err := json.Marshal(struct {
		CharacterID string `json:"character_id"`
		RecipeID    string `json:"recipe_id"`
		Station     string `json:"station"`
		Crafts      int    `json:"crafts"`
	}{CharacterID: command.CharacterID, RecipeID: command.RecipeID, Station: command.Station, Crafts: command.Crafts})
	if err != nil {
		return CraftCommandResult{}, err
	}
	requestHash := persistence.HashRequest(intent)
	var result CraftCommandResult

	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		claim, err := s.ledger.Claim(txCtx, persistence.ClaimRequest{Scope: craftCommandScope, Key: command.IdempotencyKey, RequestHash: requestHash, ClaimedAt: command.Now})
		if err != nil {
			return err
		}
		switch claim.Disposition {
		case persistence.ClaimReplay:
			if err := json.Unmarshal(claim.Entry.Response, &result); err != nil {
				return err
			}
			result.Replayed = true
			return nil
		case persistence.ClaimInProgress:
			return persistence.ErrCommandInProgress
		case persistence.ClaimAcquired:
		default:
			return errors.New("unknown idempotency claim disposition")
		}

		inventoryRecord, err := s.inventories.Get(txCtx, command.CharacterID)
		if err != nil {
			return err
		}
		playerInventory := inventoryRecord.Value
		craftResult, err := s.game.Craft(&playerInventory, command.RecipeID, command.Station, command.Crafts, command.Now)
		if err != nil {
			return err
		}
		if _, err := s.inventories.Save(txCtx, playerInventory, inventoryRecord.Version); err != nil {
			return err
		}

		result = CraftCommandResult{Result: craftResult}
		response, err := json.Marshal(result)
		if err != nil {
			return err
		}
		_, err = s.ledger.Complete(txCtx, craftCommandScope, command.IdempotencyKey, requestHash, response, command.Now)
		return err
	})
	return result, err
}
