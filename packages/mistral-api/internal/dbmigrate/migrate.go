package dbmigrate

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const migrationAdvisoryLock int64 = 73697872616

func Apply(ctx context.Context, db *sql.DB, root string) error {
	if db == nil {
		return errors.New("database is required")
	}
	if root == "" {
		return errors.New("migration root is required")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	files := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)
	if len(files) == 0 {
		return errors.New("no up migrations found")
	}

	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
        version TEXT PRIMARY KEY,
        checksum TEXT,
        applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
    )`); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS checksum TEXT`); err != nil {
		return fmt.Errorf("ensure schema_migrations checksum: %w", err)
	}

	for _, name := range files {
		if err := applyOne(ctx, db, root, name); err != nil {
			return err
		}
	}
	return nil
}

func applyOne(ctx context.Context, db *sql.DB, root, name string) error {
	payload, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		return fmt.Errorf("read migration %s: %w", name, err)
	}
	checksum := migrationChecksum(payload)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", name, err)
	}
	rollback := func(cause error) error {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			return errors.Join(cause, rollbackErr)
		}
		return cause
	}

	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, migrationAdvisoryLock); err != nil {
		return rollback(fmt.Errorf("lock migrations: %w", err))
	}

	var storedChecksum sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT checksum FROM schema_migrations WHERE version = $1`, name).Scan(&storedChecksum)
	switch {
	case err == nil:
		if storedChecksum.Valid {
			if storedChecksum.String != checksum {
				return rollback(fmt.Errorf("migration %s checksum mismatch", name))
			}
		} else {
			if _, err := tx.ExecContext(ctx, `UPDATE schema_migrations SET checksum = $2 WHERE version = $1`, name, checksum); err != nil {
				return rollback(fmt.Errorf("backfill migration %s checksum: %w", name, err))
			}
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("finish existing migration %s: %w", name, err)
		}
		return nil
	case errors.Is(err, sql.ErrNoRows):
		// Migration has not been applied yet.
	default:
		return rollback(fmt.Errorf("check migration %s: %w", name, err))
	}

	if _, err := tx.ExecContext(ctx, string(payload)); err != nil {
		return rollback(fmt.Errorf("apply migration %s: %w", name, err))
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version, checksum) VALUES ($1, $2)`, name, checksum); err != nil {
		return rollback(fmt.Errorf("record migration %s: %w", name, err))
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", name, err)
	}
	return nil
}

func migrationChecksum(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
