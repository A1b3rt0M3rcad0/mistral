package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	craftingapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/crafting/application"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
	inventorymemory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/infra/memory"
	sharedmemory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/infra/memory"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

type inlineTransactor struct{}

func (inlineTransactor) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestPersistedCraftReplaysWithoutDuplicatingOutput(t *testing.T) {
	registry := content.NewRegistry()
	registry.Items["iron_ore"] = content.ItemDefinition{ID: "iron_ore", Name: "Iron Ore", Kind: content.ItemKindMaterial}
	registry.Items["iron_ingot"] = content.ItemDefinition{ID: "iron_ingot", Name: "Iron Ingot", Kind: content.ItemKindComponent}
	registry.Recipes["smelt_iron_ingot"] = content.RecipeDefinition{ID: "smelt_iron_ingot", Station: "blacksmith", Inputs: []content.RecipeInput{{ItemID: "iron_ore", Quantity: 2}}, OutputID: "iron_ingot", OutputQty: 1}
	game := craftingapplication.NewService(registry)
	now := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	playerInventory, err := inventory.New("character-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := playerInventory.Add("iron_ore", 4, now, nil, nil); err != nil {
		t.Fatal(err)
	}

	inventories := inventorymemory.NewRepository()
	if _, err := inventories.Create(context.Background(), playerInventory); err != nil {
		t.Fatal(err)
	}
	ledger := sharedmemory.NewIdempotencyLedger()
	service := craftingapplication.NewPersistedCraftService(game, inventories, inlineTransactor{}, ledger)
	command := craftingapplication.CraftCommand{IdempotencyKey: "craft-1", CharacterID: "character-1", RecipeID: "smelt_iron_ingot", Station: "blacksmith", Crafts: 2, Now: now}

	first, err := service.Execute(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	if first.Replayed || first.Result.ProducedQuantity != 2 {
		t.Fatalf("first result = %#v", first)
	}
	command.Now = now.Add(time.Minute)
	replay, err := service.Execute(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Replayed || replay.Result.ProducedQuantity != 2 {
		t.Fatalf("replay result = %#v", replay)
	}

	stored, err := inventories.Get(context.Background(), "character-1")
	if err != nil {
		t.Fatal(err)
	}
	if got := stored.Value.Quantity("iron_ore"); got != 0 {
		t.Fatalf("iron ore quantity = %d, want 0", got)
	}
	if got := stored.Value.Quantity("iron_ingot"); got != 2 {
		t.Fatalf("iron ingot quantity = %d, want 2", got)
	}

	changed := command
	changed.Crafts = 1
	if _, err := service.Execute(context.Background(), changed); !errors.Is(err, persistence.ErrIdempotencyKeyReuse) {
		t.Fatalf("expected idempotency key reuse error, got %v", err)
	}
}
