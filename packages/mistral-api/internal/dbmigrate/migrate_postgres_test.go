package dbmigrate_test

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

func TestApplyBackfillsLegacyChecksumAndRejectsMigrationDrift(t *testing.T) {
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

	const migrationName = "900001_checksum_probe.up.sql"
	const probeTable = "mistral_migration_checksum_probe"
	if _, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS `+probeTable); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM schema_migrations WHERE version = $1`, migrationName); err != nil {
		// The meta table may not exist before the first Apply in an isolated database.
	}
	defer func() {
		_, _ = db.ExecContext(context.Background(), `DROP TABLE IF EXISTS `+probeTable)
		_, _ = db.ExecContext(context.Background(), `DELETE FROM schema_migrations WHERE version = $1`, migrationName)
	}()

	root := t.TempDir()
	path := filepath.Join(root, migrationName)
	initial := `CREATE TABLE ` + probeTable + ` (id INTEGER);`
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := dbmigrate.Apply(ctx, db, root); err != nil {
		t.Fatal(err)
	}

	var checksum string
	if err := db.QueryRowContext(ctx, `SELECT checksum FROM schema_migrations WHERE version = $1`, migrationName).Scan(&checksum); err != nil {
		t.Fatal(err)
	}
	if len(checksum) != 64 {
		t.Fatalf("checksum length = %d, want 64", len(checksum))
	}

	if _, err := db.ExecContext(ctx, `UPDATE schema_migrations SET checksum = NULL WHERE version = $1`, migrationName); err != nil {
		t.Fatal(err)
	}
	if err := dbmigrate.Apply(ctx, db, root); err != nil {
		t.Fatalf("legacy checksum backfill failed: %v", err)
	}
	var backfilled string
	if err := db.QueryRowContext(ctx, `SELECT checksum FROM schema_migrations WHERE version = $1`, migrationName).Scan(&backfilled); err != nil {
		t.Fatal(err)
	}
	if backfilled != checksum {
		t.Fatalf("backfilled checksum = %q, want %q", backfilled, checksum)
	}

	modified := `CREATE TABLE ` + probeTable + ` (id BIGINT);`
	if err := os.WriteFile(path, []byte(modified), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := dbmigrate.Apply(ctx, db, root); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected migration checksum mismatch, got %v", err)
	}
}
