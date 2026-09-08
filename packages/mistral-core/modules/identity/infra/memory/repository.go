package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/identity/application"
	identity "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/identity/domain"
)

type Repository struct {
	mu     sync.RWMutex
	owners map[string]identity.Ownership
}

func NewRepository() *Repository {
	return &Repository{owners: map[string]identity.Ownership{}}
}

func (r *Repository) Bind(ctx context.Context, ownership identity.Ownership) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	validated, err := identity.NewOwnership(ownership.SubjectID, ownership.CharacterID)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.owners[validated.CharacterID]; exists {
		return application.ErrCharacterAlreadyOwned
	}
	r.owners[validated.CharacterID] = validated
	return nil
}

func (r *Repository) ByCharacter(ctx context.Context, characterID string) (identity.Ownership, error) {
	if err := ctx.Err(); err != nil {
		return identity.Ownership{}, err
	}
	if characterID == "" {
		return identity.Ownership{}, application.ErrOwnershipNotFound
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	ownership, ok := r.owners[characterID]
	if !ok {
		return identity.Ownership{}, application.ErrOwnershipNotFound
	}
	return ownership, nil
}

func (r *Repository) BySubject(ctx context.Context, subjectID string) ([]identity.Ownership, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	ownerships := make([]identity.Ownership, 0)
	for _, ownership := range r.owners {
		if ownership.SubjectID == subjectID {
			ownerships = append(ownerships, ownership)
		}
	}
	sort.Slice(ownerships, func(left, right int) bool {
		return ownerships[left].CharacterID < ownerships[right].CharacterID
	})
	return ownerships, nil
}

var _ application.Repository = (*Repository)(nil)
