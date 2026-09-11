package category

import "context"

type CategoryRepository interface {
	Create(ctx context.Context, c *Category) error
	FindByID(ctx context.Context, id string) (*Category, error)
	FindByName(ctx context.Context, ledgerID, name string) (*Category, error)
	FindByLedgerID(ctx context.Context, ledgerID string) ([]*Category, error)
}
