package domain

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"time"

	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/determinism"
)

type Engine struct{}

func NewEngine() Engine { return Engine{} }

func (Engine) Resolve(session Session, definition content.GatheringDefinition, now time.Time) (Resolution, error) {
	version, err := determinism.Canonical(session.RulesetVersion)
	if err != nil {
		return Resolution{}, err
	}
	switch version {
	case determinism.RulesV1:
		return resolveV1(session, definition, now)
	default:
		return Resolution{}, fmt.Errorf("unsupported gathering ruleset version %q", version)
	}
}

func resolveV1(session Session, definition content.GatheringDefinition, now time.Time) (Resolution, error) {
	if session.ID == "" || session.CharacterID == "" {
		return Resolution{}, errors.New("session id and character id are required")
	}
	if session.ContentRelease == "" {
		return Resolution{}, errors.New("content release is required")
	}
	if session.GatheringID == "" || session.GatheringID != definition.ID {
		return Resolution{}, errors.New("gathering definition does not match session")
	}
	if definition.IntervalSeconds <= 0 {
		return Resolution{}, errors.New("gathering interval must be positive")
	}
	if now.Before(session.StartedAt) {
		return Resolution{}, errors.New("resolved time cannot be before started_at")
	}
	if session.ClaimedCycles < 0 {
		return Resolution{}, errors.New("claimed cycles cannot be negative")
	}

	interval := time.Duration(definition.IntervalSeconds) * time.Second
	totalCycles := int(now.Sub(session.StartedAt) / interval)
	if totalCycles <= session.ClaimedCycles {
		return Resolution{
			SessionID:    session.ID,
			ResolvedAt:   now,
			ThroughCycle: session.ClaimedCycles,
			Rewards:      []Reward{},
			Batches:      []RewardBatch{},
		}, nil
	}

	rewards := map[string]int{}
	batches := []RewardBatch{}
	for ordinal := session.ClaimedCycles + 1; ordinal <= totalCycles; ordinal++ {
		rng := rand.New(rand.NewSource(session.Seed + int64(ordinal)*7919)) // #nosec G404 -- ruleset v1 intentionally preserves deterministic gameplay RNG.
		acquiredAt := session.StartedAt.Add(time.Duration(ordinal) * interval)
		for _, drop := range definition.Drops {
			if drop.Probability <= 0 || drop.Probability > 1 {
				return Resolution{}, fmt.Errorf("invalid drop probability %.4f for item %s", drop.Probability, drop.ItemID)
			}
			if drop.MinQuantity <= 0 || drop.MaxQuantity < drop.MinQuantity {
				return Resolution{}, fmt.Errorf("invalid quantity range for item %s", drop.ItemID)
			}
			if rng.Float64() > drop.Probability {
				continue
			}
			quantity := drop.MinQuantity
			if drop.MaxQuantity > drop.MinQuantity {
				quantity += rng.Intn(drop.MaxQuantity - drop.MinQuantity + 1)
			}
			rewards[drop.ItemID] += quantity
			batches = append(batches, RewardBatch{Cycle: ordinal, AcquiredAt: acquiredAt, ItemID: drop.ItemID, Quantity: quantity})
		}
	}

	itemIDs := make([]string, 0, len(rewards))
	for itemID := range rewards {
		itemIDs = append(itemIDs, itemID)
	}
	sort.Strings(itemIDs)
	items := make([]Reward, 0, len(itemIDs))
	for _, itemID := range itemIDs {
		items = append(items, Reward{ItemID: itemID, Quantity: rewards[itemID]})
	}

	return Resolution{
		SessionID:    session.ID,
		ResolvedAt:   now,
		FromCycle:    session.ClaimedCycles + 1,
		ThroughCycle: totalCycles,
		Rewards:      items,
		Batches:      batches,
	}, nil
}
