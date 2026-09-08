package postgres

import (
	"database/sql"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/application"
	inventory "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/domain"
	sharedpostgres "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/infra/postgres"
)

type Repository struct {
	*sharedpostgres.JSONStore[inventory.Inventory]
}

func NewRepository(db *sql.DB) (*Repository, error) {
	store, err := sharedpostgres.NewJSONStore(db, "inventories", func(value inventory.Inventory) string { return value.CharacterID })
	if err != nil {
		return nil, err
	}
	return &Repository{JSONStore: store}, nil
}

var _ application.Repository = (*Repository)(nil)
