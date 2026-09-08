package gameplaydb_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/database"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/dbmigrate"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-api/internal/gameplaydb"
	contentapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/application"
	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
)

func TestPostgresContentReleaseArchiveSurvivesRepositoryRecreation(t *testing.T) {
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
	migrationRoot := filepath.Clean(filepath.Join("..", "..", "..", "..", "migrations"))
	if err := dbmigrate.Apply(ctx, db, migrationRoot); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `TRUNCATE TABLE content_releases`); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	store, err := gameplaydb.New(db)
	if err != nil {
		_ = db.Close()
		t.Fatal(err)
	}

	v1 := minimalContentRelease("1", strings.Repeat("a", 64), "Legacy Human")
	v2 := minimalContentRelease("2", strings.Repeat("b", 64), "Current Human")
	if err := store.ContentReleases.Archive(ctx, v1); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	if err := store.ContentReleases.Archive(ctx, v1); err != nil {
		_ = db.Close()
		t.Fatalf("archiving the same immutable release twice should be idempotent: %v", err)
	}
	if err := store.ContentReleases.Archive(ctx, v2); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	conflict := minimalContentRelease("1", strings.Repeat("a", 64), "Tampered Human")
	if err := store.ContentReleases.Archive(ctx, conflict); !errors.Is(err, contentapplication.ErrReleaseIntegrity) {
		_ = db.Close()
		t.Fatalf("same release id with different payload should fail integrity check, got %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := database.OpenPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reopened.Close() }()
	defer func() { _, _ = reopened.ExecContext(context.Background(), `TRUNCATE TABLE content_releases`) }()
	restartedStore, err := gameplaydb.New(reopened)
	if err != nil {
		t.Fatal(err)
	}
	releases, err := restartedStore.ContentReleases.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(releases) != 2 {
		t.Fatalf("archived releases = %d, want 2", len(releases))
	}

	catalog := contentapplication.NewCatalog(v2)
	for _, archived := range releases {
		if archived.Manifest.ReleaseID() == v2.Manifest.ReleaseID() {
			continue
		}
		if err := catalog.Add(archived); err != nil {
			t.Fatal(err)
		}
	}
	resolvedV1, err := catalog.Resolve(v1.Manifest.ReleaseID())
	if err != nil {
		t.Fatal(err)
	}
	if resolvedV1.Races["human"].Name != "Legacy Human" {
		t.Fatalf("historical release was not restored intact: %#v", resolvedV1.Races["human"])
	}
	if resolvedV1.Manifest.Hash != v1.Manifest.Hash {
		t.Fatalf("historical content hash = %q, want %q", resolvedV1.Manifest.Hash, v1.Manifest.Hash)
	}

	if _, err := reopened.ExecContext(ctx, `UPDATE content_releases SET payload_schema_version = 2 WHERE release_id = $1`, v1.Manifest.ReleaseID()); err != nil {
		t.Fatal(err)
	}
	if _, err := restartedStore.ContentReleases.List(ctx); !errors.Is(err, contentapplication.ErrReleaseIntegrity) {
		t.Fatalf("unsupported archived payload schema should fail closed, got %v", err)
	}
}

func minimalContentRelease(version, hash, raceName string) content.Registry {
	registry := content.NewRegistry()
	registry.Manifest = content.Manifest{Name: "mistral", Version: version, Hash: hash}
	registry.Races["human"] = content.RaceDefinition{ID: "human", Name: raceName}
	return registry
}
