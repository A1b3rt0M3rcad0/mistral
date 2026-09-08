package gameplaydb

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestRequiredMigrationTracksLatestUpMigration(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", "..", "..", "migrations"))
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	var migrations []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		migrations = append(migrations, entry.Name())
	}
	if len(migrations) == 0 {
		t.Fatal("no up migrations found")
	}
	sort.Strings(migrations)
	latest := migrations[len(migrations)-1]
	if RequiredMigration != latest {
		t.Fatalf("RequiredMigration = %q, latest migration = %q", RequiredMigration, latest)
	}

	payload, err := os.ReadFile(filepath.Join(root, latest))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(payload)
	checksum := hex.EncodeToString(sum[:])
	if RequiredMigrationChecksum != checksum {
		t.Fatalf("RequiredMigrationChecksum = %q, latest migration checksum = %q", RequiredMigrationChecksum, checksum)
	}
}
