package domain

import (
	"errors"
	"fmt"
)

type Registry struct {
	Manifest   Manifest
	Races      map[string]RaceDefinition
	Items      map[string]ItemDefinition
	LootTables map[string]LootTableDefinition
	Monsters   map[string]MonsterDefinition
	Recipes    map[string]RecipeDefinition
	Dungeons   map[string]DungeonDefinition
	Gathering  map[string]GatheringDefinition
}

func NewRegistry() Registry {
	return Registry{
		Races:      map[string]RaceDefinition{},
		Items:      map[string]ItemDefinition{},
		LootTables: map[string]LootTableDefinition{},
		Monsters:   map[string]MonsterDefinition{},
		Recipes:    map[string]RecipeDefinition{},
		Dungeons:   map[string]DungeonDefinition{},
		Gathering:  map[string]GatheringDefinition{},
	}
}

func (r Registry) Validate() error {
	var errs []error
	if r.Manifest.Name == "" {
		errs = append(errs, errors.New("manifest.name is required"))
	}
	if r.Manifest.Version == "" {
		errs = append(errs, errors.New("manifest.version is required"))
	}
	if r.Manifest.Hash == "" {
		errs = append(errs, errors.New("manifest content hash is required"))
	}
	if len(r.Races) == 0 {
		errs = append(errs, errors.New("at least one race is required"))
	}

	for id, table := range r.LootTables {
		if table.ID != id {
			errs = append(errs, fmt.Errorf("loot table map key %q does not match definition id %q", id, table.ID))
		}
		for index, entry := range table.Entries {
			item, ok := r.Items[entry.ItemID]
			if !ok {
				errs = append(errs, fmt.Errorf("loot table %s entry %d references unknown item %s", id, index, entry.ItemID))
				continue
			}
			if item.Kind == ItemKindEquipment || item.Kind == ItemKindTool {
				errs = append(errs, fmt.Errorf("loot table %s illegally drops craft-only item %s of kind %s", id, item.ID, item.Kind))
			}
			if entry.Probability <= 0 || entry.Probability > 1 {
				errs = append(errs, fmt.Errorf("loot table %s item %s has invalid probability %.4f", id, entry.ItemID, entry.Probability))
			}
			if entry.MinQuantity <= 0 || entry.MaxQuantity < entry.MinQuantity {
				errs = append(errs, fmt.Errorf("loot table %s item %s has invalid quantity range", id, entry.ItemID))
			}
		}
	}

	for id, monster := range r.Monsters {
		if monster.Level <= 0 {
			errs = append(errs, fmt.Errorf("monster %s must have level > 0", id))
		}
		if _, ok := r.LootTables[monster.LootTableID]; !ok {
			errs = append(errs, fmt.Errorf("monster %s references unknown loot table %s", id, monster.LootTableID))
		}
	}

	for id, recipe := range r.Recipes {
		if _, ok := r.Items[recipe.OutputID]; !ok {
			errs = append(errs, fmt.Errorf("recipe %s references unknown output item %s", id, recipe.OutputID))
		}
		if recipe.OutputQty <= 0 {
			errs = append(errs, fmt.Errorf("recipe %s must have output_quantity > 0", id))
		}
		for _, input := range recipe.Inputs {
			if _, ok := r.Items[input.ItemID]; !ok {
				errs = append(errs, fmt.Errorf("recipe %s references unknown input item %s", id, input.ItemID))
			}
			if input.Quantity <= 0 {
				errs = append(errs, fmt.Errorf("recipe %s input %s must have quantity > 0", id, input.ItemID))
			}
		}
	}

	for id, dungeon := range r.Dungeons {
		if len(dungeon.Tiers) == 0 {
			errs = append(errs, fmt.Errorf("dungeon %s must define at least one tier", id))
		}
		for _, tier := range dungeon.Tiers {
			if tier.Tier <= 0 {
				errs = append(errs, fmt.Errorf("dungeon %s has invalid tier %d", id, tier.Tier))
			}
			if tier.EncounterIntervalSeconds <= 0 {
				errs = append(errs, fmt.Errorf("dungeon %s tier %d must have encounter_interval_seconds > 0", id, tier.Tier))
			}
			if _, ok := r.Monsters[tier.BossID]; !ok {
				errs = append(errs, fmt.Errorf("dungeon %s tier %d references unknown boss %s", id, tier.Tier, tier.BossID))
			}
			key, ok := r.Items[tier.BossKeyItemID]
			if !ok {
				errs = append(errs, fmt.Errorf("dungeon %s tier %d references unknown boss key item %s", id, tier.Tier, tier.BossKeyItemID))
			} else if key.Kind != ItemKindKey {
				errs = append(errs, fmt.Errorf("dungeon %s tier %d boss key %s must have kind key", id, tier.Tier, key.ID))
			}
			totalWeight := 0
			for _, weighted := range tier.MonsterPool {
				if _, ok := r.Monsters[weighted.MonsterID]; !ok {
					errs = append(errs, fmt.Errorf("dungeon %s tier %d references unknown monster %s", id, tier.Tier, weighted.MonsterID))
				}
				if weighted.Weight <= 0 {
					errs = append(errs, fmt.Errorf("dungeon %s tier %d monster %s must have weight > 0", id, tier.Tier, weighted.MonsterID))
				}
				totalWeight += weighted.Weight
			}
			if totalWeight <= 0 {
				errs = append(errs, fmt.Errorf("dungeon %s tier %d must have a positive monster pool weight", id, tier.Tier))
			}
		}
	}

	for id, area := range r.Gathering {
		if area.IntervalSeconds <= 0 {
			errs = append(errs, fmt.Errorf("gathering area %s must have interval_seconds > 0", id))
		}
		for _, drop := range area.Drops {
			item, ok := r.Items[drop.ItemID]
			if !ok {
				errs = append(errs, fmt.Errorf("gathering area %s references unknown item %s", id, drop.ItemID))
				continue
			}
			if item.Kind == ItemKindEquipment || item.Kind == ItemKindTool {
				errs = append(errs, fmt.Errorf("gathering area %s cannot directly produce craft-only item %s", id, item.ID))
			}
			if drop.Probability <= 0 || drop.Probability > 1 {
				errs = append(errs, fmt.Errorf("gathering area %s item %s has invalid probability %.4f", id, drop.ItemID, drop.Probability))
			}
		}
	}

	return errors.Join(errs...)
}

func (r Registry) DungeonTier(dungeonID string, tier int) (DungeonDefinition, DungeonTierDefinition, error) {
	dungeon, ok := r.Dungeons[dungeonID]
	if !ok {
		return DungeonDefinition{}, DungeonTierDefinition{}, fmt.Errorf("unknown dungeon %q", dungeonID)
	}
	for _, definition := range dungeon.Tiers {
		if definition.Tier == tier {
			return dungeon, definition, nil
		}
	}
	return DungeonDefinition{}, DungeonTierDefinition{}, fmt.Errorf("dungeon %q has no tier %d", dungeonID, tier)
}
