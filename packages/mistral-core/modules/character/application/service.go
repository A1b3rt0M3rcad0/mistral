package application

import (
	"fmt"
	"sort"

	character "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/character/domain"
	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
)

type Service struct {
	registry content.Registry
}

func NewService(registry content.Registry) Service {
	return Service{registry: registry}
}

func (s Service) Create(characterID, raceID string) (character.Character, error) {
	if _, ok := s.registry.Races[raceID]; !ok {
		return character.Character{}, fmt.Errorf("unknown race %q", raceID)
	}
	dungeonIDs := make([]string, 0, len(s.registry.Dungeons))
	for dungeonID := range s.registry.Dungeons {
		dungeonIDs = append(dungeonIDs, dungeonID)
	}
	sort.Strings(dungeonIDs)
	return character.New(characterID, raceID, dungeonIDs)
}
