package account

import "context"

type AccountRepository interface {
	Create(ctx context.Context, acc *Account) error
	FindByID(ctx context.Context, id string) (*Account, error)
	FindByUserID(ctx context.Context, userID string) ([]*Account, error)
	Update(ctx context.Context, acc *Account) error
}
