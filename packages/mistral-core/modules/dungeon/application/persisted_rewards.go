package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	dungeon "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/dungeon/domain"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
	loot "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/loot/domain"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

const encounterRewardScope = "dungeon.encounter.reward"

type RewardInventoryRepository interface {
	Create(context.Context, inventory.Inventory) (persistence.Record[inventory.Inventory], error)
	Get(context.Context, string) (persistence.Record[inventory.Inventory], error)
	Save(context.Context, inventory.Inventory, persistence.Version) (persistence.Record[inventory.Inventory], error)
}

type EncounterRewardCommand struct {
	RunID            string
	EncounterOrdinal int
	Now              time.Time
}

type EncounterRewardCommandResult struct {
	Encounter dungeon.Encounter `json:"encounter"`
	Rewards   []loot.Reward     `json:"rewards"`
	Replayed  bool              `json:"replayed"`
}

type PersistedRewardService struct {
	dungeons    Service
	rewards     RewardService
	runs        RunRepository
	inventories RewardInventoryRepository
	transactor  persistence.Transactor
	ledger      persistence.IdempotencyLedger
}

func NewPersistedRewardService(dungeons Service, rewards RewardService, runs RunRepository, inventories RewardInventoryRepository, transactor persistence.Transactor, ledger persistence.IdempotencyLedger) PersistedRewardService {
	return PersistedRewardService{
		dungeons:    dungeons,
		rewards:     rewards,
		runs:        runs,
		inventories: inventories,
		transactor:  transactor,
		ledger:      ledger,
	}
}

func (s PersistedRewardService) Execute(ctx context.Context, command EncounterRewardCommand) (EncounterRewardCommandResult, error) {
	if command.RunID == "" {
		return EncounterRewardCommandResult{}, errors.New("dungeon run id is required")
	}
	if command.EncounterOrdinal <= 0 {
		return EncounterRewardCommandResult{}, errors.New("encounter ordinal must be positive")
	}
	if command.Now.IsZero() {
		return EncounterRewardCommandResult{}, errors.New("command time is required")
	}
	if s.runs == nil || s.inventories == nil || s.transactor == nil || s.ledger == nil {
		return EncounterRewardCommandResult{}, errors.New("persisted reward dependencies are required")
	}

	intent, err := json.Marshal(struct {
		RunID            string `json:"run_id"`
		EncounterOrdinal int    `json:"encounter_ordinal"`
	}{RunID: command.RunID, EncounterOrdinal: command.EncounterOrdinal})
	if err != nil {
		return EncounterRewardCommandResult{}, err
	}
	requestHash := persistence.HashRequest(intent)
	idempotencyKey := fmt.Sprintf("%s:%d", command.RunID, command.EncounterOrdinal)
	var result EncounterRewardCommandResult

	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		claim, err := s.ledger.Claim(txCtx, persistence.ClaimRequest{
			Scope:       encounterRewardScope,
			Key:         idempotencyKey,
			RequestHash: requestHash,
			ClaimedAt:   command.Now,
		})
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

		runRecord, err := s.runs.Get(txCtx, command.RunID)
		if err != nil {
			return err
		}
		resolution, err := s.dungeons.Resolve(runRecord.Value, command.Now)
		if err != nil {
			return err
		}
		if command.EncounterOrdinal > len(resolution.Encounters) {
			return fmt.Errorf("encounter %d is not due for dungeon run %s", command.EncounterOrdinal, command.RunID)
		}
		encounter := resolution.Encounters[command.EncounterOrdinal-1]
		if encounter.Ordinal != command.EncounterOrdinal {
			return errors.New("resolved encounter ordinal mismatch")
		}

		inventoryRecord, err := s.inventories.Get(txCtx, runRecord.Value.Character.CharacterID)
		if err != nil {
			return err
		}
		playerInventory := inventoryRecord.Value
		rewards, err := s.rewards.MaterializeDefeatedEncounter(&playerInventory, runRecord.Value, encounter, command.Now)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(inventoryRecord.Value, playerInventory) {
			if _, err := s.inventories.Save(txCtx, playerInventory, inventoryRecord.Version); err != nil {
				return err
			}
		}

		result = EncounterRewardCommandResult{Encounter: encounter, Rewards: rewards}
		response, err := json.Marshal(result)
		if err != nil {
			return err
		}
		_, err = s.ledger.Complete(txCtx, encounterRewardScope, idempotencyKey, requestHash, response, command.Now)
		return err
	})
	return result, err
}
