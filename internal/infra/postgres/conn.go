package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	pgxdecimal "github.com/jackc/pgx-shopspring-decimal"
)

func NewPool(ctx context.Context, connString string) (*pgxpool.Pool, error) {

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("NewPool parse config: %w", err)
	}

	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		pgxdecimal.Register(conn.TypeMap())
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("NewPool connect: %w", err)
	}

	return pool, nil
}
