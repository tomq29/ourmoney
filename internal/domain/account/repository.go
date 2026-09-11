package account

import "context"

type AccountRepository interface {
	Create(ctx context.Context, acc *Account) error
	FindByID(ctx context.Context, id string) (*Account, error)
	LockByID(ctx context.Context, id string) (*Account, error)
	FindByLedgerID(ctx context.Context, ledgerID string) ([]*Account, error)
	Update(ctx context.Context, acc *Account) error
}
