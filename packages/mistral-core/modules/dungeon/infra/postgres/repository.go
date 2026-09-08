package postgres

import (
	"database/sql"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/dungeon/application"
	dungeon "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/dungeon/domain"
	sharedpostgres "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/infra/postgres"
)

type Repository struct {
	*sharedpostgres.JSONStore[dungeon.Run]
}

func NewRepository(db *sql.DB) (*Repository, error) {
	store, err := sharedpostgres.NewJSONStore(db, "dungeon_runs", func(value dungeon.Run) string { return value.ID })
	if err != nil {
		return nil, err
	}
	return &Repository{JSONStore: store}, nil
}

var _ application.RunRepository = (*Repository)(nil)
