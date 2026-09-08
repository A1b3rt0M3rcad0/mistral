package application

import (
	"context"
	"errors"
	"reflect"
	"time"

	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	decay "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/decay/domain"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

type Service struct {
	registry content.Registry
	engine   decay.Engine
}

func NewService(registry content.Registry) Service {
	return Service{registry: registry, engine: decay.NewEngine()}
}

func (s Service) Resolve(playerInventory *inventory.Inventory, now time.Time) ([]decay.Transformation, error) {
	if playerInventory == nil {
		return nil, errors.New("inventory is required")
	}
	resolved, transformations, err := s.engine.Resolve(playerInventory.Clone(), s.registry.Decay, now)
	if err != nil {
		return nil, err
	}
	*playerInventory = resolved
	return transformations, nil
}

type Repository interface {
	Get(context.Context, string) (persistence.Record[inventory.Inventory], error)
	Save(context.Context, inventory.Inventory, persistence.Version) (persistence.Record[inventory.Inventory], error)
}

type PersistedService struct {
	game        Service
	inventories Repository
}

func NewPersistedService(game Service, inventories Repository) PersistedService {
	return PersistedService{game: game, inventories: inventories}
}

func (s PersistedService) Resolve(ctx context.Context, characterID string, now time.Time) ([]decay.Transformation, error) {
	if characterID == "" {
		return nil, errors.New("character id is required")
	}
	if now.IsZero() {
		return nil, errors.New("resolved time is required")
	}
	if s.inventories == nil {
		return nil, errors.New("inventory repository is required")
	}
	record, err := s.inventories.Get(ctx, characterID)
	if err != nil {
		return nil, err
	}
	resolved := record.Value.Clone()
	transformations, err := s.game.Resolve(&resolved, now)
	if err != nil {
		return nil, err
	}
	if reflect.DeepEqual(record.Value, resolved) {
		return transformations, nil
	}
	if _, err := s.inventories.Save(ctx, resolved, record.Version); err != nil {
		return nil, err
	}
	return transformations, nil
}
