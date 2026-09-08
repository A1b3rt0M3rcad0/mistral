package postgres

import (
	"database/sql"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/gathering/application"
	gathering "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/gathering/domain"
	sharedpostgres "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/infra/postgres"
)

type Repository struct {
	*sharedpostgres.JSONStore[gathering.Session]
}

func NewRepository(db *sql.DB) (*Repository, error) {
	store, err := sharedpostgres.NewJSONStore(db, "gathering_sessions", func(value gathering.Session) string { return value.ID })
	if err != nil {
		return nil, err
	}
	return &Repository{JSONStore: store}, nil
}

var _ application.Repository = (*Repository)(nil)
