package application

import (
	"errors"
	"fmt"
	"time"

	contentapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/application"
	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	dungeon "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/dungeon/domain"
)

type Service struct {
	catalog contentapplication.ReleaseCatalog
	engine  dungeon.Engine
}

func NewService(registry content.Registry, engine dungeon.Engine) Service {
	return NewServiceWithCatalog(contentapplication.NewCatalog(registry), engine)
}

func NewServiceWithCatalog(catalog contentapplication.ReleaseCatalog, engine dungeon.Engine) Service {
	return Service{catalog: catalog, engine: engine}
}

func (s Service) Resolve(run dungeon.Run, now time.Time) (dungeon.Resolution, error) {
	if s.catalog == nil {
		return dungeon.Resolution{}, errors.New("content release catalog is required")
	}
	registry, err := s.catalog.Resolve(run.ContentRelease)
	if err != nil {
		return dungeon.Resolution{}, fmt.Errorf("resolve dungeon content release %s: %w", run.ContentRelease, err)
	}
	_, tier, err := registry.DungeonTier(run.DungeonID, run.DungeonTier)
	if err != nil {
		return dungeon.Resolution{}, err
	}
	return s.engine.Resolve(run, tier, now)
}
