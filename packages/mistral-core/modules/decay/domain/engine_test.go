package domain

import (
	"testing"
	"time"

	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
)

func TestResolveTransformsExpiredBatchesThroughDecayChain(t *testing.T) {
	started := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	definitions := map[string]content.DecayDefinition{
		"fresh_fish": {ID: "fresh_fish", ExpiresAfterSeconds: 60, DecayIntoID: "stale_fish"},
		"stale_fish": {ID: "stale_fish", ExpiresAfterSeconds: 60, DecayIntoID: "spoiled_fish"},
	}
	playerInventory, err := inventory.New("hero")
	if err != nil {
		t.Fatal(err)
	}
	expiresAt, err := ExpirationFor(definitions, "fresh_fish", started)
	if err != nil {
		t.Fatal(err)
	}
	if err := playerInventory.Add("fresh_fish", 3, started, expiresAt, map[string]string{"source": "fishing"}); err != nil {
		t.Fatal(err)
	}

	resolved, transformations, err := NewEngine().Resolve(playerInventory, definitions, started.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Quantity("fresh_fish") != 0 || resolved.Quantity("stale_fish") != 0 || resolved.Quantity("spoiled_fish") != 3 {
		t.Fatalf("unexpected resolved inventory: %#v", resolved.Stacks)
	}
	if len(transformations) != 2 {
		t.Fatalf("transformations = %d, want 2", len(transformations))
	}
	if !transformations[0].OccurredAt.Equal(started.Add(time.Minute)) || !transformations[1].OccurredAt.Equal(started.Add(2*time.Minute)) {
		t.Fatalf("unexpected transformation times: %#v", transformations)
	}
}
