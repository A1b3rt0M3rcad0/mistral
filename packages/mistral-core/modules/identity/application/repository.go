package application

import (
	"context"
	"errors"

	identity "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/identity/domain"
)

var (
	ErrOwnershipNotFound      = errors.New("character ownership not found")
	ErrCharacterAlreadyOwned = errors.New("character already has an owner")
	ErrForbidden             = errors.New("character access forbidden")
)

type Repository interface {
	Bind(context.Context, identity.Ownership) error
	ByCharacter(context.Context, string) (identity.Ownership, error)
}
