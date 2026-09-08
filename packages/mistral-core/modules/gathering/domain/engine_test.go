package domain

import (
	"reflect"
	"testing"
	"time"

	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
)

func TestGatheringResolutionDoesNotDependOnClaimBatching(t *testing.T) {
	definition := content.GatheringDefinition{
		ID:              "iron_mine",
		IntervalSeconds: 60,
		Drops: []content.GatheringDrop{
			{ItemID: "iron_ore", Probability: 0.8, MinQuantity: 1, MaxQuantity: 2},
			{ItemID: "coal", Probability: 0.15, MinQuantity: 1, MaxQuantity: 1},
		},
	}
	started := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	base := Session{ID: "g1", CharacterID: "hero", ContentRelease: "0.1.0@sha256:test", GatheringID: "iron_mine", Seed: 42, StartedAt: started}
	engine := NewEngine()

	allAtOnce, err := engine.Resolve(base, definition, started.Add(100*time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	firstHalf, err := engine.Resolve(base, definition, started.Add(50*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	secondSession := base
	secondSession.ClaimedCycles = firstHalf.ThroughCycle
	secondHalf, err := engine.Resolve(secondSession, definition, started.Add(100*time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	combined := sumRewards(firstHalf.Rewards, secondHalf.Rewards)
	if !reflect.DeepEqual(sumRewards(allAtOnce.Rewards), combined) {
		t.Fatalf("claim batching changed deterministic rewards: all=%v split=%v", allAtOnce.Rewards, combined)
	}
}

func sumRewards(groups ...[]Reward) map[string]int {
	result := map[string]int{}
	for _, group := range groups {
		for _, reward := range group {
			result[reward.ItemID] += reward.Quantity
		}
	}
	return result
}
