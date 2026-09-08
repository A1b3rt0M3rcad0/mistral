package application

import (
	"errors"
	"testing"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
)

func TestCatalogResolvesActiveAndHistoricalReleases(t *testing.T) {
	active := domain.NewRegistry()
	active.Manifest = domain.Manifest{Name: "mistral", Version: "2", Hash: "active"}
	historical := domain.NewRegistry()
	historical.Manifest = domain.Manifest{Name: "mistral", Version: "1", Hash: "historical"}

	catalog := NewCatalog(active)
	if err := catalog.Add(historical); err != nil {
		t.Fatal(err)
	}
	resolvedActive, err := catalog.Active()
	if err != nil {
		t.Fatal(err)
	}
	if got := resolvedActive.Manifest.ReleaseID(); got != active.Manifest.ReleaseID() {
		t.Fatalf("active release = %q, want %q", got, active.Manifest.ReleaseID())
	}
	resolvedHistorical, err := catalog.Resolve(historical.Manifest.ReleaseID())
	if err != nil {
		t.Fatal(err)
	}
	if got := resolvedHistorical.Manifest.ReleaseID(); got != historical.Manifest.ReleaseID() {
		t.Fatalf("historical release = %q, want %q", got, historical.Manifest.ReleaseID())
	}
	if _, err := catalog.Resolve("missing@sha256:nope"); !errors.Is(err, ErrReleaseNotFound) {
		t.Fatalf("missing release error = %v", err)
	}
	if err := catalog.Add(historical); !errors.Is(err, ErrReleaseAlreadyExists) {
		t.Fatalf("duplicate release error = %v", err)
	}
}
