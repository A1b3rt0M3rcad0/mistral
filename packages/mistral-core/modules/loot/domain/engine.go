package domain

import (
	"errors"
	"fmt"
	"math/rand"

	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
)

type Reward struct {
	ItemID   string `json:"item_id"`
	Quantity int    `json:"quantity"`
}

type Engine struct{}

func NewEngine() Engine { return Engine{} }

func (Engine) Roll(table content.LootTableDefinition, seed int64) ([]Reward, error) {
	if table.ID == "" {
		return nil, errors.New("loot table id is required")
	}
	rng := rand.New(rand.NewSource(seed)) // #nosec G404 -- deterministic gameplay RNG is intentional.
	rewards := make([]Reward, 0, len(table.Entries))
	for _, entry := range table.Entries {
		if entry.ItemID == "" {
			return nil, errors.New("loot entry item id is required")
		}
		if entry.Probability <= 0 || entry.Probability > 1 {
			return nil, fmt.Errorf("invalid probability %.4f for item %s", entry.Probability, entry.ItemID)
		}
		if entry.MinQuantity <= 0 || entry.MaxQuantity < entry.MinQuantity {
			return nil, fmt.Errorf("invalid quantity range for item %s", entry.ItemID)
		}
		if rng.Float64() > entry.Probability {
			continue
		}
		quantity := entry.MinQuantity
		if entry.MaxQuantity > entry.MinQuantity {
			quantity += rng.Intn(entry.MaxQuantity - entry.MinQuantity + 1)
		}
		rewards = append(rewards, Reward{ItemID: entry.ItemID, Quantity: quantity})
	}
	return rewards, nil
}
