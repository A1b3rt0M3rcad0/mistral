package postgres

import (
	"database/sql"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/character/application"
	character "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/character/domain"
	sharedpostgres "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/infra/postgres"
)

type Repository struct {
	*sharedpostgres.JSONStore[character.Character]
}

func NewRepository(db *sql.DB) (*Repository, error) {
	store, err := sharedpostgres.NewJSONStore(db, "characters", func(value character.Character) string { return value.ID })
	if err != nil {
		return nil, err
	}
	return &Repository{JSONStore: store}, nil
}

var _ application.Repository = (*Repository)(nil)
