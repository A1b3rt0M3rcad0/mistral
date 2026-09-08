package application

import (
	"errors"
	"fmt"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
)

var (
	ErrReleaseNotFound      = errors.New("content release not found")
	ErrReleaseAlreadyExists = errors.New("content release already exists")
)

type ReleaseCatalog interface {
	Active() (domain.Registry, error)
	Resolve(string) (domain.Registry, error)
}

type Catalog struct {
	activeID string
	releases map[string]domain.Registry
}

func NewCatalog(active domain.Registry) *Catalog {
	catalog := &Catalog{activeID: active.Manifest.ReleaseID(), releases: map[string]domain.Registry{}}
	if catalog.activeID != "" {
		catalog.releases[catalog.activeID] = active
	}
	return catalog
}

func (c *Catalog) Active() (domain.Registry, error) {
	if c == nil || c.activeID == "" {
		return domain.Registry{}, fmt.Errorf("%w: active release is not configured", ErrReleaseNotFound)
	}
	return c.Resolve(c.activeID)
}

func (c *Catalog) Resolve(releaseID string) (domain.Registry, error) {
	if c == nil || releaseID == "" {
		return domain.Registry{}, fmt.Errorf("%w: %q", ErrReleaseNotFound, releaseID)
	}
	registry, ok := c.releases[releaseID]
	if !ok {
		return domain.Registry{}, fmt.Errorf("%w: %s", ErrReleaseNotFound, releaseID)
	}
	return registry, nil
}

func (c *Catalog) Add(registry domain.Registry) error {
	if c == nil {
		return errors.New("content release catalog is required")
	}
	releaseID := registry.Manifest.ReleaseID()
	if releaseID == "" {
		return errors.New("content release id is required")
	}
	if _, exists := c.releases[releaseID]; exists {
		return fmt.Errorf("%w: %s", ErrReleaseAlreadyExists, releaseID)
	}
	c.releases[releaseID] = registry
	return nil
}
