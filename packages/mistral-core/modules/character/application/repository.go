package application

import (
	"context"

	character "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/character/domain"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

type Repository interface {
	Create(context.Context, character.Character) (persistence.Record[character.Character], error)
	Get(context.Context, string) (persistence.Record[character.Character], error)
	Save(context.Context, character.Character, persistence.Version) (persistence.Record[character.Character], error)
}
