package gameplaydb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const RequiredMigration = "000002_identity.up.sql"

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

	var applied bool
	if err := s.DB.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, RequiredMigration).Scan(&applied); err != nil {
		return fmt.Errorf("check required migration %s: %w", RequiredMigration, err)
	}
	if !applied {
		return fmt.Errorf("required migration %s is not applied", RequiredMigration)
	}
	return nil
}
