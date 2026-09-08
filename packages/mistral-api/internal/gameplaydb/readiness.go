package gameplaydb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/determinism"
)

const RequiredMigration = "000005_content_release_schema_version.up.sql"
const RequiredMigrationChecksum = "acbccbf1b2937c0f628580d1f156d19fae48b5bc3af06a2c3600a2ec6c349281"

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
	if err := s.checkPinnedContentReferences(ctx); err != nil {
		return err
	}
	if err := s.checkPinnedRulesets(ctx); err != nil {
		return err
	}
	return nil
}

func (s *Store) checkPinnedContentReferences(ctx context.Context) error {
	var source string
	var id string
	var releaseID string
	err := s.DB.QueryRowContext(ctx, `
		SELECT pinned.source, pinned.id, pinned.release_id
		FROM (
			SELECT 'gathering_session' AS source, id, COALESCE(state->>'content_release', '') AS release_id
			FROM gathering_sessions
			UNION ALL
			SELECT 'dungeon_run' AS source, id, COALESCE(state->>'content_release', '') AS release_id
			FROM dungeon_runs
		) AS pinned
		LEFT JOIN content_releases AS releases ON releases.release_id = pinned.release_id
		WHERE pinned.release_id = '' OR releases.release_id IS NULL
		LIMIT 1
	`).Scan(&source, &id, &releaseID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check pinned content release references: %w", err)
	}
	if releaseID == "" {
		return fmt.Errorf("%s %s has no pinned content release", source, id)
	}
	return fmt.Errorf("%s %s references unavailable content release %s", source, id, releaseID)
}

func (s *Store) checkPinnedRulesets(ctx context.Context) error {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT COALESCE(state->>'ruleset_version', '') AS ruleset_version FROM gathering_sessions
		UNION
		SELECT COALESCE(state->>'ruleset_version', '') AS ruleset_version FROM dungeon_runs
	`)
	if err != nil {
		return fmt.Errorf("list pinned deterministic rulesets: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return fmt.Errorf("scan pinned deterministic ruleset: %w", err)
		}
		if _, err := determinism.Canonical(determinism.Version(version)); err != nil {
			return fmt.Errorf("persisted IDLE state uses unsupported deterministic ruleset: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate pinned deterministic rulesets: %w", err)
	}
	return nil
}
