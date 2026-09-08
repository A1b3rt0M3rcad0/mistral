package gameplaydb_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/database"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/dbmigrate"
	sharedpostgres "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/infra/postgres"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

func TestPostgresIdempotencyClaimSerializesConcurrentSameKey(t *testing.T) {
	dsn := os.Getenv("MISTRAL_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("MISTRAL_TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	db, err := database.OpenPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	migrationRoot := filepath.Clean(filepath.Join("..", "..", "..", "..", "migrations"))
	if err := dbmigrate.Apply(ctx, db, migrationRoot); err != nil {
		t.Fatal(err)
	}

	const scope = "test.idempotency.concurrent"
	const key = "same-key"
	const requestHash = "same-request"
	if _, err := db.ExecContext(ctx, `DELETE FROM idempotency_commands WHERE scope = $1 AND idempotency_key = $2`, scope, key); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = db.ExecContext(context.Background(), `DELETE FROM idempotency_commands WHERE scope = $1 AND idempotency_key = $2`, scope, key)
	}()

	ledger := sharedpostgres.NewIdempotencyLedger(db)
	transactor := sharedpostgres.NewTransactor(db, nil)
	firstClaimed := make(chan struct{})
	releaseFirst := make(chan struct{})
	firstDone := make(chan error, 1)
	claimedAt := time.Date(2026, 9, 8, 8, 0, 0, 0, time.UTC)

	go func() {
		firstDone <- transactor.WithinTransaction(context.Background(), func(txCtx context.Context) error {
			claim, err := ledger.Claim(txCtx, persistence.ClaimRequest{
				Scope: scope, Key: key, RequestHash: requestHash, ClaimedAt: claimedAt,
			})
			if err != nil {
				return err
			}
			if claim.Disposition != persistence.ClaimAcquired {
				return errors.New("first transaction did not acquire idempotency claim")
			}
			close(firstClaimed)
			<-releaseFirst
			_, err = ledger.Complete(txCtx, scope, key, requestHash, []byte(`{"ok":true}`), claimedAt.Add(time.Second))
			return err
		})
	}()

	select {
	case <-firstClaimed:
	case <-time.After(5 * time.Second):
		t.Fatal("first transaction did not acquire claim")
	}

	blockedCtx, blockedCancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	blockedErr := transactor.WithinTransaction(blockedCtx, func(txCtx context.Context) error {
		claim, err := ledger.Claim(txCtx, persistence.ClaimRequest{
			Scope: scope, Key: key, RequestHash: requestHash, ClaimedAt: claimedAt.Add(time.Millisecond),
		})
		if err != nil {
			return err
		}
		if claim.Disposition == persistence.ClaimAcquired {
			return errors.New("second transaction acquired duplicate claim while first was uncommitted")
		}
		return nil
	})
	blockedCancel()
	if blockedErr == nil {
		t.Fatal("second transaction was not blocked by the uncommitted same-key claim")
	}

	close(releaseFirst)
	select {
	case err := <-firstDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("first transaction did not complete")
	}

	var replay persistence.ClaimResult
	if err := transactor.WithinTransaction(context.Background(), func(txCtx context.Context) error {
		var err error
		replay, err = ledger.Claim(txCtx, persistence.ClaimRequest{
			Scope: scope, Key: key, RequestHash: requestHash, ClaimedAt: claimedAt.Add(2 * time.Second),
		})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if replay.Disposition != persistence.ClaimReplay || string(replay.Entry.Response) != `{"ok":true}` {
		t.Fatalf("post-commit claim = %#v, want stored replay", replay)
	}
}

func TestPostgresIdempotencyClaimCanBeReacquiredAfterRollback(t *testing.T) {
	dsn := os.Getenv("MISTRAL_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("MISTRAL_TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	db, err := database.OpenPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	migrationRoot := filepath.Clean(filepath.Join("..", "..", "..", "..", "migrations"))
	if err := dbmigrate.Apply(ctx, db, migrationRoot); err != nil {
		t.Fatal(err)
	}

	const scope = "test.idempotency.rollback"
	const key = "retry-after-rollback"
	const requestHash = "request"
	if _, err := db.ExecContext(ctx, `DELETE FROM idempotency_commands WHERE scope = $1 AND idempotency_key = $2`, scope, key); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = db.ExecContext(context.Background(), `DELETE FROM idempotency_commands WHERE scope = $1 AND idempotency_key = $2`, scope, key)
	}()

	ledger := sharedpostgres.NewIdempotencyLedger(db)
	transactor := sharedpostgres.NewTransactor(db, nil)
	claimedAt := time.Date(2026, 9, 8, 8, 0, 0, 0, time.UTC)
	sentinel := errors.New("abort command")
	firstErr := transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		claim, err := ledger.Claim(txCtx, persistence.ClaimRequest{Scope: scope, Key: key, RequestHash: requestHash, ClaimedAt: claimedAt})
		if err != nil {
			return err
		}
		if claim.Disposition != persistence.ClaimAcquired {
			return errors.New("rollback probe did not acquire claim")
		}
		return sentinel
	})
	if !errors.Is(firstErr, sentinel) {
		t.Fatalf("rollback transaction error = %v, want sentinel", firstErr)
	}

	var retry persistence.ClaimResult
	if err := transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		var err error
		retry, err = ledger.Claim(txCtx, persistence.ClaimRequest{Scope: scope, Key: key, RequestHash: requestHash, ClaimedAt: claimedAt.Add(time.Second)})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if retry.Disposition != persistence.ClaimAcquired {
		t.Fatalf("claim after rollback = %#v, want acquired", retry)
	}
}
