package gameplaydb_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/database"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/dbmigrate"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/gameplaydb"
)

func TestPostgresReadinessRequiresCurrentSchemaMigration(t *testing.T) {
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
	store, err := gameplaydb.New(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Ready(ctx); err != nil {
		t.Fatalf("migrated store should be ready: %v", err)
	}

	if _, err := db.ExecContext(ctx, `DELETE FROM schema_migrations WHERE version = $1`, gameplaydb.RequiredMigration); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = db.ExecContext(context.Background(), `INSERT INTO schema_migrations (version) VALUES ($1) ON CONFLICT (version) DO NOTHING`, gameplaydb.RequiredMigration)
	}()
	if err := store.Ready(ctx); err == nil {
		t.Fatal("store reported ready while required migration record was missing")
	}
}
