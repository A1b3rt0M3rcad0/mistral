package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/persistence"
)

var identifierPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

type JSONStore[T any] struct {
	db    *sql.DB
	table string
	id    func(T) string
}

func NewJSONStore[T any](db *sql.DB, table string, id func(T) string) (*JSONStore[T], error) {
	if db == nil {
		return nil, errors.New("postgres database is required")
	}
	if !identifierPattern.MatchString(table) {
		return nil, fmt.Errorf("unsafe postgres table identifier %q", table)
	}
	if id == nil {
		return nil, errors.New("record id function is required")
	}
	return &JSONStore[T]{db: db, table: table, id: id}, nil
}

func (s *JSONStore[T]) Create(ctx context.Context, value T) (persistence.Record[T], error) {
	if err := ctx.Err(); err != nil {
		return persistence.Record[T]{}, err
	}
	id := s.id(value)
	if id == "" {
		return persistence.Record[T]{}, errors.New("record id is required")
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return persistence.Record[T]{}, fmt.Errorf("marshal record: %w", err)
	}
	query := fmt.Sprintf(`INSERT INTO %s (id, version, state) VALUES ($1, 1, $2::jsonb) ON CONFLICT (id) DO NOTHING RETURNING version`, s.table)
	var version int64
	if err := runner(ctx, s.db).QueryRowContext(ctx, query, id, string(payload)).Scan(&version); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return persistence.Record[T]{}, persistence.ErrAlreadyExists
		}
		return persistence.Record[T]{}, err
	}
	return persistence.Record[T]{Value: value, Version: persistence.Version(version)}, nil
}

func (s *JSONStore[T]) Get(ctx context.Context, id string) (persistence.Record[T], error) {
	if err := ctx.Err(); err != nil {
		return persistence.Record[T]{}, err
	}
	if id == "" {
		return persistence.Record[T]{}, errors.New("record id is required")
	}
	query := fmt.Sprintf(`SELECT version, state::text FROM %s WHERE id = $1`, s.table)
	var version int64
	var payload string
	if err := runner(ctx, s.db).QueryRowContext(ctx, query, id).Scan(&version, &payload); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return persistence.Record[T]{}, persistence.ErrNotFound
		}
		return persistence.Record[T]{}, err
	}
	var value T
	if err := json.Unmarshal([]byte(payload), &value); err != nil {
		return persistence.Record[T]{}, fmt.Errorf("unmarshal %s record %s: %w", s.table, id, err)
	}
	return persistence.Record[T]{Value: value, Version: persistence.Version(version)}, nil
}

func (s *JSONStore[T]) Save(ctx context.Context, value T, expected persistence.Version) (persistence.Record[T], error) {
	if err := ctx.Err(); err != nil {
		return persistence.Record[T]{}, err
	}
	id := s.id(value)
	if id == "" {
		return persistence.Record[T]{}, errors.New("record id is required")
	}
	if expected == 0 {
		return persistence.Record[T]{}, errors.New("expected version must be positive")
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return persistence.Record[T]{}, fmt.Errorf("marshal record: %w", err)
	}
	query := fmt.Sprintf(`UPDATE %s SET state = $1::jsonb, version = version + 1, updated_at = now() WHERE id = $2 AND version = $3 RETURNING version`, s.table)
	var version int64
	if err := runner(ctx, s.db).QueryRowContext(ctx, query, string(payload), id, int64(expected)).Scan(&version); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return persistence.Record[T]{}, err
		}
		currentQuery := fmt.Sprintf(`SELECT version FROM %s WHERE id = $1`, s.table)
		var current int64
		if currentErr := runner(ctx, s.db).QueryRowContext(ctx, currentQuery, id).Scan(&current); currentErr != nil {
			if errors.Is(currentErr, sql.ErrNoRows) {
				return persistence.Record[T]{}, persistence.ErrNotFound
			}
			return persistence.Record[T]{}, currentErr
		}
		return persistence.Record[T]{}, persistence.ErrConflict
	}
	return persistence.Record[T]{Value: value, Version: persistence.Version(version)}, nil
}
