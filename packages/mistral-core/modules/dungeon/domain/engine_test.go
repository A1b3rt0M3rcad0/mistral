package domain

import (
	"reflect"
	"testing"
	"time"

	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
)

func TestResolveIsDeterministic(t *testing.T) {
	startedAt := time.Date(2026, 9, 7, 21, 0, 0, 0, time.UTC)
	run := Run{
		ID:             "run-1",
		ContentRelease: "0.1.0",
		DungeonID:      "abandoned_mine",
		DungeonTier:    1,
		PartySize:      1,
		Seed:           42,
		StartedAt:      startedAt,
		Character:      CharacterSnapshot{CharacterID: "char-1", Level: 1, Attack: 10, Defense: 5, MaxHealth: 100},
	}
	tier := content.DungeonTierDefinition{
		Tier:                     1,
		EncounterIntervalSeconds: 45,
		MonsterPool: []content.WeightedMonster{
			{MonsterID: "goblin", Weight: 80},
			{MonsterID: "cave_spider", Weight: 20},
		},
	}

	engine := NewEngine()
	now := startedAt.Add(3*time.Minute + 1*time.Second)
	first, err := engine.Resolve(run, tier, now)
	if err != nil {
		t.Fatal(err)
	}
	second, err := engine.Resolve(run, tier, now)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("expected deterministic resolution, got %#v and %#v", first, second)
	}
	if len(first.Encounters) != 4 {
		t.Fatalf("expected 4 encounters, got %d", len(first.Encounters))
	}
	if first.Encounters[0].DueAt != startedAt.Add(45*time.Second) {
		t.Fatalf("unexpected first encounter due time: %s", first.Encounters[0].DueAt)
	}
}

func TestResolveRejectsPartyAboveMVPBound(t *testing.T) {
	startedAt := time.Now().UTC()
	run := Run{ID: "run-1", ContentRelease: "0.1.0", DungeonID: "mine", DungeonTier: 1, PartySize: 5, StartedAt: startedAt}
	tier := content.DungeonTierDefinition{Tier: 1, EncounterIntervalSeconds: 45, MonsterPool: []content.WeightedMonster{{MonsterID: "goblin", Weight: 1}}}
	_, err := NewEngine().Resolve(run, tier, startedAt)
	if err == nil {
		t.Fatal("expected party size validation error")
	}
}
