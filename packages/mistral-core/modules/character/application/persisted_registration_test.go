package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	characterapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/character/application"
	charactermemory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/character/infra/memory"
	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	identityapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/identity/application"
	identitymemory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/identity/infra/memory"
	inventorymemory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/infra/memory"
	sharedmemory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/infra/memory"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

type registrationInlineTransactor struct{}

func (registrationInlineTransactor) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestPersistedRegistrationCreatesCharacterInventoryAndOwnershipOnce(t *testing.T) {
	registry := content.NewRegistry()
	registry.Races["human"] = content.RaceDefinition{ID: "human", Name: "Human"}
	game := characterapplication.NewService(registry)
	characters := charactermemory.NewRepository()
	inventories := inventorymemory.NewRepository()
	ownership := identitymemory.NewRepository()
	ledger := sharedmemory.NewIdempotencyLedger()
	service := characterapplication.NewPersistedRegistrationService(game, characters, inventories, ownership, registrationInlineTransactor{}, ledger)
	now := time.Date(2026, 9, 8, 1, 0, 0, 0, time.UTC)
	command := characterapplication.RegistrationCommand{
		IdempotencyKey: "register-1",
		SubjectID:      "subject-1",
		CharacterID:    "hero-1",
		RaceID:         "human",
		Now:            now,
	}

	first, err := service.Execute(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	if first.Replayed || first.Character.ID != "hero-1" || first.Character.RaceID != "human" {
		t.Fatalf("unexpected first registration: %#v", first)
	}
	if _, err := characters.Get(context.Background(), "hero-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := inventories.Get(context.Background(), "hero-1"); err != nil {
		t.Fatal(err)
	}
	authorizer := identityapplication.NewAuthorizer(ownership)
	if err := authorizer.Authorize(context.Background(), "subject-1", "hero-1"); err != nil {
		t.Fatalf("registered subject should own character: %v", err)
	}

	replay, err := service.Execute(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Replayed || replay.Character.ID != first.Character.ID {
		t.Fatalf("unexpected replay registration: %#v", replay)
	}

	command.SubjectID = "subject-2"
	if _, err := service.Execute(context.Background(), command); !errors.Is(err, persistence.ErrIdempotencyKeyReuse) {
		t.Fatalf("expected idempotency key reuse error, got %v", err)
	}
}
