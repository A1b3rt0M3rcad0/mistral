package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	character "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/character/domain"
	charactermemory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/character/infra/memory"
	combat "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/combat/application"
	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	dungeonapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/dungeon/application"
	dungeon "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/dungeon/domain"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
	inventorymemory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/infra/memory"
	sharedmemory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/infra/memory"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

type winningCombat struct{}

func (winningCombat) Resolve(combat.Request) (combat.Outcome, error) {
	return combat.Outcome{Victory: true}, nil
}

type consumeKey struct{}

func (consumeKey) Apply(playerInventory inventory.Inventory, keyItemID string, _ bool) (inventory.Inventory, error) {
	if err := playerInventory.ConsumeMany(map[string]int{keyItemID: 1}); err != nil {
		return inventory.Inventory{}, err
	}
	return playerInventory, nil
}

type inlineTransactor struct{}

func (inlineTransactor) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestPersistedBossReplaysWithoutDuplicatingProgressionOrLoot(t *testing.T) {
	registry := content.NewRegistry()
	registry.Items["boss_key"] = content.ItemDefinition{ID: "boss_key", Kind: content.ItemKindKey}
	registry.Items["boss_token"] = content.ItemDefinition{ID: "boss_token", Kind: content.ItemKindMaterial}
	registry.LootTables["boss_loot"] = content.LootTableDefinition{ID: "boss_loot", Entries: []content.LootEntry{{ItemID: "boss_token", Probability: 1, MinQuantity: 1, MaxQuantity: 1}}}
	registry.Monsters["boss"] = content.MonsterDefinition{ID: "boss", Name: "Boss", Level: 1, LootTableID: "boss_loot"}
	registry.Dungeons["dungeon"] = content.DungeonDefinition{ID: "dungeon", Tiers: []content.DungeonTierDefinition{
		{Tier: 1, EncounterIntervalSeconds: 60, BossID: "boss", BossKeyItemID: "boss_key", MonsterPool: []content.WeightedMonster{{MonsterID: "boss", Weight: 1}}},
		{Tier: 2, EncounterIntervalSeconds: 60, BossID: "boss", BossKeyItemID: "boss_key", MonsterPool: []content.WeightedMonster{{MonsterID: "boss", Weight: 1}}},
	}}

	playerCharacter, err := character.New("hero", "human", []string{"dungeon"})
	if err != nil {
		t.Fatal(err)
	}
	playerInventory, err := inventory.New("hero")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	if err := playerInventory.Add("boss_key", 1, now, nil, nil); err != nil {
		t.Fatal(err)
	}

	characters := charactermemory.NewRepository()
	inventories := inventorymemory.NewRepository()
	if _, err := characters.Create(context.Background(), playerCharacter); err != nil {
		t.Fatal(err)
	}
	if _, err := inventories.Create(context.Background(), playerInventory); err != nil {
		t.Fatal(err)
	}
	ledger := sharedmemory.NewIdempotencyLedger()
	boss := dungeonapplication.NewBossService(registry, winningCombat{}, consumeKey{})
	service := dungeonapplication.NewPersistedBossService(boss, characters, inventories, inlineTransactor{}, ledger)
	command := dungeonapplication.BossCommand{
		IdempotencyKey: "boss-1",
		CharacterID:    "hero",
		DungeonID:      "dungeon",
		Tier:           1,
		Snapshot:       dungeon.CharacterSnapshot{CharacterID: "hero", Level: 1, Attack: 1, Defense: 1, MaxHealth: 1},
		Seed:           42,
		Now:            now,
	}

	first, err := service.Execute(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	if first.Replayed || !first.Result.Victory || first.Result.UnlockedTier != 2 {
		t.Fatalf("first result = %#v", first)
	}
	command.Now = now.Add(time.Minute)
	replay, err := service.Execute(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Replayed || replay.Result.UnlockedTier != 2 {
		t.Fatalf("replay result = %#v", replay)
	}

	storedCharacter, err := characters.Get(context.Background(), "hero")
	if err != nil {
		t.Fatal(err)
	}
	if !storedCharacter.Value.CanEnter("dungeon", 2) || storedCharacter.Value.MaxUnlockedTier("dungeon") != 2 {
		t.Fatalf("unexpected progression: %#v", storedCharacter.Value.UnlockedDungeonTiers)
	}
	storedInventory, err := inventories.Get(context.Background(), "hero")
	if err != nil {
		t.Fatal(err)
	}
	if got := storedInventory.Value.Quantity("boss_key"); got != 0 {
		t.Fatalf("boss key quantity = %d, want 0", got)
	}
	if got := storedInventory.Value.Quantity("boss_token"); got != 1 {
		t.Fatalf("boss token quantity = %d, want 1", got)
	}

	changed := command
	changed.Tier = 2
	if _, err := service.Execute(context.Background(), changed); !errors.Is(err, persistence.ErrIdempotencyKeyReuse) {
		t.Fatalf("expected idempotency key reuse error, got %v", err)
	}
}
