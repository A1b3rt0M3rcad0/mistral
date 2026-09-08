package application

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	dungeon "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/dungeon/domain"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
	loot "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/loot/domain"
)

type RewardService struct {
	registry content.Registry
	loot     loot.Engine
}

func NewRewardService(registry content.Registry) RewardService {
	return RewardService{registry: registry, loot: loot.NewEngine()}
}

func (s RewardService) MaterializeDefeatedEncounter(playerInventory *inventory.Inventory, run dungeon.Run, encounter dungeon.Encounter, awardedAt time.Time) ([]loot.Reward, error) {
	if playerInventory == nil {
		return nil, errors.New("inventory is required")
	}
	if playerInventory.CharacterID != run.Character.CharacterID {
		return nil, errors.New("dungeon run and inventory belong to different characters")
	}
	if run.ContentRelease != s.registry.Manifest.ReleaseID() {
		return nil, fmt.Errorf("run content release %s does not match active release %s", run.ContentRelease, s.registry.Manifest.ReleaseID())
	}
	if encounter.Ordinal <= 0 || encounter.MonsterID == "" {
		return nil, errors.New("encounter ordinal and monster id are required")
	}
	monster, ok := s.registry.Monsters[encounter.MonsterID]
	if !ok {
		return nil, fmt.Errorf("unknown monster %q", encounter.MonsterID)
	}
	table, ok := s.registry.LootTables[monster.LootTableID]
	if !ok {
		return nil, fmt.Errorf("unknown loot table %q for monster %s", monster.LootTableID, monster.ID)
	}

	rewards, err := s.loot.Roll(table, run.Seed+int64(encounter.Ordinal)*104729)
	if err != nil {
		return nil, err
	}
	working := playerInventory.Clone()
	for _, reward := range rewards {
		if _, ok := s.registry.Items[reward.ItemID]; !ok {
			return nil, fmt.Errorf("loot references unknown item %s", reward.ItemID)
		}
		if err := working.Add(reward.ItemID, reward.Quantity, awardedAt, nil, map[string]string{
			"source":            "monster_loot",
			"run_id":            run.ID,
			"monster_id":        monster.ID,
			"encounter_ordinal": strconv.Itoa(encounter.Ordinal),
		}); err != nil {
			return nil, err
		}
	}
	*playerInventory = working
	return rewards, nil
}
