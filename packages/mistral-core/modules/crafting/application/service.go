package application

import (
	"errors"
	"fmt"
	"time"

	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	decay "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/decay/domain"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
)

type Result struct {
	RecipeID         string         `json:"recipe_id"`
	Station          string         `json:"station"`
	Crafts           int            `json:"crafts"`
	Consumed         map[string]int `json:"consumed"`
	ProducedItemID   string         `json:"produced_item_id"`
	ProducedQuantity int            `json:"produced_quantity"`
}

type Service struct {
	registry content.Registry
	decay    decay.Engine
}

func NewService(registry content.Registry) Service {
	return Service{registry: registry, decay: decay.NewEngine()}
}

func (s Service) Craft(playerInventory *inventory.Inventory, recipeID, station string, crafts int, craftedAt time.Time) (Result, error) {
	if playerInventory == nil {
		return Result{}, errors.New("inventory is required")
	}
	if recipeID == "" || station == "" {
		return Result{}, errors.New("recipe id and station are required")
	}
	if crafts <= 0 {
		return Result{}, errors.New("craft count must be positive")
	}
	if craftedAt.IsZero() {
		return Result{}, errors.New("crafted_at is required")
	}

	recipe, ok := s.registry.Recipes[recipeID]
	if !ok {
		return Result{}, fmt.Errorf("unknown recipe %q", recipeID)
	}
	if recipe.Station != station {
		return Result{}, fmt.Errorf("recipe %s requires station %s, got %s", recipe.ID, recipe.Station, station)
	}
	if _, ok := s.registry.Items[recipe.OutputID]; !ok {
		return Result{}, fmt.Errorf("recipe %s references unknown output item %s", recipe.ID, recipe.OutputID)
	}

	requirements := map[string]int{}
	for _, input := range recipe.Inputs {
		requirements[input.ItemID] += input.Quantity * crafts
	}

	working, _, err := s.decay.Resolve(playerInventory.Clone(), s.registry.Decay, craftedAt)
	if err != nil {
		return Result{}, err
	}
	if err := working.ConsumeMany(requirements); err != nil {
		return Result{}, err
	}
	produced := recipe.OutputQty * crafts
	expiresAt, err := decay.ExpirationFor(s.registry.Decay, recipe.OutputID, craftedAt)
	if err != nil {
		return Result{}, err
	}
	if err := working.Add(recipe.OutputID, produced, craftedAt, expiresAt, map[string]string{"source": "crafting", "recipe_id": recipe.ID}); err != nil {
		return Result{}, err
	}
	*playerInventory = working

	return Result{
		RecipeID:         recipe.ID,
		Station:          station,
		Crafts:           crafts,
		Consumed:         requirements,
		ProducedItemID:   recipe.OutputID,
		ProducedQuantity: produced,
	}, nil
}
