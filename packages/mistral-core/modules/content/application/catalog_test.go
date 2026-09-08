package application

import (
	"errors"
	"testing"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
)

func TestCatalogResolvesActiveAndHistoricalReleases(t *testing.T) {
	active := domain.NewRegistry()
	active.Manifest = domain.Manifest{Name: "mistral", Version: "2", Hash: "active"}
	historical := domain.NewRegistry()
	historical.Manifest = domain.Manifest{Name: "mistral", Version: "1", Hash: "historical"}

	catalog := NewCatalog(active)
	if err := catalog.Add(historical); err != nil {
		t.Fatal(err)
	}
	resolvedActive, err := catalog.Active()
	if err != nil {
		t.Fatal(err)
	}
	if got := resolvedActive.Manifest.ReleaseID(); got != active.Manifest.ReleaseID() {
		t.Fatalf("active release = %q, want %q", got, active.Manifest.ReleaseID())
	}
	resolvedHistorical, err := catalog.Resolve(historical.Manifest.ReleaseID())
	if err != nil {
		t.Fatal(err)
	}
	if got := resolvedHistorical.Manifest.ReleaseID(); got != historical.Manifest.ReleaseID() {
		t.Fatalf("historical release = %q, want %q", got, historical.Manifest.ReleaseID())
	}
	if _, err := catalog.Resolve("missing@sha256:nope"); !errors.Is(err, ErrReleaseNotFound) {
		t.Fatalf("missing release error = %v", err)
	}
	if err := catalog.Add(historical); !errors.Is(err, ErrReleaseAlreadyExists) {
		t.Fatalf("duplicate release error = %v", err)
	}
}

func TestCatalogDoesNotExposeMutableReleaseState(t *testing.T) {
	active := domain.NewRegistry()
	active.Manifest = domain.Manifest{Name: "mistral", Version: "1", Hash: "immutable"}
	active.Races["human"] = domain.RaceDefinition{
		ID:   "human",
		Name: "Human",
		BaseModifiers: map[string]float64{
			"strength": 1,
		},
	}
	active.LootTables["loot"] = domain.LootTableDefinition{
		ID:      "loot",
		Entries: []domain.LootEntry{{ItemID: "ore", Probability: 1, MinQuantity: 1, MaxQuantity: 1}},
	}

	catalog := NewCatalog(active)
	active.Races["human"] = domain.RaceDefinition{ID: "human", Name: "Mutated Source"}
	active.LootTables["loot"] = domain.LootTableDefinition{ID: "loot"}

	first, err := catalog.Active()
	if err != nil {
		t.Fatal(err)
	}
	if first.Races["human"].Name != "Human" || len(first.LootTables["loot"].Entries) != 1 {
		t.Fatalf("catalog changed after source mutation: %#v %#v", first.Races["human"], first.LootTables["loot"])
	}

	race := first.Races["human"]
	race.Name = "Mutated View"
	race.BaseModifiers["strength"] = 999
	first.Races["human"] = race
	loot := first.LootTables["loot"]
	loot.Entries[0].MinQuantity = 999
	first.LootTables["loot"] = loot

	second, err := catalog.Active()
	if err != nil {
		t.Fatal(err)
	}
	if second.Races["human"].Name != "Human" || second.Races["human"].BaseModifiers["strength"] != 1 {
		t.Fatalf("resolved mutation leaked into catalog: %#v", second.Races["human"])
	}
	if len(second.LootTables["loot"].Entries) != 1 || second.LootTables["loot"].Entries[0].MinQuantity != 1 {
		t.Fatalf("resolved loot mutation leaked into catalog: %#v", second.LootTables["loot"])
	}
}
