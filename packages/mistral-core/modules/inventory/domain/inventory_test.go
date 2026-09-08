package domain

import (
	"testing"
	"time"
)

func TestInventoryKeepsDifferentExpiryBatchesSeparate(t *testing.T) {
	inventory, err := New("hero")
	if err != nil {
		t.Fatal(err)
	}
	acquired := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	expiresSoon := acquired.Add(24 * time.Hour)
	expiresLater := acquired.Add(72 * time.Hour)

	if err := inventory.Add("raw_trout", 4, acquired, &expiresSoon, nil); err != nil {
		t.Fatal(err)
	}
	if err := inventory.Add("raw_trout", 6, acquired.Add(time.Hour), &expiresLater, nil); err != nil {
		t.Fatal(err)
	}
	if len(inventory.Stacks) != 2 {
		t.Fatalf("expected 2 expiry batches, got %d", len(inventory.Stacks))
	}

	if err := inventory.ConsumeMany(map[string]int{"raw_trout": 5}); err != nil {
		t.Fatal(err)
	}
	if got := inventory.Quantity("raw_trout"); got != 5 {
		t.Fatalf("expected 5 items remaining, got %d", got)
	}
	if len(inventory.Stacks) != 1 || !inventory.Stacks[0].ExpiresAt.Equal(expiresLater) {
		t.Fatal("consumption should use the earliest-expiring batch first")
	}
}

func TestConsumeManyIsAtomic(t *testing.T) {
	inventory, _ := New("hero")
	now := time.Now().UTC()
	_ = inventory.Add("iron_ore", 10, now, nil, nil)
	_ = inventory.Add("coal", 1, now, nil, nil)

	err := inventory.ConsumeMany(map[string]int{"iron_ore": 2, "coal": 2})
	if err == nil {
		t.Fatal("expected insufficient coal error")
	}
	if inventory.Quantity("iron_ore") != 10 || inventory.Quantity("coal") != 1 {
		t.Fatal("failed multi-item consumption must not partially mutate inventory")
	}
}
