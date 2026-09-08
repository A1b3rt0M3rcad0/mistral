package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync"
	"testing"
)

type transactionState struct {
	mu        sync.Mutex
	begins    int
	commits   int
	rollbacks int
}

func (s *transactionState) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.begins = 0
	s.commits = 0
	s.rollbacks = 0
}

func (s *transactionState) snapshot() (int, int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.begins, s.commits, s.rollbacks
}

type transactionDriver struct{ state *transactionState }

type transactionConn struct{ state *transactionState }

type transactionTx struct{ state *transactionState }

func (d transactionDriver) Open(string) (driver.Conn, error) {
	return &transactionConn{state: d.state}, nil
}

func (c *transactionConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare not supported")
}

func (c *transactionConn) Close() error { return nil }

func (c *transactionConn) Begin() (driver.Tx, error) { return c.begin() }

func (c *transactionConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return c.begin()
}

func (c *transactionConn) begin() (driver.Tx, error) {
	c.state.mu.Lock()
	c.state.begins++
	c.state.mu.Unlock()
	return &transactionTx{state: c.state}, nil
}

func (t *transactionTx) Commit() error {
	t.state.mu.Lock()
	defer t.state.mu.Unlock()
	t.state.commits++
	return nil
}

func (t *transactionTx) Rollback() error {
	t.state.mu.Lock()
	defer t.state.mu.Unlock()
	t.state.rollbacks++
	return nil
}

var (
	registerTransactionDriver sync.Once
	postgresTestState         = &transactionState{}
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	registerTransactionDriver.Do(func() {
		sql.Register("mistral-transaction-test", transactionDriver{state: postgresTestState})
	})
	postgresTestState.reset()
	db, err := sql.Open("mistral-transaction-test", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestTransactorCommitsSuccessfulWork(t *testing.T) {
	db := testDB(t)
	transactor := NewTransactor(db, nil)
	if err := transactor.WithinTransaction(context.Background(), func(ctx context.Context) error {
		if _, ok := ctx.Value(transactionKey{}).(*sql.Tx); !ok {
			t.Fatal("transaction was not bound to context")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	begins, commits, rollbacks := postgresTestState.snapshot()
	if begins != 1 || commits != 1 || rollbacks != 0 {
		t.Fatalf("begin/commit/rollback = %d/%d/%d, want 1/1/0", begins, commits, rollbacks)
	}
}

func TestTransactorRollsBackFailedWork(t *testing.T) {
	db := testDB(t)
	transactor := NewTransactor(db, nil)
	sentinel := errors.New("command failed")
	err := transactor.WithinTransaction(context.Background(), func(context.Context) error { return sentinel })
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
	begins, commits, rollbacks := postgresTestState.snapshot()
	if begins != 1 || commits != 0 || rollbacks != 1 {
		t.Fatalf("begin/commit/rollback = %d/%d/%d, want 1/0/1", begins, commits, rollbacks)
	}
}

func TestNestedTransactorReusesExistingTransaction(t *testing.T) {
	db := testDB(t)
	transactor := NewTransactor(db, nil)
	if err := transactor.WithinTransaction(context.Background(), func(ctx context.Context) error {
		return transactor.WithinTransaction(ctx, func(inner context.Context) error {
			outerTx, _ := ctx.Value(transactionKey{}).(*sql.Tx)
			innerTx, _ := inner.Value(transactionKey{}).(*sql.Tx)
			if outerTx == nil || innerTx != outerTx {
				t.Fatal("nested transaction did not reuse context transaction")
			}
			return nil
		})
	}); err != nil {
		t.Fatal(err)
	}
	begins, commits, rollbacks := postgresTestState.snapshot()
	if begins != 1 || commits != 1 || rollbacks != 0 {
		t.Fatalf("begin/commit/rollback = %d/%d/%d, want 1/1/0", begins, commits, rollbacks)
	}
}
