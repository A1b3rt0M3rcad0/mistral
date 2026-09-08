package domain

import (
	"errors"
	"fmt"
	"time"

	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
)

type Transformation struct {
	FromItemID string    `json:"from_item_id"`
	ToItemID   string    `json:"to_item_id"`
	Quantity   int       `json:"quantity"`
	OccurredAt time.Time `json:"occurred_at"`
}

type Engine struct{}

func NewEngine() Engine { return Engine{} }

func ExpirationFor(definitions map[string]content.DecayDefinition, itemID string, acquiredAt time.Time) (*time.Time, error) {
	definition, ok := definitions[itemID]
	if !ok {
		return nil, nil
	}
	if definition.ID != itemID || definition.ExpiresAfterSeconds <= 0 || definition.DecayIntoID == "" {
		return nil, fmt.Errorf("invalid decay definition for item %s", itemID)
	}
	if acquiredAt.IsZero() {
		return nil, errors.New("acquired_at is required for perishable item")
	}
	expiresAt := acquiredAt.Add(time.Duration(definition.ExpiresAfterSeconds) * time.Second)
	return &expiresAt, nil
}

func (Engine) Resolve(source inventory.Inventory, definitions map[string]content.DecayDefinition, now time.Time) (inventory.Inventory, []Transformation, error) {
	if source.CharacterID == "" {
		return inventory.Inventory{}, nil, errors.New("inventory character id is required")
	}
	if now.IsZero() {
		return inventory.Inventory{}, nil, errors.New("resolved time is required")
	}
	result, err := inventory.New(source.CharacterID)
	if err != nil {
		return inventory.Inventory{}, nil, err
	}
	transformations := []Transformation{}
	for _, stack := range source.Stacks {
		resolved, stackTransformations, err := resolveStack(stack, definitions, now)
		if err != nil {
			return inventory.Inventory{}, nil, err
		}
		if err := result.Add(resolved.ItemID, resolved.Quantity, resolved.AcquiredAt, resolved.ExpiresAt, resolved.Metadata); err != nil {
			return inventory.Inventory{}, nil, err
		}
		transformations = append(transformations, stackTransformations...)
	}
	return result, transformations, nil
}

func resolveStack(stack inventory.Stack, definitions map[string]content.DecayDefinition, now time.Time) (inventory.Stack, []Transformation, error) {
	current := stack
	current.Metadata = cloneMetadata(stack.Metadata)
	transformations := []Transformation{}
	visited := map[string]struct{}{}
	for current.ExpiresAt != nil && !current.ExpiresAt.After(now) {
		if _, seen := visited[current.ItemID]; seen {
			return inventory.Stack{}, nil, fmt.Errorf("decay cycle encountered at item %s", current.ItemID)
		}
		visited[current.ItemID] = struct{}{}
		definition, ok := definitions[current.ItemID]
		if !ok {
			return inventory.Stack{}, nil, fmt.Errorf("expired item %s has no decay definition", current.ItemID)
		}
		if definition.ID != current.ItemID || definition.ExpiresAfterSeconds <= 0 || definition.DecayIntoID == "" {
			return inventory.Stack{}, nil, fmt.Errorf("invalid decay definition for item %s", current.ItemID)
		}
		occurredAt := *current.ExpiresAt
		transformations = append(transformations, Transformation{
			FromItemID: current.ItemID,
			ToItemID:   definition.DecayIntoID,
			Quantity:   current.Quantity,
			OccurredAt: occurredAt,
		})
		current.ItemID = definition.DecayIntoID
		current.AcquiredAt = occurredAt
		current.ExpiresAt = nil
		if _, chained := definitions[current.ItemID]; chained {
			expiresAt, err := ExpirationFor(definitions, current.ItemID, occurredAt)
			if err != nil {
				return inventory.Stack{}, nil, err
			}
			current.ExpiresAt = expiresAt
		}
	}
	return current, transformations, nil
}

func cloneMetadata(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
