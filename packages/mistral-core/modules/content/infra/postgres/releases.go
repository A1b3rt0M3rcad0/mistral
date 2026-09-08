package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	contentapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/application"
	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	sharedpostgres "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/infra/postgres"
)

type ReleaseArchive struct {
	db *sql.DB
}

type releasePayload struct {
	Races      map[string]content.RaceDefinition       `json:"races"`
	Items      map[string]content.ItemDefinition       `json:"items"`
	LootTables map[string]content.LootTableDefinition  `json:"loot_tables"`
	Monsters   map[string]content.MonsterDefinition    `json:"monsters"`
	Recipes    map[string]content.RecipeDefinition     `json:"recipes"`
	Dungeons   map[string]content.DungeonDefinition    `json:"dungeons"`
	Gathering  map[string]content.GatheringDefinition  `json:"gathering"`
	Decay      map[string]content.DecayDefinition      `json:"decay"`
}

func NewReleaseArchive(db *sql.DB) (*ReleaseArchive, error) {
	if db == nil {
		return nil, errors.New("database is required")
	}
	return &ReleaseArchive{db: db}, nil
}

func (r *ReleaseArchive) Archive(ctx context.Context, registry content.Registry) error {
	if r == nil || r.db == nil {
		return errors.New("content release archive database is required")
	}
	if err := registry.Validate(); err != nil {
		return fmt.Errorf("validate content release before archive: %w", err)
	}
	payload, err := json.Marshal(payloadFromRegistry(registry))
	if err != nil {
		return fmt.Errorf("marshal content release %s: %w", registry.Manifest.ReleaseID(), err)
	}
	runner := sharedpostgres.Runner(ctx, r.db)
	if _, err := runner.ExecContext(ctx, `
		INSERT INTO content_releases (release_id, name, version, content_hash, payload)
		VALUES ($1, $2, $3, $4, $5::jsonb)
		ON CONFLICT (release_id) DO NOTHING
	`, registry.Manifest.ReleaseID(), registry.Manifest.Name, registry.Manifest.Version, registry.Manifest.Hash, payload); err != nil {
		return fmt.Errorf("archive content release %s: %w", registry.Manifest.ReleaseID(), err)
	}

	stored, err := r.load(ctx, runner, registry.Manifest.ReleaseID())
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(stored, registry) {
		return fmt.Errorf("%w: release %s already exists with different content", contentapplication.ErrReleaseIntegrity, registry.Manifest.ReleaseID())
	}
	return nil
}

func (r *ReleaseArchive) List(ctx context.Context) ([]content.Registry, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("content release archive database is required")
	}
	rows, err := sharedpostgres.Runner(ctx, r.db).QueryContext(ctx, `
		SELECT name, version, content_hash, payload
		FROM content_releases
		ORDER BY archived_at, release_id
	`)
	if err != nil {
		return nil, fmt.Errorf("list content releases: %w", err)
	}
	defer rows.Close()

	releases := make([]content.Registry, 0)
	for rows.Next() {
		registry, err := scanRelease(rows)
		if err != nil {
			return nil, err
		}
		releases = append(releases, registry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate content releases: %w", err)
	}
	return releases, nil
}

func (r *ReleaseArchive) load(ctx context.Context, runner sharedpostgres.DBTX, releaseID string) (content.Registry, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT name, version, content_hash, payload
		FROM content_releases
		WHERE release_id = $1
	`, releaseID)
	registry, err := scanRelease(row)
	if errors.Is(err, sql.ErrNoRows) {
		return content.Registry{}, fmt.Errorf("%w: %s", contentapplication.ErrReleaseNotFound, releaseID)
	}
	return registry, err
}

type releaseScanner interface {
	Scan(...any) error
}

func scanRelease(scanner releaseScanner) (content.Registry, error) {
	var name string
	var version string
	var hash string
	var payload []byte
	if err := scanner.Scan(&name, &version, &hash, &payload); err != nil {
		return content.Registry{}, err
	}
	var stored releasePayload
	if err := json.Unmarshal(payload, &stored); err != nil {
		return content.Registry{}, fmt.Errorf("decode archived content release: %w", err)
	}
	registry := content.NewRegistry()
	registry.Manifest = content.Manifest{Name: name, Version: version, Hash: hash}
	registry.Races = stored.Races
	registry.Items = stored.Items
	registry.LootTables = stored.LootTables
	registry.Monsters = stored.Monsters
	registry.Recipes = stored.Recipes
	registry.Dungeons = stored.Dungeons
	registry.Gathering = stored.Gathering
	registry.Decay = stored.Decay
	if err := registry.Validate(); err != nil {
		return content.Registry{}, fmt.Errorf("validate archived content release %s: %w", registry.Manifest.ReleaseID(), err)
	}
	return registry, nil
}

func payloadFromRegistry(registry content.Registry) releasePayload {
	return releasePayload{
		Races:      registry.Races,
		Items:      registry.Items,
		LootTables: registry.LootTables,
		Monsters:   registry.Monsters,
		Recipes:    registry.Recipes,
		Dungeons:   registry.Dungeons,
		Gathering:  registry.Gathering,
		Decay:      registry.Decay,
	}
}

var _ contentapplication.ReleaseArchive = (*ReleaseArchive)(nil)
