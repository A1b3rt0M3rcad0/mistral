package database

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestPoolConfigValidation(t *testing.T) {
	valid := DefaultPoolConfig()
	if err := valid.Validate(); err != nil {
		t.Fatalf("default pool config is invalid: %v", err)
	}

	cases := []PoolConfig{
		{MaxOpenConns: 0, MaxIdleConns: 0, ConnMaxLifetime: time.Minute, ConnMaxIdleTime: time.Minute},
		{MaxOpenConns: 1, MaxIdleConns: -1, ConnMaxLifetime: time.Minute, ConnMaxIdleTime: time.Minute},
		{MaxOpenConns: 1, MaxIdleConns: 2, ConnMaxLifetime: time.Minute, ConnMaxIdleTime: time.Minute},
		{MaxOpenConns: 1, MaxIdleConns: 1, ConnMaxLifetime: 0, ConnMaxIdleTime: time.Minute},
		{MaxOpenConns: 1, MaxIdleConns: 1, ConnMaxLifetime: time.Minute, ConnMaxIdleTime: 0},
	}
	for index, config := range cases {
		if err := config.Validate(); err == nil {
			t.Fatalf("case %d unexpectedly accepted invalid config: %#v", index, config)
		}
	}
}

func TestOpenPostgresWithPoolAppliesMaxOpenConnections(t *testing.T) {
	dsn := os.Getenv("MISTRAL_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("MISTRAL_TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	config := DefaultPoolConfig()
	config.MaxOpenConns = 7
	config.MaxIdleConns = 3
	db, err := OpenPostgresWithPool(ctx, dsn, config)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if got := db.Stats().MaxOpenConnections; got != config.MaxOpenConns {
		t.Fatalf("max open connections = %d, want %d", got, config.MaxOpenConns)
	}
}
