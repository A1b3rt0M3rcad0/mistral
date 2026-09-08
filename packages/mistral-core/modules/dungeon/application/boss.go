package application

import (
	"errors"
	"fmt"
	"time"

	character "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/character/domain"
	combat "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/combat/application"
	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	decay "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/decay/domain"
	dungeon "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/dungeon/domain"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
	loot "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/loot/domain"
)

type BossKeyPolicy interface {
	Apply(inventory.Inventory, string, bool) (inventory.Inventory, error)
}

type BossResult struct {
	BossID            string        `json:"boss_id"`
	KeyItemID         string        `json:"key_item_id"`
	Victory           bool          `json:"victory"`
	NextTierAvailable bool          `json:"next_tier_available"`
	UnlockedTier      int           `json:"unlocked_tier,omitempty"`
	Rewards           []loot.Reward `json:"rewards"`
}

type BossService struct {
	registry  content.Registry
	combat    combat.Resolver
	keyPolicy BossKeyPolicy
	loot      loot.Engine
	decay     decay.Engine
}

func NewBossService(registry content.Registry, resolver combat.Resolver, keyPolicy BossKeyPolicy) BossService {
	return BossService{registry: registry, combat: resolver, keyPolicy: keyPolicy, loot: loot.NewEngine(), decay: decay.NewEngine()}
}

func (s BossService) Challenge(playerCharacter *character.Character, playerInventory *inventory.Inventory, dungeonID string, tier int, snapshot dungeon.CharacterSnapshot, seed int64, resolvedAt time.Time) (BossResult, error) {
	if playerCharacter == nil || playerInventory == nil {
		return BossResult{}, errors.New("character and inventory are required")
	}
	if s.combat == nil || s.keyPolicy == nil {
		return BossResult{}, errors.New("combat resolver and boss key policy are required")
	}
	if playerCharacter.ID != playerInventory.CharacterID || snapshot.CharacterID != playerCharacter.ID {
		return BossResult{}, errors.New("character, inventory and combat snapshot must belong to the same character")
	}
	if !playerCharacter.CanEnter(dungeonID, tier) {
		return BossResult{}, fmt.Errorf("character cannot enter dungeon %s tier %d", dungeonID, tier)
	}
	_, definition, err := s.registry.DungeonTier(dungeonID, tier)
	if err != nil {
		return BossResult{}, err
	}
	workingInventory, _, err := s.decay.Resolve(playerInventory.Clone(), s.registry.Decay, resolvedAt)
	if err != nil {
		return BossResult{}, err
	}
	if workingInventory.Quantity(definition.BossKeyItemID) < 1 {
		return BossResult{}, fmt.Errorf("boss key %s is required", definition.BossKeyItemID)
	}
	boss, ok := s.registry.Monsters[definition.BossID]
	if !ok {
		return BossResult{}, fmt.Errorf("unknown boss %q", definition.BossID)
	}
	table, ok := s.registry.LootTables[boss.LootTableID]
	if !ok {
		return BossResult{}, fmt.Errorf("unknown boss loot table %q", boss.LootTableID)
	}

	outcome, err := s.combat.Resolve(combat.Request{
		Character: snapshot,
		Monster:   boss,
		Seed:      seed + int64(tier)*999983,
		Boss:      true,
	})
	if err != nil {
		return BossResult{}, err
	}

	workingInventory, err = s.keyPolicy.Apply(workingInventory, definition.BossKeyItemID, outcome.Victory)
	if err != nil {
		return BossResult{}, err
	}
	result := BossResult{BossID: boss.ID, KeyItemID: definition.BossKeyItemID, Victory: outcome.Victory, Rewards: []loot.Reward{}}
	if !outcome.Victory {
		*playerInventory = workingInventory
		return result, nil
	}

	rewards, err := s.loot.Roll(table, seed+int64(tier)*15485863)
	if err != nil {
		return BossResult{}, err
	}
	for _, reward := range rewards {
		expiresAt, err := decay.ExpirationFor(s.registry.Decay, reward.ItemID, resolvedAt)
		if err != nil {
			return BossResult{}, err
		}
		if err := workingInventory.Add(reward.ItemID, reward.Quantity, resolvedAt, expiresAt, map[string]string{
			"source":     "boss_loot",
			"dungeon_id": dungeonID,
			"boss_id":    boss.ID,
		}); err != nil {
			return BossResult{}, err
		}
	}
	result.Rewards = rewards

	workingCharacter := playerCharacter.Clone()
	if _, _, err := s.registry.DungeonTier(dungeonID, tier+1); err == nil {
		if err := workingCharacter.AdvanceDungeon(dungeonID, tier); err != nil {
			return BossResult{}, err
		}
		result.NextTierAvailable = true
		result.UnlockedTier = tier + 1
	}

	*playerInventory = workingInventory
	*playerCharacter = workingCharacter
	return result, nil
}
