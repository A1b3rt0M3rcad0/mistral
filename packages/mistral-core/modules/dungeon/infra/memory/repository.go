package memory

import (
	"context"

	dungeon "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/dungeon/domain"
	sharedmemory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/infra/memory"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

type Repository struct {
	store *sharedmemory.Store[dungeon.Run]
}

func NewRepository() *Repository {
	return &Repository{store: sharedmemory.NewStore(func(value dungeon.Run) string { return value.ID }, func(value dungeon.Run) dungeon.Run { return value })}
}

func (r *Repository) Create(ctx context.Context, value dungeon.Run) (persistence.Record[dungeon.Run], error) {
	return r.store.Create(ctx, value)
}

func (r *Repository) Get(ctx context.Context, id string) (persistence.Record[dungeon.Run], error) {
	return r.store.Get(ctx, id)
}

func (r *Repository) Save(ctx context.Context, value dungeon.Run, expected persistence.Version) (persistence.Record[dungeon.Run], error) {
	return r.store.Save(ctx, value, expected)
}
