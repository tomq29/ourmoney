package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tomq29/ourmoney/internal/domain/category"
	"github.com/tomq29/ourmoney/internal/domain/common"
)

type CategoryRepo struct {
	pool *pgxpool.Pool
}

func NewCategoryRepo(pool *pgxpool.Pool) *CategoryRepo {
	return &CategoryRepo{pool: pool}
}

func (r *CategoryRepo) Create(ctx context.Context, c *category.Category) error {
	if _, err := db(ctx, r.pool).Exec(ctx,
		`INSERT INTO categories (id, ledger_id, name, parent_id, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		c.ID(), c.LedgerID(), c.Name(), nullableString(c.ParentID()), c.CreatedAt(),
	); err != nil {
		return fmt.Errorf("CategoryRepo, Create: %w", err)
	}
	return nil
}

func (r *CategoryRepo) FindByID(ctx context.Context, id string) (*category.Category, error) {
	row := db(ctx, r.pool).QueryRow(ctx,
		`SELECT id, ledger_id, name, parent_id, created_at FROM categories WHERE id = $1`, id)

	return scanCategory(row, "FindByID")
}

// FindByName matches the way people type: "Еда" and "еда" are one category.
func (r *CategoryRepo) FindByName(ctx context.Context, ledgerID, name string) (*category.Category, error) {
	row := db(ctx, r.pool).QueryRow(ctx,
		`SELECT id, ledger_id, name, parent_id, created_at
		   FROM categories WHERE ledger_id = $1 AND lower(name) = lower($2)`,
		ledgerID, name)

	return scanCategory(row, "FindByName")
}

func (r *CategoryRepo) FindByLedgerID(ctx context.Context, ledgerID string) ([]*category.Category, error) {
	rows, err := db(ctx, r.pool).Query(ctx,
		`SELECT id, ledger_id, name, parent_id, created_at
		   FROM categories WHERE ledger_id = $1 ORDER BY name`, ledgerID)
	if err != nil {
		return nil, fmt.Errorf("CategoryRepo, FindByLedgerID: %w", err)
	}
	defer rows.Close()

	var categories []*category.Category
	for rows.Next() {
		c, err := scanCategory(rows, "FindByLedgerID")
		if err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("CategoryRepo, FindByLedgerID: %w", err)
	}
	return categories, nil
}

func scanCategory(row commonRow, method string) (*category.Category, error) {
	var (
		id, ledgerID, name string
		parentID           *string
		createdAt          time.Time
	)

	err := row.Scan(&id, &ledgerID, &name, &parentID, &createdAt)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, fmt.Errorf("CategoryRepo, %v: %w", method, common.ErrNotFound)
	case err != nil:
		return nil, fmt.Errorf("CategoryRepo, %v: %w", method, err)
	}

	parent := ""
	if parentID != nil {
		parent = *parentID
	}
	return category.RestoreCategory(id, ledgerID, name, parent, createdAt), nil
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
