package application

import (
	"context"
	"errors"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"
)

var ErrReleaseIntegrity = errors.New("content release integrity violation")

type ReleaseArchive interface {
	Archive(context.Context, domain.Registry) error
	List(context.Context) ([]domain.Registry, error)
}
