package gameplaydb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const RequiredMigration = "000004_content_releases.up.sql"
const RequiredMigrationChecksum = "040a32761865d1fc9cee4a301b944243c9114854ac8183360c1b22b0a4dcf8d2"

var requiredTables = []string{
	"schema_migrations",
	"characters",
	"inventories",
	"gathering_sessions",
	"dungeon_runs",
	"idempotency_commands",
	"character_ownerships",
	"content_releases",
}

func (s *Store) Ready(ctx context.Context) error {
	if s == nil || s.DB == nil {
		return errors.New("gameplay database is not configured")
	}
	if err := s.DB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping gameplay database: %w", err)
	}

	for _, table := range requiredTables {
		var relation sql.NullString
		if err := s.DB.QueryRowContext(ctx, `SELECT to_regclass($1)`, "public."+table).Scan(&relation); err != nil {
			return fmt.Errorf("check required table %s: %w", table, err)
		}
		if !relation.Valid || relation.String == "" {
			return fmt.Errorf("required table %s is missing", table)
		}
	}

	var checksum sql.NullString
	err := s.DB.QueryRowContext(ctx, `SELECT checksum FROM schema_migrations WHERE version = $1`, RequiredMigration).Scan(&checksum)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("required migration %s is not applied", RequiredMigration)
	}
	if err != nil {
		return fmt.Errorf("check required migration %s: %w", RequiredMigration, err)
	}
	if !checksum.Valid || checksum.String != RequiredMigrationChecksum {
		return fmt.Errorf("required migration %s checksum is invalid", RequiredMigration)
	}
	return nil
}
