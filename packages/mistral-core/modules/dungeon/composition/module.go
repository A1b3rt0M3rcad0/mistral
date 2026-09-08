package composition

import (
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/dungeon/application"
	dungeon "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/dungeon/domain"
)

func NewService(registry domain.Registry) application.Service {
	return application.NewService(registry, dungeon.NewEngine())
}
