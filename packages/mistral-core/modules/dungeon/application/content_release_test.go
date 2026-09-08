package application

import (
	"testing"
	"time"

	contentapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/application"
	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	dungeon "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/dungeon/domain"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/determinism"
)

func TestResolveUsesPinnedHistoricalContentRelease(t *testing.T) {
	active := content.NewRegistry()
	active.Manifest = content.Manifest{Name: "mistral", Version: "2", Hash: "active"}
	active.Dungeons["mine"] = content.DungeonDefinition{ID: "mine", Tiers: []content.DungeonTierDefinition{{Tier: 1, EncounterIntervalSeconds: 45, MonsterPool: []content.WeightedMonster{{MonsterID: "new_monster", Weight: 1}}}}}

	historical := content.NewRegistry()
	historical.Manifest = content.Manifest{Name: "mistral", Version: "1", Hash: "historical"}
	historical.Dungeons["mine"] = content.DungeonDefinition{ID: "mine", Tiers: []content.DungeonTierDefinition{{Tier: 1, EncounterIntervalSeconds: 45, MonsterPool: []content.WeightedMonster{{MonsterID: "old_monster", Weight: 1}}}}}

	catalog := contentapplication.NewCatalog(active)
	if err := catalog.Add(historical); err != nil {
		t.Fatal(err)
	}
	started := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	run := dungeon.Run{
		ID:             "run-1",
		ContentRelease: historical.Manifest.ReleaseID(),
		RulesetVersion: determinism.RulesV1,
		DungeonID:      "mine",
		DungeonTier:    1,
		PartySize:      1,
		Seed:           42,
		StartedAt:      started,
	}
	resolution, err := NewServiceWithCatalog(catalog, dungeon.NewEngine()).Resolve(run, started.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(resolution.Encounters) != 1 || resolution.Encounters[0].MonsterID != "old_monster" {
		t.Fatalf("historical run resolved against wrong content: %#v", resolution.Encounters)
	}
}
