package application

import (
	"testing"
	"time"

	contentapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/application"
	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	dungeon "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/dungeon/domain"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/determinism"
)

func TestRewardMaterializationUsesPinnedHistoricalContentRelease(t *testing.T) {
	active := content.NewRegistry()
	active.Manifest = content.Manifest{Name: "mistral", Version: "2", Hash: "active"}
	active.Items["new_drop"] = content.ItemDefinition{ID: "new_drop", Name: "New Drop", Kind: content.ItemKindMaterial}
	active.LootTables["monster_loot"] = content.LootTableDefinition{ID: "monster_loot", Entries: []content.LootEntry{{ItemID: "new_drop", Probability: 1, MinQuantity: 1, MaxQuantity: 1}}}
	active.Monsters["monster"] = content.MonsterDefinition{ID: "monster", Name: "Monster", Level: 1, LootTableID: "monster_loot"}

	historical := content.NewRegistry()
	historical.Manifest = content.Manifest{Name: "mistral", Version: "1", Hash: "historical"}
	historical.Items["old_drop"] = content.ItemDefinition{ID: "old_drop", Name: "Old Drop", Kind: content.ItemKindMaterial}
	historical.LootTables["monster_loot"] = content.LootTableDefinition{ID: "monster_loot", Entries: []content.LootEntry{{ItemID: "old_drop", Probability: 1, MinQuantity: 1, MaxQuantity: 1}}}
	historical.Monsters["monster"] = content.MonsterDefinition{ID: "monster", Name: "Monster", Level: 1, LootTableID: "monster_loot"}

	catalog := contentapplication.NewCatalog(active)
	if err := catalog.Add(historical); err != nil {
		t.Fatal(err)
	}
	playerInventory, err := inventory.New("hero")
	if err != nil {
		t.Fatal(err)
	}
	run := dungeon.Run{
		ID:             "run-1",
		ContentRelease: historical.Manifest.ReleaseID(),
		RulesetVersion: determinism.RulesV1,
		Seed:           42,
		Character:      dungeon.CharacterSnapshot{CharacterID: "hero"},
	}
	encounter := dungeon.Encounter{Ordinal: 1, MonsterID: "monster"}
	if _, err := NewRewardServiceWithCatalog(catalog).MaterializeDefeatedEncounter(&playerInventory, run, encounter, time.Date(2026, 9, 8, 0, 1, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if got := playerInventory.Quantity("old_drop"); got != 1 {
		t.Fatalf("historical loot quantity = %d, want 1", got)
	}
	if got := playerInventory.Quantity("new_drop"); got != 0 {
		t.Fatalf("active-release loot leaked into historical run: %d", got)
	}
}
