package persistence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
)

type CommandStatus string

const (
	CommandStatusInProgress CommandStatus = "in_progress"
	CommandStatusCompleted  CommandStatus = "completed"
)

type ClaimDisposition string

const (
	ClaimAcquired   ClaimDisposition = "acquired"
	ClaimReplay     ClaimDisposition = "replay"
	ClaimInProgress ClaimDisposition = "in_progress"
)

type CommandEntry struct {
	Scope       string        `json:"scope"`
	Key         string        `json:"key"`
	RequestHash string        `json:"request_hash"`
	Status      CommandStatus `json:"status"`
	Response    []byte        `json:"response,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
}

type ClaimRequest struct {
	Scope       string
	Key         string
	RequestHash string
	ClaimedAt   time.Time
}

type ClaimResult struct {
	Disposition ClaimDisposition
	Entry       CommandEntry
}

type IdempotencyLedger interface {
	Claim(context.Context, ClaimRequest) (ClaimResult, error)
	Complete(context.Context, string, string, string, []byte, time.Time) (CommandEntry, error)
}

var (
	ErrIdempotencyKeyReuse = errors.New("idempotency key reused with a different request")
	ErrCommandInProgress   = errors.New("idempotent command is already in progress")
	ErrCommandNotClaimed   = errors.New("idempotent command was not claimed")
)

func (r ClaimRequest) Validate() error {
	if r.Scope == "" {
		return errors.New("idempotency scope is required")
	}
	if r.Key == "" {
		return errors.New("idempotency key is required")
	}
	if r.RequestHash == "" {
		return errors.New("request hash is required")
	}
	if r.ClaimedAt.IsZero() {
		return errors.New("claimed_at is required")
	}
	return nil
}

func HashRequest(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
