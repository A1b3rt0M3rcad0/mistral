package postgres

import (
	"context"
	"database/sql"
	"errors"
)

type DBTX interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type transactionKey struct{}

type Transactor struct {
	db      *sql.DB
	options *sql.TxOptions
}

func NewTransactor(db *sql.DB, options *sql.TxOptions) Transactor {
	return Transactor{db: db, options: options}
}

func (t Transactor) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	if t.db == nil {
		return errors.New("postgres transactor database is required")
	}
	if fn == nil {
		return errors.New("transaction function is required")
	}
	if _, ok := ctx.Value(transactionKey{}).(*sql.Tx); ok {
		return fn(ctx)
	}

	tx, err := t.db.BeginTx(ctx, t.options)
	if err != nil {
		return err
	}
	txCtx := context.WithValue(ctx, transactionKey{}, tx)
	if err := fn(txCtx); err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			return errors.Join(err, rollbackErr)
		}
		return err
	}
	return tx.Commit()
}

func runner(ctx context.Context, db *sql.DB) DBTX {
	if tx, ok := ctx.Value(transactionKey{}).(*sql.Tx); ok {
		return tx
	}
	return db
}
