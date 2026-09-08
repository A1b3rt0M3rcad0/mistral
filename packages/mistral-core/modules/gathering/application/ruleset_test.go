package application

import (
	"testing"
	"time"

	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/determinism"
)

func TestStartPinsCurrentDeterministicRuleset(t *testing.T) {
	registry := content.NewRegistry()
	registry.Manifest = content.Manifest{Name: "mistral", Version: "1", Hash: "test"}
	registry.Gathering["iron_mine"] = content.GatheringDefinition{ID: "iron_mine", IntervalSeconds: 60}
	started := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	session, err := NewService(registry).Start("g1", "hero", "iron_mine", 42, started)
	if err != nil {
		t.Fatal(err)
	}
	if session.RulesetVersion != determinism.Current {
		t.Fatalf("ruleset version = %q, want %q", session.RulesetVersion, determinism.Current)
	}
}
