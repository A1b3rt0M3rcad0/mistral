package application_test

import (
	"context"
	"testing"
	"time"

	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	dungeonapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/dungeon/application"
	dungeon "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/dungeon/domain"
	dungeonmemory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/dungeon/infra/memory"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
	inventorymemory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/infra/memory"
	sharedmemory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/infra/memory"
)

type rewardInlineTransactor struct{}

func (rewardInlineTransactor) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestPersistedEncounterRewardUsesDeterministicMaterializationIdentity(t *testing.T) {
	registry := content.NewRegistry()
	registry.Manifest = content.Manifest{Name: "test", Version: "1", Hash: "reward"}
	registry.Items["monster_token"] = content.ItemDefinition{ID: "monster_token", Name: "Monster Token", Kind: content.ItemKindMaterial}
	registry.LootTables["monster_loot"] = content.LootTableDefinition{ID: "monster_loot", Entries: []content.LootEntry{{ItemID: "monster_token", Probability: 1, MinQuantity: 1, MaxQuantity: 1}}}
	registry.Monsters["monster"] = content.MonsterDefinition{ID: "monster", Name: "Monster", Level: 1, LootTableID: "monster_loot"}
	registry.Dungeons["dungeon"] = content.DungeonDefinition{ID: "dungeon", Tiers: []content.DungeonTierDefinition{{
		Tier:                     1,
		EncounterIntervalSeconds: 60,
		MonsterPool:              []content.WeightedMonster{{MonsterID: "monster", Weight: 1}},
		BossID:                   "monster",
		BossKeyItemID:            "monster_token",
	}}}

	startedAt := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	run := dungeon.Run{
		ID:             "run-1",
		ContentRelease: registry.Manifest.ReleaseID(),
		DungeonID:      "dungeon",
		DungeonTier:    1,
		PartySize:      1,
		Seed:           73,
		StartedAt:      startedAt,
		Character:      dungeon.CharacterSnapshot{CharacterID: "hero", Level: 1, Attack: 1, Defense: 1, MaxHealth: 1},
	}
	playerInventory, err := inventory.New("hero")
	if err != nil {
		t.Fatal(err)
	}
	runs := dungeonmemory.NewRepository()
	inventories := inventorymemory.NewRepository()
	if _, err := runs.Create(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	if _, err := inventories.Create(context.Background(), playerInventory); err != nil {
		t.Fatal(err)
	}

	dungeons := dungeonapplication.NewService(registry, dungeon.NewEngine())
	rewards := dungeonapplication.NewRewardService(registry)
	ledger := sharedmemory.NewIdempotencyLedger()
	service := dungeonapplication.NewPersistedRewardService(dungeons, rewards, runs, inventories, rewardInlineTransactor{}, ledger)

	first, err := service.Execute(context.Background(), dungeonapplication.EncounterRewardCommand{RunID: run.ID, EncounterOrdinal: 1, Now: startedAt.Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if first.Replayed || first.Encounter.Ordinal != 1 || len(first.Rewards) != 1 {
		t.Fatalf("first result = %#v", first)
	}

	replay, err := service.Execute(context.Background(), dungeonapplication.EncounterRewardCommand{RunID: run.ID, EncounterOrdinal: 1, Now: startedAt.Add(10 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Replayed || replay.Encounter.Ordinal != 1 {
		t.Fatalf("replay result = %#v", replay)
	}
	stored, err := inventories.Get(context.Background(), "hero")
	if err != nil {
		t.Fatal(err)
	}
	if got := stored.Value.Quantity("monster_token"); got != 1 {
		t.Fatalf("monster token quantity after replay = %d, want 1", got)
	}

	second, err := service.Execute(context.Background(), dungeonapplication.EncounterRewardCommand{RunID: run.ID, EncounterOrdinal: 2, Now: startedAt.Add(2 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if second.Replayed || second.Encounter.Ordinal != 2 {
		t.Fatalf("second result = %#v", second)
	}
	stored, err = inventories.Get(context.Background(), "hero")
	if err != nil {
		t.Fatal(err)
	}
	if got := stored.Value.Quantity("monster_token"); got != 2 {
		t.Fatalf("monster token quantity = %d, want 2", got)
	}
}
