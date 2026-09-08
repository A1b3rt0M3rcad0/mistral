package application

import "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/content/domain"

type ReleaseLoader interface {
	Load() (domain.Registry, error)
}

type Service struct {
	loader ReleaseLoader
}

func NewService(loader ReleaseLoader) Service {
	return Service{loader: loader}
}

func (s Service) LoadValidatedRelease() (domain.Registry, error) {
	registry, err := s.loader.Load()
	if err != nil {
		return domain.Registry{}, err
	}
	if err := registry.Validate(); err != nil {
		return domain.Registry{}, err
	}
	return registry, nil
}
