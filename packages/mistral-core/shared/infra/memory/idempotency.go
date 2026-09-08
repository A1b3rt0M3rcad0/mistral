package memory

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

type IdempotencyLedger struct {
	mu      sync.Mutex
	entries map[string]persistence.CommandEntry
}

func NewIdempotencyLedger() *IdempotencyLedger {
	return &IdempotencyLedger{entries: map[string]persistence.CommandEntry{}}
}

func (l *IdempotencyLedger) Claim(ctx context.Context, request persistence.ClaimRequest) (persistence.ClaimResult, error) {
	if err := ctx.Err(); err != nil {
		return persistence.ClaimResult{}, err
	}
	if err := request.Validate(); err != nil {
		return persistence.ClaimResult{}, err
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	key := ledgerKey(request.Scope, request.Key)
	entry, exists := l.entries[key]
	if !exists {
		entry = persistence.CommandEntry{
			Scope:       request.Scope,
			Key:         request.Key,
			RequestHash: request.RequestHash,
			Status:      persistence.CommandStatusInProgress,
			CreatedAt:   request.ClaimedAt,
		}
		l.entries[key] = cloneEntry(entry)
		return persistence.ClaimResult{Disposition: persistence.ClaimAcquired, Entry: cloneEntry(entry)}, nil
	}
	if entry.RequestHash != request.RequestHash {
		return persistence.ClaimResult{}, persistence.ErrIdempotencyKeyReuse
	}
	if entry.Status == persistence.CommandStatusCompleted {
		return persistence.ClaimResult{Disposition: persistence.ClaimReplay, Entry: cloneEntry(entry)}, nil
	}
	return persistence.ClaimResult{Disposition: persistence.ClaimInProgress, Entry: cloneEntry(entry)}, nil
}

func (l *IdempotencyLedger) Complete(ctx context.Context, scope, key, requestHash string, response []byte, completedAt time.Time) (persistence.CommandEntry, error) {
	if err := ctx.Err(); err != nil {
		return persistence.CommandEntry{}, err
	}
	if scope == "" || key == "" || requestHash == "" {
		return persistence.CommandEntry{}, errors.New("scope, key and request hash are required")
	}
	if completedAt.IsZero() {
		return persistence.CommandEntry{}, errors.New("completed_at is required")
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	mapKey := ledgerKey(scope, key)
	entry, exists := l.entries[mapKey]
	if !exists {
		return persistence.CommandEntry{}, persistence.ErrCommandNotClaimed
	}
	if entry.RequestHash != requestHash {
		return persistence.CommandEntry{}, persistence.ErrIdempotencyKeyReuse
	}
	if entry.Status == persistence.CommandStatusCompleted {
		return cloneEntry(entry), nil
	}
	entry.Status = persistence.CommandStatusCompleted
	entry.Response = append([]byte(nil), response...)
	completed := completedAt
	entry.CompletedAt = &completed
	l.entries[mapKey] = cloneEntry(entry)
	return cloneEntry(entry), nil
}

func ledgerKey(scope, key string) string { return scope + "\x00" + key }

func cloneEntry(entry persistence.CommandEntry) persistence.CommandEntry {
	entry.Response = append([]byte(nil), entry.Response...)
	if entry.CompletedAt != nil {
		completed := *entry.CompletedAt
		entry.CompletedAt = &completed
	}
	return entry
}
