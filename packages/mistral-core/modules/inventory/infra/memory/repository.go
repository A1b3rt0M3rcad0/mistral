package memory

import (
	"context"

	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
	sharedmemory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/infra/memory"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

type Repository struct {
	store *sharedmemory.Store[inventory.Inventory]
}

func NewRepository() *Repository {
	return &Repository{store: sharedmemory.NewStore(func(value inventory.Inventory) string { return value.CharacterID }, func(value inventory.Inventory) inventory.Inventory { return value.Clone() })}
}

func (r *Repository) Create(ctx context.Context, value inventory.Inventory) (persistence.Record[inventory.Inventory], error) {
	return r.store.Create(ctx, value)
}

func (r *Repository) Get(ctx context.Context, id string) (persistence.Record[inventory.Inventory], error) {
	return r.store.Get(ctx, id)
}

func (r *Repository) Save(ctx context.Context, value inventory.Inventory, expected persistence.Version) (persistence.Record[inventory.Inventory], error) {
	return r.store.Save(ctx, value, expected)
}
