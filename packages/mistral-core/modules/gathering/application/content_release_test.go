package application

import (
	"testing"
	"time"

	contentapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/application"
	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	gathering "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/gathering/domain"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/determinism"
)

func TestClaimUsesPinnedHistoricalContentRelease(t *testing.T) {
	active := content.NewRegistry()
	active.Manifest = content.Manifest{Name: "mistral", Version: "2", Hash: "active"}
	active.Items["new_ore"] = content.ItemDefinition{ID: "new_ore", Name: "New Ore", Kind: content.ItemKindMaterial}
	active.Gathering["mine"] = content.GatheringDefinition{ID: "mine", IntervalSeconds: 60, Drops: []content.GatheringDrop{{ItemID: "new_ore", Probability: 1, MinQuantity: 1, MaxQuantity: 1}}}

	historical := content.NewRegistry()
	historical.Manifest = content.Manifest{Name: "mistral", Version: "1", Hash: "historical"}
	historical.Items["old_ore"] = content.ItemDefinition{ID: "old_ore", Name: "Old Ore", Kind: content.ItemKindMaterial}
	historical.Gathering["mine"] = content.GatheringDefinition{ID: "mine", IntervalSeconds: 60, Drops: []content.GatheringDrop{{ItemID: "old_ore", Probability: 1, MinQuantity: 1, MaxQuantity: 1}}}

	catalog := contentapplication.NewCatalog(active)
	if err := catalog.Add(historical); err != nil {
		t.Fatal(err)
	}
	started := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	session := gathering.Session{
		ID:             "session-1",
		CharacterID:    "hero",
		ContentRelease: historical.Manifest.ReleaseID(),
		RulesetVersion: determinism.RulesV1,
		GatheringID:    "mine",
		Seed:           42,
		StartedAt:      started,
	}
	playerInventory, err := inventory.New("hero")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewServiceWithCatalog(catalog).Claim(&session, &playerInventory, started.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if got := playerInventory.Quantity("old_ore"); got != 1 {
		t.Fatalf("historical reward quantity = %d, want 1", got)
	}
	if got := playerInventory.Quantity("new_ore"); got != 0 {
		t.Fatalf("active-release reward leaked into historical session: %d", got)
	}
}
