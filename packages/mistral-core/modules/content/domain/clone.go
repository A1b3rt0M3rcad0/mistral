package domain

func (r Registry) Clone() Registry {
	clone := NewRegistry()
	clone.Manifest = r.Manifest

	for id, race := range r.Races {
		copy := race
		if race.BaseModifiers != nil {
			copy.BaseModifiers = make(map[string]float64, len(race.BaseModifiers))
			for key, value := range race.BaseModifiers {
				copy.BaseModifiers[key] = value
			}
		}
		clone.Races[id] = copy
	}
	for id, item := range r.Items {
		clone.Items[id] = item
	}
	for id, table := range r.LootTables {
		copy := table
		copy.Entries = append([]LootEntry(nil), table.Entries...)
		clone.LootTables[id] = copy
	}
	for id, monster := range r.Monsters {
		clone.Monsters[id] = monster
	}
	for id, recipe := range r.Recipes {
		copy := recipe
		copy.Inputs = append([]RecipeInput(nil), recipe.Inputs...)
		clone.Recipes[id] = copy
	}
	for id, dungeon := range r.Dungeons {
		copy := dungeon
		copy.Tiers = append([]DungeonTierDefinition(nil), dungeon.Tiers...)
		for index := range copy.Tiers {
			copy.Tiers[index].MonsterPool = append([]WeightedMonster(nil), dungeon.Tiers[index].MonsterPool...)
		}
		clone.Dungeons[id] = copy
	}
	for id, gathering := range r.Gathering {
		copy := gathering
		copy.Drops = append([]GatheringDrop(nil), gathering.Drops...)
		clone.Gathering[id] = copy
	}
	for id, decay := range r.Decay {
		clone.Decay[id] = decay
	}
	return clone
}
