package application

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	character "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/character/domain"
	identity "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/identity/domain"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

const registrationCommandScope = "character.register"

type CharacterIDGenerator interface {
	NewCharacterID() (string, error)
}

type RegistrationInventoryRepository interface {
	Create(context.Context, inventory.Inventory) (persistence.Record[inventory.Inventory], error)
}

type RegistrationOwnershipRepository interface {
	Bind(context.Context, identity.Ownership) error
}

type RegistrationCommand struct {
	IdempotencyKey string
	SubjectID      string
	RaceID         string
	Now            time.Time
}

type RegistrationResult struct {
	Character character.Character `json:"character"`
	Replayed  bool                `json:"replayed"`
}

type PersistedRegistrationService struct {
	game        Service
	ids         CharacterIDGenerator
	characters  Repository
	inventories RegistrationInventoryRepository
	ownership   RegistrationOwnershipRepository
	transactor  persistence.Transactor
	ledger      persistence.IdempotencyLedger
}

func NewPersistedRegistrationService(game Service, ids CharacterIDGenerator, characters Repository, inventories RegistrationInventoryRepository, ownership RegistrationOwnershipRepository, transactor persistence.Transactor, ledger persistence.IdempotencyLedger) PersistedRegistrationService {
	return PersistedRegistrationService{
		game:        game,
		ids:         ids,
		characters:  characters,
		inventories: inventories,
		ownership:   ownership,
		transactor:  transactor,
		ledger:      ledger,
	}
}

func (s PersistedRegistrationService) Execute(ctx context.Context, command RegistrationCommand) (RegistrationResult, error) {
	if command.IdempotencyKey == "" || command.SubjectID == "" || command.RaceID == "" {
		return RegistrationResult{}, errors.New("idempotency key, subject id and race id are required")
	}
	if command.Now.IsZero() {
		return RegistrationResult{}, errors.New("command time is required")
	}
	if s.ids == nil || s.characters == nil || s.inventories == nil || s.ownership == nil || s.transactor == nil || s.ledger == nil {
		return RegistrationResult{}, errors.New("persisted registration dependencies are required")
	}
	intent, err := json.Marshal(struct {
		SubjectID string `json:"subject_id"`
		RaceID    string `json:"race_id"`
	}{SubjectID: command.SubjectID, RaceID: command.RaceID})
	if err != nil {
		return RegistrationResult{}, err
	}
	requestHash := persistence.HashRequest(intent)
	scope := registrationCommandScope + ":" + command.SubjectID
	var result RegistrationResult

	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		claim, err := s.ledger.Claim(txCtx, persistence.ClaimRequest{
			Scope:       scope,
			Key:         command.IdempotencyKey,
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

		characterID, err := s.ids.NewCharacterID()
		if err != nil {
			return err
		}
		if characterID == "" {
			return errors.New("generated character id is empty")
		}
		playerCharacter, err := s.game.Create(characterID, command.RaceID)
		if err != nil {
			return err
		}
		playerInventory, err := inventory.New(characterID)
		if err != nil {
			return err
		}
		ownership, err := identity.NewOwnership(command.SubjectID, characterID)
		if err != nil {
			return err
		}
		if _, err := s.characters.Create(txCtx, playerCharacter); err != nil {
			return err
		}
		if _, err := s.inventories.Create(txCtx, playerInventory); err != nil {
			return err
		}
		if err := s.ownership.Bind(txCtx, ownership); err != nil {
			return err
		}

		result = RegistrationResult{Character: playerCharacter}
		response, err := json.Marshal(result)
		if err != nil {
			return err
		}
		_, err = s.ledger.Complete(txCtx, scope, command.IdempotencyKey, requestHash, response, command.Now)
		return err
	})
	return result, err
}
