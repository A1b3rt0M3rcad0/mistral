package gameplaydb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const RequiredMigration = "000003_idempotency_bounds.up.sql"
const RequiredMigrationChecksum = "8e84a46c95bbf1bbf168d409c24dd7b65c31492a72b97b2d5e46597414d6e4dd"

var requiredTables = []string{
	"schema_migrations",
	"characters",
	"inventories",
	"gathering_sessions",
	"dungeon_runs",
	"idempotency_commands",
	"character_ownerships",
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
