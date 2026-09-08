package application

import (
	"fmt"
	"time"

	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	dungeon "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/dungeon/domain"
)

type Service struct {
	registry content.Registry
	engine   dungeon.Engine
}

func NewService(registry content.Registry, engine dungeon.Engine) Service {
	return Service{registry: registry, engine: engine}
}

func (s Service) Resolve(run dungeon.Run, now time.Time) (dungeon.Resolution, error) {
	loadedRelease := s.registry.Manifest.ReleaseID()
	if run.ContentRelease != loadedRelease {
		return dungeon.Resolution{}, fmt.Errorf("run content release %s does not match loaded release %s", run.ContentRelease, loadedRelease)
	}
	_, tier, err := s.registry.DungeonTier(run.DungeonID, run.DungeonTier)
	if err != nil {
		return dungeon.Resolution{}, err
	}
	return s.engine.Resolve(run, tier, now)
}
