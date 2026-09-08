package application

import (
	"testing"
	"time"

	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
)

func TestOfflineGatheringUsesProductionTimeForPerishableBatches(t *testing.T) {
	registry := content.NewRegistry()
	registry.Manifest = content.Manifest{Name: "test", Version: "1", Hash: "perishable"}
	registry.Items["fresh_fish"] = content.ItemDefinition{ID: "fresh_fish", Name: "Fresh Fish", Kind: content.ItemKindConsumable}
	registry.Items["spoiled_fish"] = content.ItemDefinition{ID: "spoiled_fish", Name: "Spoiled Fish", Kind: content.ItemKindMaterial}
	registry.Decay["fresh_fish"] = content.DecayDefinition{ID: "fresh_fish", ExpiresAfterSeconds: 60, DecayIntoID: "spoiled_fish"}
	registry.Gathering["fishing_spot"] = content.GatheringDefinition{
		ID:              "fishing_spot",
		Discipline:      "fishing",
		Name:            "Fishing Spot",
		IntervalSeconds: 60,
		Drops:           []content.GatheringDrop{{ItemID: "fresh_fish", Probability: 1, MinQuantity: 1, MaxQuantity: 1}},
	}

	service := NewService(registry)
	startedAt := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	session, err := service.Start("fishing-1", "hero", "fishing_spot", 10, startedAt)
	if err != nil {
		t.Fatal(err)
	}
	playerInventory, err := inventory.New("hero")
	if err != nil {
		t.Fatal(err)
	}
	resolution, err := service.Claim(&session, &playerInventory, startedAt.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(resolution.Batches) != 3 {
		t.Fatalf("batches = %d, want 3", len(resolution.Batches))
	}
	if got := playerInventory.Quantity("spoiled_fish"); got != 2 {
		t.Fatalf("spoiled fish = %d, want 2", got)
	}
	if got := playerInventory.Quantity("fresh_fish"); got != 1 {
		t.Fatalf("fresh fish = %d, want 1", got)
	}
	for _, stack := range playerInventory.Stacks {
		if stack.ItemID == "fresh_fish" {
			if stack.ExpiresAt == nil || !stack.ExpiresAt.Equal(startedAt.Add(4*time.Minute)) {
				t.Fatalf("fresh fish expiry = %v, want %v", stack.ExpiresAt, startedAt.Add(4*time.Minute))
			}
		}
	}
}
