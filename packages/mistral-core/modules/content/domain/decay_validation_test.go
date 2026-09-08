package domain

import (
	"strings"
	"testing"
)

func TestValidateRejectsDecayCycles(t *testing.T) {
	registry := NewRegistry()
	registry.Manifest = Manifest{Name: "test", Version: "1", Hash: "hash"}
	registry.Races["human"] = RaceDefinition{ID: "human", Name: "Human"}
	registry.Items["fresh"] = ItemDefinition{ID: "fresh", Kind: ItemKindConsumable}
	registry.Items["stale"] = ItemDefinition{ID: "stale", Kind: ItemKindConsumable}
	registry.Decay["fresh"] = DecayDefinition{ID: "fresh", ExpiresAfterSeconds: 60, DecayIntoID: "stale"}
	registry.Decay["stale"] = DecayDefinition{ID: "stale", ExpiresAfterSeconds: 60, DecayIntoID: "fresh"}

	err := registry.Validate()
	if err == nil || !strings.Contains(err.Error(), "decay graph contains a cycle") {
		t.Fatalf("expected decay cycle error, got %v", err)
	}
}
