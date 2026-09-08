package domain

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

type Stack struct {
	ItemID     string            `json:"item_id"`
	Quantity   int               `json:"quantity"`
	AcquiredAt time.Time         `json:"acquired_at"`
	ExpiresAt  *time.Time        `json:"expires_at,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type Inventory struct {
	CharacterID string  `json:"character_id"`
	Stacks      []Stack `json:"stacks"`
}

func New(characterID string) (Inventory, error) {
	if characterID == "" {
		return Inventory{}, errors.New("character id is required")
	}
	return Inventory{CharacterID: characterID, Stacks: []Stack{}}, nil
}

func (i Inventory) Clone() Inventory {
	clone := Inventory{CharacterID: i.CharacterID, Stacks: make([]Stack, len(i.Stacks))}
	for index, stack := range i.Stacks {
		clone.Stacks[index] = cloneStack(stack)
	}
	return clone
}

func (i Inventory) Quantity(itemID string) int {
	total := 0
	for _, stack := range i.Stacks {
		if stack.ItemID == itemID {
			total += stack.Quantity
		}
	}
	return total
}

func (i *Inventory) Add(itemID string, quantity int, acquiredAt time.Time, expiresAt *time.Time, metadata map[string]string) error {
	if itemID == "" {
		return errors.New("item id is required")
	}
	if quantity <= 0 {
		return errors.New("quantity must be positive")
	}
	if acquiredAt.IsZero() {
		return errors.New("acquired_at is required")
	}
	if expiresAt != nil && !expiresAt.After(acquiredAt) {
		return errors.New("expires_at must be after acquired_at")
	}

	candidate := Stack{
		ItemID:     itemID,
		Quantity:   quantity,
		AcquiredAt: acquiredAt,
		ExpiresAt:  cloneTime(expiresAt),
		Metadata:   cloneMetadata(metadata),
	}
	for index := range i.Stacks {
		if compatible(i.Stacks[index], candidate) {
			i.Stacks[index].Quantity += quantity
			if acquiredAt.Before(i.Stacks[index].AcquiredAt) {
				i.Stacks[index].AcquiredAt = acquiredAt
			}
			return nil
		}
	}
	i.Stacks = append(i.Stacks, candidate)
	return nil
}

func (i *Inventory) ConsumeMany(requirements map[string]int) error {
	if len(requirements) == 0 {
		return errors.New("at least one requirement is required")
	}
	for itemID, quantity := range requirements {
		if itemID == "" || quantity <= 0 {
			return fmt.Errorf("invalid requirement %q x%d", itemID, quantity)
		}
		if available := i.Quantity(itemID); available < quantity {
			return fmt.Errorf("insufficient item %s: need %d, have %d", itemID, quantity, available)
		}
	}

	working := i.Clone()
	for itemID, quantity := range requirements {
		working.consume(itemID, quantity)
	}
	*i = working
	return nil
}

func (i *Inventory) consume(itemID string, quantity int) {
	indices := make([]int, 0)
	for index, stack := range i.Stacks {
		if stack.ItemID == itemID && stack.Quantity > 0 {
			indices = append(indices, index)
		}
	}
	sort.SliceStable(indices, func(left, right int) bool {
		a := i.Stacks[indices[left]]
		b := i.Stacks[indices[right]]
		if expiresBefore(a.ExpiresAt, b.ExpiresAt) {
			return true
		}
		if expiresBefore(b.ExpiresAt, a.ExpiresAt) {
			return false
		}
		return a.AcquiredAt.Before(b.AcquiredAt)
	})

	remaining := quantity
	for _, index := range indices {
		if remaining == 0 {
			break
		}
		take := min(remaining, i.Stacks[index].Quantity)
		i.Stacks[index].Quantity -= take
		remaining -= take
	}

	compacted := i.Stacks[:0]
	for _, stack := range i.Stacks {
		if stack.Quantity > 0 {
			compacted = append(compacted, stack)
		}
	}
	i.Stacks = compacted
}

func compatible(a, b Stack) bool {
	return a.ItemID == b.ItemID && sameTime(a.ExpiresAt, b.ExpiresAt) && sameMetadata(a.Metadata, b.Metadata)
}

func expiresBefore(a, b *time.Time) bool {
	if a == nil {
		return false
	}
	if b == nil {
		return true
	}
	return a.Before(*b)
}

func sameTime(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}

func sameMetadata(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		other, ok := b[key]
		if !ok || other != value {
			return false
		}
	}
	return true
}

func cloneStack(stack Stack) Stack {
	stack.ExpiresAt = cloneTime(stack.ExpiresAt)
	stack.Metadata = cloneMetadata(stack.Metadata)
	return stack
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneMetadata(value map[string]string) map[string]string {
	if len(value) == 0 {
		return nil
	}
	copy := make(map[string]string, len(value))
	for key, item := range value {
		copy[key] = item
	}
	return copy
}
