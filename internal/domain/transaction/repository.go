package transaction

import (
	"context"
	"time"
)

type Filter struct {
	LedgerID   string
	AccountID  string
	CategoryID string
	Type       *TransactionType
	From       time.Time
	To         time.Time
	Limit      int
	Offset     int
}

type TransactionRepository interface {
	Create(ctx context.Context, t *Transaction) error
	FindByID(ctx context.Context, id string) (*Transaction, error)
	Find(ctx context.Context, f Filter) ([]*Transaction, error)
	Last(ctx context.Context, ledgerID string) (*Transaction, error)
	Delete(ctx context.Context, id string) error
}
