package entrypoint

import (
	"time"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/dungeon/application"
	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/dungeon/domain"
)

type Entrypoint struct {
	service application.Service
}

func New(service application.Service) Entrypoint {
	return Entrypoint{service: service}
}

func (e Entrypoint) Resolve(run domain.Run, now time.Time) (domain.Resolution, error) {
	return e.service.Resolve(run, now)
}
