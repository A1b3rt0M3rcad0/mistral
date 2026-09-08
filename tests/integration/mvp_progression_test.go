package integration_test

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"

	characterapp "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/character/application"
	combat "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/combat/application"
	contentcomposition "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/composition"
	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	contententrypoint "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/entrypoint"
	crafting "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/crafting/application"
	dungeonapp "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/dungeon/application"
	dungeon "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/dungeon/domain"
	gathering "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/gathering/application"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
)

type alwaysWin struct{}

func (alwaysWin) Resolve(combat.Request) (combat.Outcome, error) {
	return combat.Outcome{Victory: true}, nil
}

type consumeBossKeyOnAttempt struct{}

func (consumeBossKeyOnAttempt) Apply(inv inventory.Inventory, keyItemID string, _ bool) (inventory.Inventory, error) {
	if err := inv.ConsumeMany(map[string]int{keyItemID: 1}); err != nil {
		return inventory.Inventory{}, err
	}
	return inv, nil
}

func TestCurrentContentProgressionSlice(t *testing.T) {
	registry := loadRegistry(t)
	now := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)

	characterService := characterapp.NewService(registry)
	playerCharacter, err := characterService.Create("hero", "human")
	if err != nil {
		t.Fatal(err)
	}
	playerInventory, err := inventory.New(playerCharacter.ID)
	if err != nil {
		t.Fatal(err)
	}

	gatheringService := gathering.NewService(registry)
	session, err := gatheringService.Start("iron-session", playerCharacter.ID, "iron_mine", 42, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gatheringService.Claim(&session, &playerInventory, now.Add(100*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if playerInventory.Quantity("iron_ore") < 6 || playerInventory.Quantity("coal") < 3 {
		t.Fatalf("deterministic mining fixture did not yield enough smelting inputs: ore=%d coal=%d", playerInventory.Quantity("iron_ore"), playerInventory.Quantity("coal"))
	}

	craftingService := crafting.NewService(registry)
	if _, err := craftingService.Craft(&playerInventory, "smelt_iron_ingot", "blacksmith", 3, now.Add(101*time.Minute)); err != nil {
		t.Fatal(err)
	}

	// The current content release defines the Iron Sword recipe but does not yet
	// define acquisition chains for oak_handle and pre-dungeon leather_strip.
	// These two ingredients are fixture-seeded so this test proves crafting
	// mechanics without silently inventing new game content.
	if err := playerInventory.Add("oak_handle", 1, now, nil, map[string]string{"source": "test_fixture_pending_content_contract"}); err != nil {
		t.Fatal(err)
	}
	if err := playerInventory.Add("leather_strip", 1, now, nil, map[string]string{"source": "test_fixture_pending_content_contract"}); err != nil {
		t.Fatal(err)
	}
	if _, err := craftingService.Craft(&playerInventory, "iron_sword", "blacksmith", 1, now.Add(102*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if playerInventory.Quantity("iron_sword") != 1 {
		t.Fatal("iron sword should be produced by crafting, never direct loot")
	}

	_, tier, err := registry.DungeonTier("abandoned_mine", 1)
	if err != nil {
		t.Fatal(err)
	}
	run := dungeon.Run{
		ID:             "run-1",
		ContentRelease: registry.Manifest.ReleaseID(),
		DungeonID:      "abandoned_mine",
		DungeonTier:    1,
		PartySize:      1,
		Seed:           42,
		StartedAt:      now,
		Character: dungeon.CharacterSnapshot{
			CharacterID: playerCharacter.ID,
			Level:       playerCharacter.Level,
			Attack:      1,
			Defense:     1,
			MaxHealth:   1,
		},
	}
	resolution, err := dungeon.NewEngine().Resolve(run, tier, now.Add(200*45*time.Second))
	if err != nil {
		t.Fatal(err)
	}

	rewardService := dungeonapp.NewRewardService(registry)
	resolver := alwaysWin{}
	for _, encounter := range resolution.Encounters {
		monster := registry.Monsters[encounter.MonsterID]
		outcome, err := resolver.Resolve(combat.Request{Character: run.Character, Monster: monster, Seed: run.Seed + int64(encounter.Ordinal)})
		if err != nil {
			t.Fatal(err)
		}
		if outcome.Victory {
			if _, err := rewardService.MaterializeDefeatedEncounter(&playerInventory, run, encounter, encounter.DueAt); err != nil {
				t.Fatal(err)
			}
		}
	}
	if playerInventory.Quantity("goblin_king_key") < 1 {
		t.Fatal("deterministic dungeon fixture should find at least one Goblin King key")
	}

	bossService := dungeonapp.NewBossService(registry, resolver, consumeBossKeyOnAttempt{})
	bossResult, err := bossService.Challenge(&playerCharacter, &playerInventory, "abandoned_mine", 1, run.Character, run.Seed, now.Add(201*45*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if !bossResult.Victory {
		t.Fatal("test combat resolver should defeat the Goblin King")
	}
	if bossResult.NextTierAvailable {
		t.Fatal("content release 0.1.0 currently defines no Abandoned Mine tier 2; progression engine must not invent one")
	}
}

func loadRegistry(t *testing.T) content.Registry {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve integration test path")
	}
	contentRoot := filepath.Join(filepath.Dir(file), "..", "..", "content")
	service := contentcomposition.NewService(contentRoot)
	registry, err := contententrypoint.New(service).LoadRelease()
	if err != nil {
		t.Fatalf("load content: %v", err)
	}
	return registry
}
