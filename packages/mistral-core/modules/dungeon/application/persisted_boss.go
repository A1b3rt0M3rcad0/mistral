package application

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	character "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/character/domain"
	dungeon "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/dungeon/domain"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

const bossChallengeScope = "dungeon.boss.challenge"

type CharacterRepository interface {
	Create(context.Context, character.Character) (persistence.Record[character.Character], error)
	Get(context.Context, string) (persistence.Record[character.Character], error)
	Save(context.Context, character.Character, persistence.Version) (persistence.Record[character.Character], error)
}

type InventoryRepository interface {
	Create(context.Context, inventory.Inventory) (persistence.Record[inventory.Inventory], error)
	Get(context.Context, string) (persistence.Record[inventory.Inventory], error)
	Save(context.Context, inventory.Inventory, persistence.Version) (persistence.Record[inventory.Inventory], error)
}

type BossCommand struct {
	IdempotencyKey string
	CharacterID    string
	DungeonID      string
	Tier           int
	Snapshot       dungeon.CharacterSnapshot
	Seed           int64
	Now            time.Time
}

type BossCommandResult struct {
	Result   BossResult `json:"result"`
	Replayed bool       `json:"replayed"`
}

type PersistedBossService struct {
	game        BossService
	characters  CharacterRepository
	inventories InventoryRepository
	transactor  persistence.Transactor
	ledger      persistence.IdempotencyLedger
}

func NewPersistedBossService(game BossService, characters CharacterRepository, inventories InventoryRepository, transactor persistence.Transactor, ledger persistence.IdempotencyLedger) PersistedBossService {
	return PersistedBossService{game: game, characters: characters, inventories: inventories, transactor: transactor, ledger: ledger}
}

func (s PersistedBossService) Execute(ctx context.Context, command BossCommand) (BossCommandResult, error) {
	if command.IdempotencyKey == "" || command.CharacterID == "" || command.DungeonID == "" {
		return BossCommandResult{}, errors.New("idempotency key, character id and dungeon id are required")
	}
	if command.Tier <= 0 {
		return BossCommandResult{}, errors.New("dungeon tier must be positive")
	}
	if command.Now.IsZero() {
		return BossCommandResult{}, errors.New("command time is required")
	}
	if command.Snapshot.CharacterID != command.CharacterID {
		return BossCommandResult{}, errors.New("combat snapshot belongs to a different character")
	}
	if s.characters == nil || s.inventories == nil || s.transactor == nil || s.ledger == nil {
		return BossCommandResult{}, errors.New("persisted boss dependencies are required")
	}
	intent, err := json.Marshal(struct {
		CharacterID string                    `json:"character_id"`
		DungeonID   string                    `json:"dungeon_id"`
		Tier        int                       `json:"tier"`
		Snapshot    dungeon.CharacterSnapshot `json:"snapshot"`
		Seed        int64                     `json:"seed"`
	}{CharacterID: command.CharacterID, DungeonID: command.DungeonID, Tier: command.Tier, Snapshot: command.Snapshot, Seed: command.Seed})
	if err != nil {
		return BossCommandResult{}, err
	}
	requestHash := persistence.HashRequest(intent)
	scope := bossChallengeScope + ":" + command.CharacterID
	var result BossCommandResult

	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		claim, err := s.ledger.Claim(txCtx, persistence.ClaimRequest{Scope: scope, Key: command.IdempotencyKey, RequestHash: requestHash, ClaimedAt: command.Now})
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

		characterRecord, err := s.characters.Get(txCtx, command.CharacterID)
		if err != nil {
			return err
		}
		inventoryRecord, err := s.inventories.Get(txCtx, command.CharacterID)
		if err != nil {
			return err
		}
		playerCharacter := characterRecord.Value
		playerInventory := inventoryRecord.Value
		bossResult, err := s.game.Challenge(&playerCharacter, &playerInventory, command.DungeonID, command.Tier, command.Snapshot, command.Seed, command.Now)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(characterRecord.Value, playerCharacter) {
			if _, err := s.characters.Save(txCtx, playerCharacter, characterRecord.Version); err != nil {
				return err
			}
		}
		if !reflect.DeepEqual(inventoryRecord.Value, playerInventory) {
			if _, err := s.inventories.Save(txCtx, playerInventory, inventoryRecord.Version); err != nil {
				return err
			}
		}

		result = BossCommandResult{Result: bossResult}
		response, err := json.Marshal(result)
		if err != nil {
			return err
		}
		_, err = s.ledger.Complete(txCtx, scope, command.IdempotencyKey, requestHash, response, command.Now)
		return err
	})
	return result, err
}
