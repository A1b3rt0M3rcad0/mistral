package domain

import (
	"testing"
	"time"

	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/determinism"
)

func TestGatheringResolutionRejectsUnknownRulesetVersion(t *testing.T) {
	started := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	session := Session{
		ID:             "g1",
		CharacterID:    "hero",
		ContentRelease: "0.1.0@sha256:test",
		RulesetVersion: determinism.Version("mistral.rules.future"),
		GatheringID:    "iron_mine",
		Seed:           42,
		StartedAt:      started,
	}
	definition := content.GatheringDefinition{
		ID:              "iron_mine",
		IntervalSeconds: 60,
		Drops:           []content.GatheringDrop{{ItemID: "iron_ore", Probability: 1, MinQuantity: 1, MaxQuantity: 1}},
	}
	if _, err := NewEngine().Resolve(session, definition, started.Add(time.Minute)); err == nil {
		t.Fatal("unsupported gathering ruleset version was accepted")
	}
}
