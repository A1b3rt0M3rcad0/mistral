package domain

import (
	"errors"
	"fmt"
)

type Character struct {
	ID                   string         `json:"id"`
	RaceID               string         `json:"race_id"`
	Level                int            `json:"level"`
	UnlockedDungeonTiers map[string]int `json:"unlocked_dungeon_tiers"`
}

func New(id, raceID string, initialDungeonIDs []string) (Character, error) {
	if id == "" {
		return Character{}, errors.New("character id is required")
	}
	if raceID == "" {
		return Character{}, errors.New("race id is required")
	}

	unlocked := make(map[string]int, len(initialDungeonIDs))
	for _, dungeonID := range initialDungeonIDs {
		if dungeonID == "" {
			return Character{}, errors.New("initial dungeon id cannot be empty")
		}
		unlocked[dungeonID] = 1
	}

	return Character{
		ID:                   id,
		RaceID:               raceID,
		Level:                1,
		UnlockedDungeonTiers: unlocked,
	}, nil
}

func (c Character) MaxUnlockedTier(dungeonID string) int {
	return c.UnlockedDungeonTiers[dungeonID]
}

func (c Character) CanEnter(dungeonID string, tier int) bool {
	return tier > 0 && c.MaxUnlockedTier(dungeonID) >= tier
}

func (c *Character) AdvanceDungeon(dungeonID string, defeatedTier int) error {
	if dungeonID == "" {
		return errors.New("dungeon id is required")
	}
	if defeatedTier <= 0 {
		return errors.New("defeated tier must be positive")
	}
	if c.UnlockedDungeonTiers == nil {
		return errors.New("character dungeon progression is not initialized")
	}

	current := c.UnlockedDungeonTiers[dungeonID]
	if current == 0 {
		return fmt.Errorf("dungeon %q is not unlocked", dungeonID)
	}
	if current < defeatedTier {
		return fmt.Errorf("cannot advance dungeon %q from locked tier %d; max unlocked tier is %d", dungeonID, defeatedTier, current)
	}
	if current > defeatedTier {
		return nil
	}

	c.UnlockedDungeonTiers[dungeonID] = defeatedTier + 1
	return nil
}
