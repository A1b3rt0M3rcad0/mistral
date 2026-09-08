package application

import (
	"errors"

	content "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	dungeon "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/dungeon/domain"
)

type Request struct {
	Character dungeon.CharacterSnapshot
	Monster   content.MonsterDefinition
	Seed      int64
	Boss      bool
}

type Outcome struct {
	Victory bool `json:"victory"`
}

type Resolver interface {
	Resolve(Request) (Outcome, error)
}

type DisabledResolver struct{}

func (DisabledResolver) Resolve(Request) (Outcome, error) {
	return Outcome{}, errors.New("combat resolver is not configured")
}
