package domain

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/determinism"
)

type Engine struct{}

func NewEngine() Engine { return Engine{} }

func (Engine) Resolve(run Run, tier content.DungeonTierDefinition, now time.Time) (Resolution, error) {
	version, err := determinism.Canonical(run.RulesetVersion)
	if err != nil {
		return Resolution{}, err
	}
	switch version {
	case determinism.RulesV1:
		return resolveV1(run, tier, now)
	default:
		return Resolution{}, fmt.Errorf("unsupported dungeon ruleset version %q", version)
	}
}

func resolveV1(run Run, tier content.DungeonTierDefinition, now time.Time) (Resolution, error) {
	if run.ID == "" {
		return Resolution{}, errors.New("run id is required")
	}
	if run.ContentRelease == "" {
		return Resolution{}, errors.New("content release is required")
	}
	if run.DungeonID == "" || run.DungeonTier <= 0 {
		return Resolution{}, errors.New("dungeon id and tier are required")
	}
	if run.PartySize < 1 || run.PartySize > 4 {
		return Resolution{}, fmt.Errorf("party size %d is outside MVP bounds 1..4", run.PartySize)
	}
	if tier.Tier != run.DungeonTier {
		return Resolution{}, fmt.Errorf("run tier %d does not match definition tier %d", run.DungeonTier, tier.Tier)
	}
	if tier.EncounterIntervalSeconds <= 0 {
		return Resolution{}, errors.New("encounter interval must be positive")
	}
	if now.Before(run.StartedAt) {
		return Resolution{}, errors.New("resolved time cannot be before started_at")
	}

	interval := time.Duration(tier.EncounterIntervalSeconds) * time.Second
	count := int(now.Sub(run.StartedAt) / interval)
	encounters := make([]Encounter, 0, count)
	rng := rand.New(rand.NewSource(run.Seed)) // #nosec G404 -- ruleset v1 intentionally preserves deterministic gameplay RNG.

	for ordinal := 1; ordinal <= count; ordinal++ {
		monsterID, err := chooseMonster(rng, tier.MonsterPool)
		if err != nil {
			return Resolution{}, err
		}
		encounters = append(encounters, Encounter{
			Ordinal:   ordinal,
			DueAt:     run.StartedAt.Add(time.Duration(ordinal) * interval),
			MonsterID: monsterID,
		})
	}

	return Resolution{RunID: run.ID, ResolvedAt: now, Encounters: encounters}, nil
}

func chooseMonster(rng *rand.Rand, pool []content.WeightedMonster) (string, error) {
	total := 0
	for _, monster := range pool {
		if monster.Weight <= 0 {
			return "", fmt.Errorf("monster %s has non-positive weight", monster.MonsterID)
		}
		total += monster.Weight
	}
	if total <= 0 {
		return "", errors.New("monster pool is empty")
	}

	roll := rng.Intn(total)
	cursor := 0
	for _, monster := range pool {
		cursor += monster.Weight
		if roll < cursor {
			return monster.MonsterID, nil
		}
	}
	return "", errors.New("monster selection failed")
}
