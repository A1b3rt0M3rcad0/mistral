package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

type IdempotencyLedger struct {
	db *sql.DB
}

func NewIdempotencyLedger(db *sql.DB) *IdempotencyLedger {
	return &IdempotencyLedger{db: db}
}

func (l *IdempotencyLedger) Claim(ctx context.Context, request persistence.ClaimRequest) (persistence.ClaimResult, error) {
	if l.db == nil {
		return persistence.ClaimResult{}, errors.New("postgres database is required")
	}
	if err := request.Validate(); err != nil {
		return persistence.ClaimResult{}, err
	}
	const insert = `INSERT INTO idempotency_commands (scope, idempotency_key, request_hash, status, created_at) VALUES ($1, $2, $3, 'in_progress', $4) ON CONFLICT (scope, idempotency_key) DO NOTHING RETURNING created_at`
	var createdAt time.Time
	err := runner(ctx, l.db).QueryRowContext(ctx, insert, request.Scope, request.Key, request.RequestHash, request.ClaimedAt).Scan(&createdAt)
	if err == nil {
		entry := persistence.CommandEntry{Scope: request.Scope, Key: request.Key, RequestHash: request.RequestHash, Status: persistence.CommandStatusInProgress, CreatedAt: createdAt}
		return persistence.ClaimResult{Disposition: persistence.ClaimAcquired, Entry: entry}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return persistence.ClaimResult{}, err
	}
	entry, err := l.get(ctx, request.Scope, request.Key)
	if err != nil {
		return persistence.ClaimResult{}, err
	}
	if entry.RequestHash != request.RequestHash {
		return persistence.ClaimResult{}, persistence.ErrIdempotencyKeyReuse
	}
	if entry.Status == persistence.CommandStatusCompleted {
		return persistence.ClaimResult{Disposition: persistence.ClaimReplay, Entry: entry}, nil
	}
	return persistence.ClaimResult{Disposition: persistence.ClaimInProgress, Entry: entry}, nil
}

func (l *IdempotencyLedger) Complete(ctx context.Context, scope, key, requestHash string, response []byte, completedAt time.Time) (persistence.CommandEntry, error) {
	if l.db == nil {
		return persistence.CommandEntry{}, errors.New("postgres database is required")
	}
	if scope == "" || key == "" || requestHash == "" || completedAt.IsZero() {
		return persistence.CommandEntry{}, errors.New("scope, key, request hash and completed_at are required")
	}
	const update = `UPDATE idempotency_commands SET status = 'completed', response = $1, completed_at = $2 WHERE scope = $3 AND idempotency_key = $4 AND request_hash = $5 AND status = 'in_progress' RETURNING created_at`
	var createdAt time.Time
	err := runner(ctx, l.db).QueryRowContext(ctx, update, response, completedAt, scope, key, requestHash).Scan(&createdAt)
	if err == nil {
		completed := completedAt
		return persistence.CommandEntry{Scope: scope, Key: key, RequestHash: requestHash, Status: persistence.CommandStatusCompleted, Response: append([]byte(nil), response...), CreatedAt: createdAt, CompletedAt: &completed}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return persistence.CommandEntry{}, err
	}
	entry, getErr := l.get(ctx, scope, key)
	if getErr != nil {
		if errors.Is(getErr, persistence.ErrNotFound) {
			return persistence.CommandEntry{}, persistence.ErrCommandNotClaimed
		}
		return persistence.CommandEntry{}, getErr
	}
	if entry.RequestHash != requestHash {
		return persistence.CommandEntry{}, persistence.ErrIdempotencyKeyReuse
	}
	if entry.Status == persistence.CommandStatusCompleted {
		return entry, nil
	}
	return persistence.CommandEntry{}, persistence.ErrCommandInProgress
}

func (l *IdempotencyLedger) get(ctx context.Context, scope, key string) (persistence.CommandEntry, error) {
	const query = `SELECT request_hash, status, response, created_at, completed_at FROM idempotency_commands WHERE scope = $1 AND idempotency_key = $2`
	var entry persistence.CommandEntry
	entry.Scope = scope
	entry.Key = key
	var status string
	var response []byte
	var completedAt sql.NullTime
	if err := runner(ctx, l.db).QueryRowContext(ctx, query, scope, key).Scan(&entry.RequestHash, &status, &response, &entry.CreatedAt, &completedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return persistence.CommandEntry{}, persistence.ErrNotFound
		}
		return persistence.CommandEntry{}, err
	}
	entry.Status = persistence.CommandStatus(status)
	entry.Response = append([]byte(nil), response...)
	if completedAt.Valid {
		completed := completedAt.Time
		entry.CompletedAt = &completed
	}
	return entry, nil
}
