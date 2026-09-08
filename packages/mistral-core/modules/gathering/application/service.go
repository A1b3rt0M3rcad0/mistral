package application

import (
	"errors"
	"fmt"
	"time"

	contentapplication "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/application"
	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	decay "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/decay/domain"
	gathering "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/gathering/domain"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/determinism"
)

type Service struct {
	catalog contentapplication.ReleaseCatalog
	engine  gathering.Engine
	decay   decay.Engine
}

func NewService(registry content.Registry) Service {
	return NewServiceWithCatalog(contentapplication.NewCatalog(registry))
}

func NewServiceWithCatalog(catalog contentapplication.ReleaseCatalog) Service {
	return Service{catalog: catalog, engine: gathering.NewEngine(), decay: decay.NewEngine()}
}

func (s Service) Start(sessionID, characterID, gatheringID string, seed int64, startedAt time.Time) (gathering.Session, error) {
	if sessionID == "" || characterID == "" || gatheringID == "" {
		return gathering.Session{}, errors.New("session id, character id and gathering id are required")
	}
	if startedAt.IsZero() {
		return gathering.Session{}, errors.New("started_at is required")
	}
	registry, err := s.activeRegistry()
	if err != nil {
		return gathering.Session{}, err
	}
	if _, ok := registry.Gathering[gatheringID]; !ok {
		return gathering.Session{}, fmt.Errorf("unknown gathering area %q", gatheringID)
	}
	return gathering.Session{
		ID:             sessionID,
		CharacterID:    characterID,
		ContentRelease: registry.Manifest.ReleaseID(),
		RulesetVersion: determinism.Current,
		GatheringID:    gatheringID,
		Seed:           seed,
		StartedAt:      startedAt,
	}, nil
}

func (s Service) Claim(session *gathering.Session, playerInventory *inventory.Inventory, now time.Time) (gathering.Resolution, error) {
	if session == nil || playerInventory == nil {
		return gathering.Resolution{}, errors.New("session and inventory are required")
	}
	if session.CharacterID != playerInventory.CharacterID {
		return gathering.Resolution{}, errors.New("gathering session and inventory belong to different characters")
	}
	registry, err := s.resolveRegistry(session.ContentRelease)
	if err != nil {
		return gathering.Resolution{}, fmt.Errorf("resolve gathering content release %s: %w", session.ContentRelease, err)
	}
	definition, ok := registry.Gathering[session.GatheringID]
	if !ok {
		return gathering.Resolution{}, fmt.Errorf("unknown gathering area %q in release %s", session.GatheringID, session.ContentRelease)
	}

	resolution, err := s.engine.Resolve(*session, definition, now)
	if err != nil {
		return gathering.Resolution{}, err
	}

	working := playerInventory.Clone()
	for _, reward := range resolution.Batches {
		if _, ok := registry.Items[reward.ItemID]; !ok {
			return gathering.Resolution{}, fmt.Errorf("gathering reward references unknown item %s", reward.ItemID)
		}
		expiresAt, err := decay.ExpirationFor(registry.Decay, reward.ItemID, reward.AcquiredAt)
		if err != nil {
			return gathering.Resolution{}, err
		}
		if err := working.Add(reward.ItemID, reward.Quantity, reward.AcquiredAt, expiresAt, map[string]string{"source": "gathering", "activity_id": session.GatheringID}); err != nil {
			return gathering.Resolution{}, err
		}
	}
	working, _, err = s.decay.Resolve(working, registry.Decay, now)
	if err != nil {
		return gathering.Resolution{}, err
	}

	*playerInventory = working
	session.ClaimedCycles = resolution.ThroughCycle
	return resolution, nil
}

func (s Service) activeRegistry() (content.Registry, error) {
	if s.catalog == nil {
		return content.Registry{}, errors.New("content release catalog is required")
	}
	return s.catalog.Active()
}

func (s Service) resolveRegistry(releaseID string) (content.Registry, error) {
	if s.catalog == nil {
		return content.Registry{}, errors.New("content release catalog is required")
	}
	return s.catalog.Resolve(releaseID)
}
