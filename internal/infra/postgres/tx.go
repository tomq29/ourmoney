package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// querier is what a repository needs, and both a pool and a transaction
// satisfy it. That is the whole trick behind the unit of work below.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type txKey struct{}

type TxManager struct {
	pool *pgxpool.Pool
}

func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

// Do runs fn inside one transaction. The transaction travels in the context
// rather than in every signature, so a use case can compose three repositories
// into one atomic change without any of them knowing about the others.
//
// The cost of that is honesty: a method signature no longer says whether the
// call joins a transaction. The alternative -- threading a tx through every
// argument list -- says it, and buries the domain in database types.
func (m *TxManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	// Already inside one: join it. Nesting would need savepoints, and nothing
	// here wants to roll back half of a transfer.
	if _, ok := txFrom(ctx); ok {
		return fn(ctx)
	}

	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func txFrom(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(pgx.Tx)
	return tx, ok
}

func db(ctx context.Context, pool *pgxpool.Pool) querier {
	if tx, ok := txFrom(ctx); ok {
		return tx
	}
	return pool
}
