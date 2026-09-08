package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/lib/pq"
)

func OpenPostgres(ctx context.Context, dsn string) (*sql.DB, error) {
	if dsn == "" {
		return nil, errors.New("postgres database URL is required")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return db, nil
}
