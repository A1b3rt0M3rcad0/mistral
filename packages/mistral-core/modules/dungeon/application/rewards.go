package application

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	contentapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/application"
	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	decay "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/decay/domain"
	dungeon "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/dungeon/domain"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
	loot "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/loot/domain"
)

type RewardService struct {
	catalog contentapplication.ReleaseCatalog
	loot    loot.Engine
	decay   decay.Engine
}

func NewRewardService(registry content.Registry) RewardService {
	return NewRewardServiceWithCatalog(contentapplication.NewCatalog(registry))
}

func NewRewardServiceWithCatalog(catalog contentapplication.ReleaseCatalog) RewardService {
	return RewardService{catalog: catalog, loot: loot.NewEngine(), decay: decay.NewEngine()}
}

func (s RewardService) MaterializeDefeatedEncounter(playerInventory *inventory.Inventory, run dungeon.Run, encounter dungeon.Encounter, awardedAt time.Time) ([]loot.Reward, error) {
	if playerInventory == nil {
		return nil, errors.New("inventory is required")
	}
	if playerInventory.CharacterID != run.Character.CharacterID {
		return nil, errors.New("dungeon run and inventory belong to different characters")
	}
	if s.catalog == nil {
		return nil, errors.New("content release catalog is required")
	}
	registry, err := s.catalog.Resolve(run.ContentRelease)
	if err != nil {
		return nil, fmt.Errorf("resolve dungeon reward content release %s: %w", run.ContentRelease, err)
	}
	if encounter.Ordinal <= 0 || encounter.MonsterID == "" {
		return nil, errors.New("encounter ordinal and monster id are required")
	}
	monster, ok := registry.Monsters[encounter.MonsterID]
	if !ok {
		return nil, fmt.Errorf("unknown monster %q in release %s", encounter.MonsterID, run.ContentRelease)
	}
	table, ok := registry.LootTables[monster.LootTableID]
	if !ok {
		return nil, fmt.Errorf("unknown loot table %q for monster %s", monster.LootTableID, monster.ID)
	}

	rewards, err := s.loot.RollVersioned(run.RulesetVersion, table, run.Seed+int64(encounter.Ordinal)*104729)
	if err != nil {
		return nil, err
	}
	working, _, err := s.decay.Resolve(playerInventory.Clone(), registry.Decay, awardedAt)
	if err != nil {
		return nil, err
	}
	for _, reward := range rewards {
		if _, ok := registry.Items[reward.ItemID]; !ok {
			return nil, fmt.Errorf("loot references unknown item %s", reward.ItemID)
		}
		expiresAt, err := decay.ExpirationFor(registry.Decay, reward.ItemID, awardedAt)
		if err != nil {
			return nil, err
		}
		if err := working.Add(reward.ItemID, reward.Quantity, awardedAt, expiresAt, map[string]string{
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
