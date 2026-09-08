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

type sequenceCharacterIDs struct {
	ids   []string
	calls int
}

func (g *sequenceCharacterIDs) NewCharacterID() (string, error) {
	if g.calls >= len(g.ids) {
		return "", errors.New("no character ids available")
	}
	id := g.ids[g.calls]
	g.calls++
	return id, nil
}

func newRegistrationFixture(t *testing.T, ids *sequenceCharacterIDs) (characterapplication.PersistedRegistrationService, *charactermemory.Repository, *inventorymemory.Repository, *identitymemory.Repository) {
	t.Helper()
	registry := content.NewRegistry()
	registry.Races["human"] = content.RaceDefinition{ID: "human", Name: "Human"}
	registry.Races["elf"] = content.RaceDefinition{ID: "elf", Name: "Elf"}
	characters := charactermemory.NewRepository()
	inventories := inventorymemory.NewRepository()
	ownership := identitymemory.NewRepository()
	ledger := sharedmemory.NewIdempotencyLedger()
	service := characterapplication.NewPersistedRegistrationService(
		characterapplication.NewService(registry),
		ids,
		characters,
		inventories,
		ownership,
		registrationInlineTransactor{},
		ledger,
	)
	return service, characters, inventories, ownership
}

func TestPersistedRegistrationGeneratesIdentityOnceAndReplaysIt(t *testing.T) {
	ids := &sequenceCharacterIDs{ids: []string{"hero-1", "hero-2"}}
	service, characters, inventories, ownership := newRegistrationFixture(t, ids)
	now := time.Date(2026, 9, 8, 1, 0, 0, 0, time.UTC)
	command := characterapplication.RegistrationCommand{
		IdempotencyKey: "register-1",
		SubjectID:      "subject-1",
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
	if ids.calls != 1 {
		t.Fatalf("id generator calls = %d, want 1", ids.calls)
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
	if ids.calls != 1 {
		t.Fatalf("replay called id generator: calls=%d", ids.calls)
	}

	command.RaceID = "elf"
	if _, err := service.Execute(context.Background(), command); !errors.Is(err, persistence.ErrIdempotencyKeyReuse) {
		t.Fatalf("expected idempotency key reuse error, got %v", err)
	}
}

func TestRegistrationIdempotencyKeyIsScopedPerSubject(t *testing.T) {
	ids := &sequenceCharacterIDs{ids: []string{"hero-1", "hero-2"}}
	service, _, _, ownership := newRegistrationFixture(t, ids)
	now := time.Date(2026, 9, 8, 1, 0, 0, 0, time.UTC)
	first, err := service.Execute(context.Background(), characterapplication.RegistrationCommand{
		IdempotencyKey: "same-key",
		SubjectID:      "subject-1",
		RaceID:         "human",
		Now:            now,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Execute(context.Background(), characterapplication.RegistrationCommand{
		IdempotencyKey: "same-key",
		SubjectID:      "subject-2",
		RaceID:         "human",
		Now:            now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Character.ID == second.Character.ID || first.Character.ID != "hero-1" || second.Character.ID != "hero-2" {
		t.Fatalf("unexpected generated characters: first=%#v second=%#v", first.Character, second.Character)
	}
	authorizer := identityapplication.NewAuthorizer(ownership)
	if err := authorizer.Authorize(context.Background(), "subject-1", "hero-1"); err != nil {
		t.Fatal(err)
	}
	if err := authorizer.Authorize(context.Background(), "subject-2", "hero-2"); err != nil {
		t.Fatal(err)
	}
}
