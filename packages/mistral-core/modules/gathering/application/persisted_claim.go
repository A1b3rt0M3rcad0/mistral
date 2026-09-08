package application

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	gathering "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/gathering/domain"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

const gatheringClaimScope = "gathering.claim"

type InventoryRepository interface {
	Create(context.Context, inventory.Inventory) (persistence.Record[inventory.Inventory], error)
	Get(context.Context, string) (persistence.Record[inventory.Inventory], error)
	Save(context.Context, inventory.Inventory, persistence.Version) (persistence.Record[inventory.Inventory], error)
}

type ClaimCommand struct {
	IdempotencyKey string
	SessionID      string
	CharacterID    string
	Now            time.Time
}

type ClaimCommandResult struct {
	Resolution gathering.Resolution `json:"resolution"`
	Replayed   bool                 `json:"replayed"`
}

type PersistedClaimService struct {
	game        Service
	sessions    Repository
	inventories InventoryRepository
	transactor  persistence.Transactor
	ledger      persistence.IdempotencyLedger
}

func NewPersistedClaimService(game Service, sessions Repository, inventories InventoryRepository, transactor persistence.Transactor, ledger persistence.IdempotencyLedger) PersistedClaimService {
	return PersistedClaimService{game: game, sessions: sessions, inventories: inventories, transactor: transactor, ledger: ledger}
}

func (s PersistedClaimService) Execute(ctx context.Context, command ClaimCommand) (ClaimCommandResult, error) {
	if command.IdempotencyKey == "" || command.SessionID == "" || command.CharacterID == "" {
		return ClaimCommandResult{}, errors.New("idempotency key, session id and character id are required")
	}
	if command.Now.IsZero() {
		return ClaimCommandResult{}, errors.New("command time is required")
	}
	if s.sessions == nil || s.inventories == nil || s.transactor == nil || s.ledger == nil {
		return ClaimCommandResult{}, errors.New("persisted claim dependencies are required")
	}
	intent, err := json.Marshal(struct {
		SessionID   string `json:"session_id"`
		CharacterID string `json:"character_id"`
	}{SessionID: command.SessionID, CharacterID: command.CharacterID})
	if err != nil {
		return ClaimCommandResult{}, err
	}
	requestHash := persistence.HashRequest(intent)
	var result ClaimCommandResult

	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		claim, err := s.ledger.Claim(txCtx, persistence.ClaimRequest{Scope: gatheringClaimScope, Key: command.IdempotencyKey, RequestHash: requestHash, ClaimedAt: command.Now})
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

		sessionRecord, err := s.sessions.Get(txCtx, command.SessionID)
		if err != nil {
			return err
		}
		if sessionRecord.Value.CharacterID != command.CharacterID {
			return errors.New("gathering session belongs to a different character")
		}
		inventoryRecord, err := s.inventories.Get(txCtx, command.CharacterID)
		if err != nil {
			return err
		}

		session := sessionRecord.Value
		playerInventory := inventoryRecord.Value
		resolution, err := s.game.Claim(&session, &playerInventory, command.Now)
		if err != nil {
			return err
		}
		if resolution.ThroughCycle > sessionRecord.Value.ClaimedCycles {
			if _, err := s.sessions.Save(txCtx, session, sessionRecord.Version); err != nil {
				return err
			}
			if _, err := s.inventories.Save(txCtx, playerInventory, inventoryRecord.Version); err != nil {
				return err
			}
		}
		result = ClaimCommandResult{Resolution: resolution}
		response, err := json.Marshal(result)
		if err != nil {
			return err
		}
		_, err = s.ledger.Complete(txCtx, gatheringClaimScope, command.IdempotencyKey, requestHash, response, command.Now)
		return err
	})
	return result, err
}
