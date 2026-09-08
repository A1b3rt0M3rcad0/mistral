package application

import (
	"errors"
	"fmt"
	"time"

	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	decay "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/decay/domain"
	gathering "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/gathering/domain"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/determinism"
)

type Service struct {
	registry content.Registry
	engine   gathering.Engine
	decay    decay.Engine
}

func NewService(registry content.Registry) Service {
	return Service{registry: registry, engine: gathering.NewEngine(), decay: decay.NewEngine()}
}

func (s Service) Start(sessionID, characterID, gatheringID string, seed int64, startedAt time.Time) (gathering.Session, error) {
	if sessionID == "" || characterID == "" || gatheringID == "" {
		return gathering.Session{}, errors.New("session id, character id and gathering id are required")
	}
	if startedAt.IsZero() {
		return gathering.Session{}, errors.New("started_at is required")
	}
	if _, ok := s.registry.Gathering[gatheringID]; !ok {
		return gathering.Session{}, fmt.Errorf("unknown gathering area %q", gatheringID)
	}
	return gathering.Session{
		ID:             sessionID,
		CharacterID:    characterID,
		ContentRelease: s.registry.Manifest.ReleaseID(),
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
	if session.ContentRelease != s.registry.Manifest.ReleaseID() {
		return gathering.Resolution{}, fmt.Errorf("session content release %s does not match active release %s", session.ContentRelease, s.registry.Manifest.ReleaseID())
	}
	definition, ok := s.registry.Gathering[session.GatheringID]
	if !ok {
		return gathering.Resolution{}, fmt.Errorf("unknown gathering area %q", session.GatheringID)
	}

	resolution, err := s.engine.Resolve(*session, definition, now)
	if err != nil {
		return gathering.Resolution{}, err
	}

	working := playerInventory.Clone()
	for _, reward := range resolution.Batches {
		if _, ok := s.registry.Items[reward.ItemID]; !ok {
			return gathering.Resolution{}, fmt.Errorf("gathering reward references unknown item %s", reward.ItemID)
		}
		expiresAt, err := decay.ExpirationFor(s.registry.Decay, reward.ItemID, reward.AcquiredAt)
		if err != nil {
			return gathering.Resolution{}, err
		}
		if err := working.Add(reward.ItemID, reward.Quantity, reward.AcquiredAt, expiresAt, map[string]string{"source": "gathering", "activity_id": session.GatheringID}); err != nil {
			return gathering.Resolution{}, err
		}
	}
	working, _, err = s.decay.Resolve(working, s.registry.Decay, now)
	if err != nil {
		return gathering.Resolution{}, err
	}

	*playerInventory = working
	session.ClaimedCycles = resolution.ThroughCycle
	return resolution, nil
}
