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
	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	gatheringapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/gathering/application"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

func TestPostgresPersistedGatheringClaimIsTransactionalAndReplaySafe(t *testing.T) {
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
	if _, err := db.ExecContext(ctx, `TRUNCATE TABLE idempotency_commands, dungeon_runs, gathering_sessions, inventories, characters`); err != nil {
		t.Fatal(err)
	}

	store, err := gameplaydb.New(db)
	if err != nil {
		t.Fatal(err)
	}
	registry := content.NewRegistry()
	registry.Manifest = content.Manifest{Name: "test", Version: "1", Hash: "postgres"}
	registry.Items["iron_ore"] = content.ItemDefinition{ID: "iron_ore", Name: "Iron Ore", Kind: content.ItemKindMaterial}
	registry.Gathering["iron_mine"] = content.GatheringDefinition{ID: "iron_mine", Discipline: "mining", Name: "Iron Mine", IntervalSeconds: 60, Drops: []content.GatheringDrop{{ItemID: "iron_ore", Probability: 1, MinQuantity: 1, MaxQuantity: 1}}}
	game := gatheringapplication.NewService(registry)
	startedAt := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	session, err := game.Start("session-postgres", "character-postgres", "iron_mine", 91, startedAt)
	if err != nil {
		t.Fatal(err)
	}
	playerInventory, err := inventory.New("character-postgres")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Gathering.Create(ctx, session); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Inventories.Create(ctx, playerInventory); err != nil {
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
