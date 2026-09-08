package entrypoint

import (
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/application"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
)

type Entrypoint struct {
	service application.Service
}

func New(service application.Service) Entrypoint {
	return Entrypoint{service: service}
}

func (e Entrypoint) LoadRelease() (domain.Registry, error) {
	return e.service.LoadValidatedRelease()
}
