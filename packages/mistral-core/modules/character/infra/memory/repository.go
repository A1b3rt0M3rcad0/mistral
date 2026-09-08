package memory

import (
	"context"

	character "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/character/domain"
	sharedmemory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/infra/memory"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

type Repository struct {
	store *sharedmemory.Store[character.Character]
}

func NewRepository() *Repository {
	return &Repository{store: sharedmemory.NewStore(func(value character.Character) string { return value.ID }, func(value character.Character) character.Character { return value.Clone() })}
}

func (r *Repository) Create(ctx context.Context, value character.Character) (persistence.Record[character.Character], error) {
	return r.store.Create(ctx, value)
}

func (r *Repository) Get(ctx context.Context, id string) (persistence.Record[character.Character], error) {
	return r.store.Get(ctx, id)
}

func (r *Repository) Save(ctx context.Context, value character.Character, expected persistence.Version) (persistence.Record[character.Character], error) {
	return r.store.Save(ctx, value, expected)
}
