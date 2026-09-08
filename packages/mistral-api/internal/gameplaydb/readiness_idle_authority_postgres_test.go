package gameplaydb_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/database"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/dbmigrate"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/gameplaydb"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/determinism"
)

func TestPostgresReadinessRequiresResolvablePinnedIdleAuthority(t *testing.T) {
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
	if _, err := db.ExecContext(ctx, `TRUNCATE TABLE gathering_sessions, dungeon_runs, content_releases`); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = db.ExecContext(context.Background(), `TRUNCATE TABLE gathering_sessions, dungeon_runs, content_releases`)
	}()
	store, err := gameplaydb.New(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Ready(ctx); err != nil {
		t.Fatalf("empty migrated store should be ready: %v", err)
	}

	hash := strings.Repeat("a", 64)
	releaseID := "1@sha256:" + hash
	state := fmt.Sprintf(`{"content_release":%q,"ruleset_version":%q}`, releaseID, determinism.RulesV1)
	if _, err := db.ExecContext(ctx, `INSERT INTO gathering_sessions (id, state) VALUES ('readiness-session', $1::jsonb)`, state); err != nil {
		t.Fatal(err)
	}
	if err := store.Ready(ctx); err == nil {
		t.Fatal("store reported ready while gathering session referenced an unavailable content release")
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO content_releases (release_id, name, version, content_hash, payload_schema_version, payload)
		VALUES ($1, 'mistral', '1', $2, 1, '{}'::jsonb)
	`, releaseID, hash); err != nil {
		t.Fatal(err)
	}
	if err := store.Ready(ctx); err != nil {
		t.Fatalf("store should be ready after pinned content release becomes available: %v", err)
	}

	if _, err := db.ExecContext(ctx, `UPDATE gathering_sessions SET state = jsonb_set(state, '{ruleset_version}', '"mistral.rules.v999"'::jsonb) WHERE id = 'readiness-session'`); err != nil {
		t.Fatal(err)
	}
	if err := store.Ready(ctx); err == nil {
		t.Fatal("store reported ready while persisted IDLE state required an unsupported deterministic ruleset")
	}

	if _, err := db.ExecContext(ctx, `UPDATE gathering_sessions SET state = state - 'ruleset_version' WHERE id = 'readiness-session'`); err != nil {
		t.Fatal(err)
	}
	if err := store.Ready(ctx); err != nil {
		t.Fatalf("legacy empty ruleset should remain resolvable as v1: %v", err)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO dungeon_runs (id, state) VALUES ('readiness-run', '{"ruleset_version":"mistral.rules.v1"}'::jsonb)`); err != nil {
		t.Fatal(err)
	}
	if err := store.Ready(ctx); err == nil {
		t.Fatal("store reported ready while dungeon run had no pinned content release")
	}
}
