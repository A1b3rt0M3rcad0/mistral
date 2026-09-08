package application

import (
	"context"

	gathering "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/gathering/domain"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

type Repository interface {
	Create(context.Context, gathering.Session) (persistence.Record[gathering.Session], error)
	Get(context.Context, string) (persistence.Record[gathering.Session], error)
	Save(context.Context, gathering.Session, persistence.Version) (persistence.Record[gathering.Session], error)
}
