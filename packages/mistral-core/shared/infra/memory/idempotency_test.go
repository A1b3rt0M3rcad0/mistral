package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

func TestIdempotencyLedgerReplayAndKeyReuse(t *testing.T) {
	ledger := NewIdempotencyLedger()
	now := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	request := persistence.ClaimRequest{Scope: "gathering.claim", Key: "cmd-1", RequestHash: persistence.HashRequest([]byte("session-1")), ClaimedAt: now}

	first, err := ledger.Claim(context.Background(), request)
	if err != nil || first.Disposition != persistence.ClaimAcquired {
		t.Fatalf("first claim = %#v, %v", first, err)
	}
	pending, err := ledger.Claim(context.Background(), request)
	if err != nil || pending.Disposition != persistence.ClaimInProgress {
		t.Fatalf("pending claim = %#v, %v", pending, err)
	}

	completed, err := ledger.Complete(context.Background(), request.Scope, request.Key, request.RequestHash, []byte(`{"ok":true}`), now.Add(time.Second))
	if err != nil || completed.Status != persistence.CommandStatusCompleted {
		t.Fatalf("complete = %#v, %v", completed, err)
	}
	replay, err := ledger.Claim(context.Background(), request)
	if err != nil || replay.Disposition != persistence.ClaimReplay || string(replay.Entry.Response) != `{"ok":true}` {
		t.Fatalf("replay = %#v, %v", replay, err)
	}

	request.RequestHash = persistence.HashRequest([]byte("different"))
	if _, err := ledger.Claim(context.Background(), request); !errors.Is(err, persistence.ErrIdempotencyKeyReuse) {
		t.Fatalf("expected key reuse error, got %v", err)
	}
}
