package gameplaydb_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/database"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/dbmigrate"
)

func TestPostgresIdempotencyKeyLengthConstraint(t *testing.T) {
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

	const scope = "test.idempotency.bounds"
	defer func() {
		_, _ = db.ExecContext(context.Background(), `DELETE FROM idempotency_commands WHERE scope = $1`, scope)
	}()

	accepted := strings.Repeat("a", 255)
	if _, err := db.ExecContext(ctx, `INSERT INTO idempotency_commands (scope, idempotency_key, request_hash, status, created_at) VALUES ($1, $2, 'hash', 'in_progress', now())`, scope, accepted); err != nil {
		t.Fatalf("255-byte key should be accepted: %v", err)
	}

	tooLong := strings.Repeat("b", 256)
	if _, err := db.ExecContext(ctx, `INSERT INTO idempotency_commands (scope, idempotency_key, request_hash, status, created_at) VALUES ($1, $2, 'hash', 'in_progress', now())`, scope, tooLong); err == nil {
		t.Fatal("256-byte idempotency key bypassed database constraint")
	}
}
