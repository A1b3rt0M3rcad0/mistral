package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	gatheringapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/gathering/application"
	gatheringmemory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/gathering/infra/memory"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
	inventorymemory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/infra/memory"
	sharedmemory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/infra/memory"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

type inlineTransactor struct{}

func (inlineTransactor) WithinTransaction(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

func TestPersistedClaimReplaysWithoutDuplicatingInventory(t *testing.T) {
	registry := content.NewRegistry()
	registry.Manifest = content.Manifest{Name: "test", Version: "1", Hash: "abc"}
	registry.Items["iron_ore"] = content.ItemDefinition{ID: "iron_ore", Name: "Iron Ore", Kind: content.ItemKindMaterial}
	registry.Gathering["iron_mine"] = content.GatheringDefinition{ID: "iron_mine", Discipline: "mining", Name: "Iron Mine", IntervalSeconds: 60, Drops: []content.GatheringDrop{{ItemID: "iron_ore", Probability: 1, MinQuantity: 1, MaxQuantity: 1}}}
	game := gatheringapplication.NewService(registry)
	startedAt := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	session, err := game.Start("session-1", "character-1", "iron_mine", 7, startedAt)
	if err != nil {
		t.Fatal(err)
	}
	playerInventory, err := inventory.New("character-1")
	if err != nil {
		t.Fatal(err)
	}

	sessions := gatheringmemory.NewRepository()
	inventories := inventorymemory.NewRepository()
	if _, err := sessions.Create(context.Background(), session); err != nil {
		t.Fatal(err)
	}
	if _, err := inventories.Create(context.Background(), playerInventory); err != nil {
		t.Fatal(err)
	}
	ledger := sharedmemory.NewIdempotencyLedger()
	service := gatheringapplication.NewPersistedClaimService(game, sessions, inventories, inlineTransactor{}, ledger)

	command := gatheringapplication.ClaimCommand{IdempotencyKey: "claim-1", SessionID: "session-1", CharacterID: "character-1", Now: startedAt.Add(2 * time.Minute)}
	first, err := service.Execute(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	if first.Replayed || first.Resolution.ThroughCycle != 2 {
		t.Fatalf("first result = %#v", first)
	}

	command.Now = startedAt.Add(10 * time.Minute)
	replay, err := service.Execute(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Replayed || replay.Resolution.ThroughCycle != first.Resolution.ThroughCycle {
		t.Fatalf("replay result = %#v", replay)
	}
	storedInventory, err := inventories.Get(context.Background(), "character-1")
	if err != nil {
		t.Fatal(err)
	}
	if got := storedInventory.Value.Quantity("iron_ore"); got != 2 {
		t.Fatalf("iron ore quantity = %d, want 2", got)
	}
	storedSession, err := sessions.Get(context.Background(), "session-1")
	if err != nil {
		t.Fatal(err)
	}
	if storedSession.Value.ClaimedCycles != 2 {
		t.Fatalf("claimed cycles = %d, want 2", storedSession.Value.ClaimedCycles)
	}

	other := gatheringapplication.ClaimCommand{IdempotencyKey: "claim-1", SessionID: "different-session", CharacterID: "character-1", Now: command.Now}
	if _, err := service.Execute(context.Background(), other); !errors.Is(err, persistence.ErrIdempotencyKeyReuse) {
		t.Fatalf("expected idempotency key reuse error, got %v", err)
	}
}
