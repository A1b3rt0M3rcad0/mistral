package gameplaydb_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/database"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/dbmigrate"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/gameplaydb"
	characterapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/character/application"
	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	gatheringapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/gathering/application"
	identityapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/identity/application"
	identity "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/identity/domain"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

type fixedCharacterIDGenerator struct {
	id string
}

func (g fixedCharacterIDGenerator) NewCharacterID() (string, error) {
	return g.id, nil
}

func TestPostgresRegistrationOwnershipAndGatheringAreTransactionalAndReplaySafe(t *testing.T) {
	dsn := os.Getenv("MISTRAL_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("MISTRAL_TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	db, err := database.OpenPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	migrationRoot := filepath.Clean(filepath.Join("..", "..", "..", "..", "migrations"))
	if err := dbmigrate.Apply(ctx, db, migrationRoot); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `TRUNCATE TABLE character_ownerships, idempotency_commands, dungeon_runs, gathering_sessions, inventories, characters`); err != nil {
		t.Fatal(err)
	}

	store, err := gameplaydb.New(db)
	if err != nil {
		t.Fatal(err)
	}
	registry := content.NewRegistry()
	registry.Manifest = content.Manifest{Name: "test", Version: "1", Hash: "postgres"}
	registry.Races["human"] = content.RaceDefinition{ID: "human", Name: "Human"}
	registry.Items["iron_ore"] = content.ItemDefinition{ID: "iron_ore", Name: "Iron Ore", Kind: content.ItemKindMaterial}
	registry.Gathering["iron_mine"] = content.GatheringDefinition{ID: "iron_mine", Discipline: "mining", Name: "Iron Mine", IntervalSeconds: 60, Drops: []content.GatheringDrop{{ItemID: "iron_ore", Probability: 1, MinQuantity: 1, MaxQuantity: 1}}}

	registeredAt := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	registrationService := characterapplication.NewPersistedRegistrationService(
		characterapplication.NewService(registry),
		fixedCharacterIDGenerator{id: "character-postgres"},
		store.Characters,
		store.Inventories,
		store.Ownership,
		store.Transactor,
		store.Idempotency,
	)
	registrationCommand := characterapplication.RegistrationCommand{
		IdempotencyKey: "register-postgres",
		SubjectID:      "subject-postgres",
		RaceID:         "human",
		Now:            registeredAt,
	}
	registration, err := registrationService.Execute(ctx, registrationCommand)
	if err != nil {
		t.Fatal(err)
	}
	if registration.Replayed || registration.Character.ID != "character-postgres" {
		t.Fatalf("unexpected registration result: %#v", registration)
	}
	replayedRegistration, err := registrationService.Execute(ctx, registrationCommand)
	if err != nil {
		t.Fatal(err)
	}
	if !replayedRegistration.Replayed || replayedRegistration.Character.ID != registration.Character.ID {
		t.Fatalf("unexpected registration replay: %#v", replayedRegistration)
	}

	authorizer := identityapplication.NewAuthorizer(store.Ownership)
	if err := authorizer.Authorize(ctx, registrationCommand.SubjectID, registration.Character.ID); err != nil {
		t.Fatalf("owner should be authorized: %v", err)
	}
	if err := authorizer.Authorize(ctx, "other-subject", registration.Character.ID); !errors.Is(err, identityapplication.ErrForbidden) {
		t.Fatalf("non-owner should be forbidden, got %v", err)
	}
	ownedCharacters, err := store.Ownership.BySubject(ctx, registrationCommand.SubjectID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ownedCharacters) != 1 || ownedCharacters[0].CharacterID != registration.Character.ID {
		t.Fatalf("owned characters = %#v", ownedCharacters)
	}
	unownedCharacters, err := store.Ownership.BySubject(ctx, "subject-without-characters")
	if err != nil {
		t.Fatal(err)
	}
	if len(unownedCharacters) != 0 {
		t.Fatalf("unexpected ownerships = %#v", unownedCharacters)
	}
	otherOwnership, _ := identity.NewOwnership("other-subject", registration.Character.ID)
	if err := store.Ownership.Bind(ctx, otherOwnership); !errors.Is(err, identityapplication.ErrCharacterAlreadyOwned) {
		t.Fatalf("ownership rebinding should fail, got %v", err)
	}

	game := gatheringapplication.NewService(registry)
	startedAt := registeredAt.Add(time.Minute)
	session, err := game.Start("session-postgres", registration.Character.ID, "iron_mine", 91, startedAt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Gathering.Create(ctx, session); err != nil {
		t.Fatal(err)
	}

	service := gatheringapplication.NewPersistedClaimService(game, store.Gathering, store.Inventories, store.Transactor, store.Idempotency)
	command := gatheringapplication.ClaimCommand{IdempotencyKey: "claim-postgres", SessionID: session.ID, CharacterID: session.CharacterID, Now: startedAt.Add(3 * time.Minute)}
	first, err := service.Execute(ctx, command)
	if err != nil {
		t.Fatal(err)
	}
	if first.Replayed || first.Resolution.ThroughCycle != 3 {
		t.Fatalf("first result = %#v", first)
	}
	command.Now = startedAt.Add(10 * time.Minute)
	replay, err := service.Execute(ctx, command)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Replayed || replay.Resolution.ThroughCycle != 3 {
		t.Fatalf("replay result = %#v", replay)
	}
	storedInventory, err := store.Inventories.Get(ctx, session.CharacterID)
	if err != nil {
		t.Fatal(err)
	}
	if got := storedInventory.Value.Quantity("iron_ore"); got != 3 {
		t.Fatalf("iron ore quantity = %d, want 3", got)
	}

	original, err := store.Inventories.Get(ctx, session.CharacterID)
	if err != nil {
		t.Fatal(err)
	}
	updated := original.Value.Clone()
	if err := updated.Add("iron_ore", 1, startedAt, nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Inventories.Save(ctx, updated, original.Version); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Inventories.Save(ctx, updated, original.Version); !errors.Is(err, persistence.ErrConflict) {
		t.Fatalf("expected optimistic conflict, got %v", err)
	}
}
