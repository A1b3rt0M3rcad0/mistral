package application

import (
	"testing"
	"time"

	character "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/character/domain"
	combat "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/combat/application"
	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	dungeon "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/dungeon/domain"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
)

type winningCombat struct{}

func (winningCombat) Resolve(combat.Request) (combat.Outcome, error) {
	return combat.Outcome{Victory: true}, nil
}

type consumeOnAttempt struct{}

func (consumeOnAttempt) Apply(inv inventory.Inventory, keyItemID string, _ bool) (inventory.Inventory, error) {
	if err := inv.ConsumeMany(map[string]int{keyItemID: 1}); err != nil {
		return inventory.Inventory{}, err
	}
	return inv, nil
}

func TestBossVictoryUnlocksNextDefinedTier(t *testing.T) {
	registry := content.NewRegistry()
	registry.Items["boss_key"] = content.ItemDefinition{ID: "boss_key", Kind: content.ItemKindKey}
	registry.Items["boss_token"] = content.ItemDefinition{ID: "boss_token", Kind: content.ItemKindMaterial}
	registry.LootTables["boss_loot"] = content.LootTableDefinition{ID: "boss_loot", Entries: []content.LootEntry{{ItemID: "boss_token", Probability: 1, MinQuantity: 1, MaxQuantity: 1}}}
	registry.Monsters["boss"] = content.MonsterDefinition{ID: "boss", Name: "Boss", Level: 1, LootTableID: "boss_loot"}
	registry.Dungeons["dungeon"] = content.DungeonDefinition{ID: "dungeon", Tiers: []content.DungeonTierDefinition{
		{Tier: 1, EncounterIntervalSeconds: 60, BossID: "boss", BossKeyItemID: "boss_key", MonsterPool: []content.WeightedMonster{{MonsterID: "boss", Weight: 1}}},
		{Tier: 2, EncounterIntervalSeconds: 60, BossID: "boss", BossKeyItemID: "boss_key", MonsterPool: []content.WeightedMonster{{MonsterID: "boss", Weight: 1}}},
	}}

	playerCharacter, _ := character.New("hero", "human", []string{"dungeon"})
	playerInventory, _ := inventory.New("hero")
	now := time.Now().UTC()
	_ = playerInventory.Add("boss_key", 1, now, nil, nil)

	service := NewBossService(registry, winningCombat{}, consumeOnAttempt{})
	result, err := service.Challenge(&playerCharacter, &playerInventory, "dungeon", 1, dungeon.CharacterSnapshot{CharacterID: "hero", Level: 1, Attack: 1, Defense: 1, MaxHealth: 1}, 42, now)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Victory || !result.NextTierAvailable || result.UnlockedTier != 2 {
		t.Fatalf("unexpected boss result: %+v", result)
	}
	if !playerCharacter.CanEnter("dungeon", 2) {
		t.Fatal("tier 2 should be unlocked")
	}
	if playerInventory.Quantity("boss_key") != 0 {
		t.Fatal("test key policy should consume the boss key")
	}
	if playerInventory.Quantity("boss_token") != 1 {
		t.Fatal("boss loot should be materialized atomically with victory")
	}
}
