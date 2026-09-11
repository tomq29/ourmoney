package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tomq29/ourmoney/internal/domain/common"
	"github.com/tomq29/ourmoney/internal/domain/ledger"
)

type InviteRepo struct {
	pool *pgxpool.Pool
}

func NewInviteRepo(pool *pgxpool.Pool) *InviteRepo {
	return &InviteRepo{pool: pool}
}

func (r *InviteRepo) Create(ctx context.Context, i *ledger.Invite) error {
	if _, err := db(ctx, r.pool).Exec(ctx,
		`INSERT INTO invites (code, ledger_id, created_by, created_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		i.Code(), i.LedgerID(), i.CreatedBy(), i.CreatedAt(), i.ExpiresAt(),
	); err != nil {
		return fmt.Errorf("InviteRepo, Create: %w", err)
	}
	return nil
}

func (r *InviteRepo) FindByCode(ctx context.Context, code string) (*ledger.Invite, error) {
	var (
		ledgerID             string
		createdBy            int64
		createdAt, expiresAt time.Time
		usedBy               *int64
		usedAt               *time.Time
	)

	err := db(ctx, r.pool).QueryRow(ctx,
		`SELECT ledger_id, created_by, created_at, expires_at, used_by, used_at
		   FROM invites WHERE code = $1`, code,
	).Scan(&ledgerID, &createdBy, &createdAt, &expiresAt, &usedBy, &usedAt)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, fmt.Errorf("InviteRepo, FindByCode: %w", common.ErrNotFound)
	case err != nil:
		return nil, fmt.Errorf("InviteRepo, FindByCode: %w", err)
	}

	var (
		usedByID int64
		usedTime time.Time
	)
	if usedBy != nil {
		usedByID = *usedBy
	}
	if usedAt != nil {
		usedTime = *usedAt
	}

	return ledger.RestoreInvite(code, ledgerID, createdBy, createdAt, expiresAt, usedByID, usedTime), nil
}

func (r *InviteRepo) Save(ctx context.Context, i *ledger.Invite) error {
	// Claiming an invite is a race between two people with the same code, so
	// the "not used yet" check belongs in the statement, not before it.
	tag, err := db(ctx, r.pool).Exec(ctx,
		`UPDATE invites SET used_by = $1, used_at = $2 WHERE code = $3 AND used_by IS NULL`,
		nullableID(i.UsedBy()), nullableTime(i.UsedAt()), i.Code(),
	)
	if err != nil {
		return fmt.Errorf("InviteRepo, Save: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("InviteRepo, Save: %w", ledger.ErrInviteUsed)
	}
	return nil
}

func nullableID(id int64) *int64 {
	if id == 0 {
		return nil
	}
	return &id
}

func nullableTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
