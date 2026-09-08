package composition

import (
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/application"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/infra/jsonloader"
)

func NewService(contentRoot string) application.Service {
	return application.NewService(jsonloader.New(contentRoot))
}
