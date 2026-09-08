package application

import (
	"context"
	"errors"
)

type Authorizer struct {
	repository Repository
}

func NewAuthorizer(repository Repository) Authorizer {
	return Authorizer{repository: repository}
}

func (a Authorizer) Authorize(ctx context.Context, subjectID, characterID string) error {
	if subjectID == "" || characterID == "" {
		return ErrForbidden
	}
	if a.repository == nil {
		return errors.New("ownership repository is required")
	}
	ownership, err := a.repository.ByCharacter(ctx, characterID)
	if err != nil {
		if errors.Is(err, ErrOwnershipNotFound) {
			return ErrForbidden
		}
		return err
	}
	if ownership.SubjectID != subjectID {
		return ErrForbidden
	}
	return nil
}
