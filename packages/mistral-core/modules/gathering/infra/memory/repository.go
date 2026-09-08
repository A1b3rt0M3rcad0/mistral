package memory

import (
	"context"

	gathering "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/gathering/domain"
	sharedmemory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/infra/memory"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

type Repository struct {
	store *sharedmemory.Store[gathering.Session]
}

func NewRepository() *Repository {
	return &Repository{store: sharedmemory.NewStore(func(value gathering.Session) string { return value.ID }, func(value gathering.Session) gathering.Session { return value })}
}

func (r *Repository) Create(ctx context.Context, value gathering.Session) (persistence.Record[gathering.Session], error) {
	return r.store.Create(ctx, value)
}

func (r *Repository) Get(ctx context.Context, id string) (persistence.Record[gathering.Session], error) {
	return r.store.Get(ctx, id)
}

func (r *Repository) Save(ctx context.Context, value gathering.Session, expected persistence.Version) (persistence.Record[gathering.Session], error) {
	return r.store.Save(ctx, value, expected)
}
