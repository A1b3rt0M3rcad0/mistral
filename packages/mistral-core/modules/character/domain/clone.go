package domain

func (c Character) Clone() Character {
	clone := c
	clone.UnlockedDungeonTiers = make(map[string]int, len(c.UnlockedDungeonTiers))
	for dungeonID, tier := range c.UnlockedDungeonTiers {
		clone.UnlockedDungeonTiers[dungeonID] = tier
	}
	return clone
}
