package domain

import (
	"testing"

	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/determinism"
)

func TestLootRejectsUnknownRulesetVersion(t *testing.T) {
	table := content.LootTableDefinition{
		ID:      "loot",
		Entries: []content.LootEntry{{ItemID: "ore", Probability: 1, MinQuantity: 1, MaxQuantity: 1}},
	}
	if _, err := NewEngine().RollVersioned(determinism.Version("mistral.rules.future"), table, 42); err == nil {
		t.Fatal("unsupported loot ruleset version was accepted")
	}
}
