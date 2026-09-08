package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type PoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxOpenConns:    20,
		MaxIdleConns:    10,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	}
}

func (c PoolConfig) Validate() error {
	if c.MaxOpenConns <= 0 {
		return errors.New("postgres max open connections must be positive")
	}
	if c.MaxIdleConns < 0 {
		return errors.New("postgres max idle connections must not be negative")
	}
	if c.MaxIdleConns > c.MaxOpenConns {
		return errors.New("postgres max idle connections must not exceed max open connections")
	}
	if c.ConnMaxLifetime <= 0 {
		return errors.New("postgres connection max lifetime must be positive")
	}
	if c.ConnMaxIdleTime <= 0 {
		return errors.New("postgres connection max idle time must be positive")
	}
	return nil
}

func OpenPostgres(ctx context.Context, dsn string) (*sql.DB, error) {
	return openPostgres(ctx, dsn, nil)
}

func OpenPostgresWithPool(ctx context.Context, dsn string, config PoolConfig) (*sql.DB, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return openPostgres(ctx, dsn, &config)
}

func openPostgres(ctx context.Context, dsn string, config *PoolConfig) (*sql.DB, error) {
	if dsn == "" {
		return nil, errors.New("postgres database URL is required")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if config != nil {
		db.SetMaxOpenConns(config.MaxOpenConns)
		db.SetMaxIdleConns(config.MaxIdleConns)
		db.SetConnMaxLifetime(config.ConnMaxLifetime)
		db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return db, nil
}
