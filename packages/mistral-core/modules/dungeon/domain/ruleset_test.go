package domain

import (
	"testing"
	"time"

	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/determinism"
)

func TestDungeonResolutionRejectsUnknownRulesetVersion(t *testing.T) {
	started := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	run := Run{
		ID:             "run-1",
		ContentRelease: "0.1.0@sha256:test",
		RulesetVersion: determinism.Version("mistral.rules.future"),
		DungeonID:      "mine",
		DungeonTier:    1,
		PartySize:      1,
		StartedAt:      started,
	}
	tier := content.DungeonTierDefinition{
		Tier:                     1,
		EncounterIntervalSeconds: 45,
		MonsterPool:              []content.WeightedMonster{{MonsterID: "goblin", Weight: 1}},
	}
	if _, err := NewEngine().Resolve(run, tier, started.Add(time.Minute)); err == nil {
		t.Fatal("unsupported dungeon ruleset version was accepted")
	}
}
