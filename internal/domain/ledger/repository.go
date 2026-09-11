package ledger

import "context"

type LedgerRepository interface {
	Create(ctx context.Context, l *Ledger) error
	FindByID(ctx context.Context, id string) (*Ledger, error)
	FindByMember(ctx context.Context, telegramID int64) (*Ledger, error)
	Save(ctx context.Context, l *Ledger) error
}

type InviteRepository interface {
	Create(ctx context.Context, i *Invite) error
	FindByCode(ctx context.Context, code string) (*Invite, error)
	Save(ctx context.Context, i *Invite) error
}
