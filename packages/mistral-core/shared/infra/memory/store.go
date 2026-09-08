package memory

import (
	"context"
	"errors"
	"sync"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

type Store[T any] struct {
	mu      sync.RWMutex
	records map[string]persistence.Record[T]
	id      func(T) string
	clone   func(T) T
}

func NewStore[T any](id func(T) string, clone func(T) T) *Store[T] {
	return &Store[T]{records: map[string]persistence.Record[T]{}, id: id, clone: clone}
}

func (s *Store[T]) Create(ctx context.Context, value T) (persistence.Record[T], error) {
	if err := ctx.Err(); err != nil {
		return persistence.Record[T]{}, err
	}
	id := s.id(value)
	if id == "" {
		return persistence.Record[T]{}, errors.New("record id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.records[id]; exists {
		return persistence.Record[T]{}, persistence.ErrAlreadyExists
	}
	record := persistence.Record[T]{Value: s.clone(value), Version: 1}
	s.records[id] = record
	return persistence.Record[T]{Value: s.clone(record.Value), Version: record.Version}, nil
}

func (s *Store[T]) Get(ctx context.Context, id string) (persistence.Record[T], error) {
	if err := ctx.Err(); err != nil {
		return persistence.Record[T]{}, err
	}
	if id == "" {
		return persistence.Record[T]{}, errors.New("record id is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.records[id]
	if !ok {
		return persistence.Record[T]{}, persistence.ErrNotFound
	}
	return persistence.Record[T]{Value: s.clone(record.Value), Version: record.Version}, nil
}

func (s *Store[T]) Save(ctx context.Context, value T, expected persistence.Version) (persistence.Record[T], error) {
	if err := ctx.Err(); err != nil {
		return persistence.Record[T]{}, err
	}
	id := s.id(value)
	if id == "" {
		return persistence.Record[T]{}, errors.New("record id is required")
	}
	if expected == 0 {
		return persistence.Record[T]{}, errors.New("expected version must be positive")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.records[id]
	if !ok {
		return persistence.Record[T]{}, persistence.ErrNotFound
	}
	if current.Version != expected {
		return persistence.Record[T]{}, persistence.ErrConflict
	}
	updated := persistence.Record[T]{Value: s.clone(value), Version: current.Version + 1}
	s.records[id] = updated
	return persistence.Record[T]{Value: s.clone(updated.Value), Version: updated.Version}, nil
}
