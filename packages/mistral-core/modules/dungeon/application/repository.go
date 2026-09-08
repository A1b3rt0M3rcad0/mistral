package application

import (
	"context"

	dungeon "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/dungeon/domain"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

type RunRepository interface {
	Create(context.Context, dungeon.Run) (persistence.Record[dungeon.Run], error)
	Get(context.Context, string) (persistence.Record[dungeon.Run], error)
	Save(context.Context, dungeon.Run, persistence.Version) (persistence.Record[dungeon.Run], error)
}
