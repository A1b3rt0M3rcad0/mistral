package gameplaydb

import (
	"database/sql"
	"errors"

	characterpostgres "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/character/infra/postgres"
	dungeonpostgres "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/dungeon/infra/postgres"
	gatheringpostgres "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/gathering/infra/postgres"
	inventorypostgres "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/inventory/infra/postgres"
	sharedpostgres "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/infra/postgres"
)

type Store struct {
	DB          *sql.DB
	Characters  *characterpostgres.Repository
	Inventories *inventorypostgres.Repository
	Gathering   *gatheringpostgres.Repository
	DungeonRuns *dungeonpostgres.Repository
	Transactor  sharedpostgres.Transactor
	Idempotency *sharedpostgres.IdempotencyLedger
}

func New(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("database is required")
	}
	characters, err := characterpostgres.NewRepository(db)
	if err != nil {
		return nil, err
	}
	inventories, err := inventorypostgres.NewRepository(db)
	if err != nil {
		return nil, err
	}
	gathering, err := gatheringpostgres.NewRepository(db)
	if err != nil {
		return nil, err
	}
	dungeonRuns, err := dungeonpostgres.NewRepository(db)
	if err != nil {
		return nil, err
	}
	return &Store{
		DB:          db,
		Characters:  characters,
		Inventories: inventories,
		Gathering:   gathering,
		DungeonRuns: dungeonRuns,
		Transactor:  sharedpostgres.NewTransactor(db, nil),
		Idempotency: sharedpostgres.NewIdempotencyLedger(db),
	}, nil
}
