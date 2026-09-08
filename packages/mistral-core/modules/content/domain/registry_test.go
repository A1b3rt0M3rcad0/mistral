package domain

import (
	"strings"
	"testing"
)

func TestValidateRejectsEquipmentInMonsterLoot(t *testing.T) {
	registry := NewRegistry()
	registry.Manifest = Manifest{Name: "test", Version: "0.0.1", Hash: "test-hash"}
	registry.Races["human"] = RaceDefinition{ID: "human", Name: "Human"}
	registry.Items["iron_sword"] = ItemDefinition{ID: "iron_sword", Name: "Iron Sword", Kind: ItemKindEquipment}
	registry.LootTables["illegal"] = LootTableDefinition{ID: "illegal", Entries: []LootEntry{{ItemID: "iron_sword", Probability: 1, MinQuantity: 1, MaxQuantity: 1}}}

	err := registry.Validate()
	if err == nil || !strings.Contains(err.Error(), "craft-only") {
		t.Fatalf("expected craft-only loot invariant error, got %v", err)
	}
}
