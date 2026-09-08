package application_test

import (
	"context"
	"errors"
	"testing"

	identityapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/identity/application"
	identity "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/identity/domain"
	identitymemory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/identity/infra/memory"
)

func TestAuthorizerRequiresCharacterOwnership(t *testing.T) {
	repository := identitymemory.NewRepository()
	ownership, err := identity.NewOwnership("subject-1", "character-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Bind(context.Background(), ownership); err != nil {
		t.Fatal(err)
	}
	authorizer := identityapplication.NewAuthorizer(repository)
	if err := authorizer.Authorize(context.Background(), "subject-1", "character-1"); err != nil {
		t.Fatalf("owner should be authorized: %v", err)
	}
	if err := authorizer.Authorize(context.Background(), "subject-2", "character-1"); !errors.Is(err, identityapplication.ErrForbidden) {
		t.Fatalf("different subject should be forbidden, got %v", err)
	}
	if err := authorizer.Authorize(context.Background(), "subject-1", "missing"); !errors.Is(err, identityapplication.ErrForbidden) {
		t.Fatalf("unbound character should be forbidden, got %v", err)
	}
}

func TestOwnershipBindingIsImmutablePerCharacter(t *testing.T) {
	repository := identitymemory.NewRepository()
	first, _ := identity.NewOwnership("subject-1", "character-1")
	second, _ := identity.NewOwnership("subject-2", "character-1")
	if err := repository.Bind(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if err := repository.Bind(context.Background(), second); !errors.Is(err, identityapplication.ErrCharacterAlreadyOwned) {
		t.Fatalf("expected immutable ownership error, got %v", err)
	}
}
