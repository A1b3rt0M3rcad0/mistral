package application

import (
	"testing"
	"time"

	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
)

func TestCraftIsAtomic(t *testing.T) {
	registry := content.NewRegistry()
	registry.Items["iron_ore"] = content.ItemDefinition{ID: "iron_ore", Kind: content.ItemKindMaterial}
	registry.Items["coal"] = content.ItemDefinition{ID: "coal", Kind: content.ItemKindMaterial}
	registry.Items["iron_ingot"] = content.ItemDefinition{ID: "iron_ingot", Kind: content.ItemKindComponent}
	registry.Recipes["smelt_iron_ingot"] = content.RecipeDefinition{
		ID:       "smelt_iron_ingot",
		Station:  "blacksmith",
		Inputs:   []content.RecipeInput{{ItemID: "iron_ore", Quantity: 2}, {ItemID: "coal", Quantity: 1}},
		OutputID: "iron_ingot", OutputQty: 1,
	}

	playerInventory, _ := inventory.New("hero")
	now := time.Now().UTC()
	_ = playerInventory.Add("iron_ore", 10, now, nil, nil)
	_ = playerInventory.Add("coal", 2, now, nil, nil)

	service := NewService(registry)
	if _, err := service.Craft(&playerInventory, "smelt_iron_ingot", "blacksmith", 3, now); err == nil {
		t.Fatal("expected insufficient coal")
	}
	if playerInventory.Quantity("iron_ore") != 10 || playerInventory.Quantity("coal") != 2 || playerInventory.Quantity("iron_ingot") != 0 {
		t.Fatal("failed craft must not mutate inventory")
	}

	_ = playerInventory.Add("coal", 1, now, nil, nil)
	result, err := service.Craft(&playerInventory, "smelt_iron_ingot", "blacksmith", 3, now)
	if err != nil {
		t.Fatal(err)
	}
	if result.ProducedQuantity != 3 || playerInventory.Quantity("iron_ingot") != 3 {
		t.Fatalf("expected 3 iron ingots, result=%+v inventory=%+v", result, playerInventory)
	}
}
