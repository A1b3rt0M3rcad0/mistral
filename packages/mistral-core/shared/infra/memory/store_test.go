package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

type fixture struct {
	ID    string
	Value int
}

func TestStoreEnforcesOptimisticConcurrency(t *testing.T) {
	store := NewStore(func(value fixture) string { return value.ID }, func(value fixture) fixture { return value })
	ctx := context.Background()
	created, err := store.Create(ctx, fixture{ID: "x", Value: 1})
	if err != nil {
		t.Fatal(err)
	}
	if created.Version != 1 {
		t.Fatalf("expected version 1, got %d", created.Version)
	}
	updated, err := store.Save(ctx, fixture{ID: "x", Value: 2}, created.Version)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != 2 {
		t.Fatalf("expected version 2, got %d", updated.Version)
	}
	if _, err := store.Save(ctx, fixture{ID: "x", Value: 3}, created.Version); !errors.Is(err, persistence.ErrConflict) {
		t.Fatalf("expected optimistic conflict, got %v", err)
	}
}
