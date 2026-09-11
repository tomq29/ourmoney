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

type LedgerRepo struct {
	pool *pgxpool.Pool
}

func NewLedgerRepo(pool *pgxpool.Pool) *LedgerRepo {
	return &LedgerRepo{pool: pool}
}

func (r *LedgerRepo) Create(ctx context.Context, l *ledger.Ledger) error {
	q := db(ctx, r.pool)

	if _, err := q.Exec(ctx,
		`INSERT INTO ledgers (id, name, currency, created_at) VALUES ($1, $2, $3, $4)`,
		l.ID(), l.Name(), l.Currency().String(), l.CreatedAt(),
	); err != nil {
		return fmt.Errorf("LedgerRepo, Create: %w", err)
	}

	return r.saveMembers(ctx, l)
}

func (r *LedgerRepo) FindByID(ctx context.Context, id string) (*ledger.Ledger, error) {
	q := db(ctx, r.pool)

	var (
		name, currency string
		createdAt      time.Time
	)
	err := q.QueryRow(ctx,
		`SELECT name, currency, created_at FROM ledgers WHERE id = $1`, id,
	).Scan(&name, &currency, &createdAt)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, fmt.Errorf("LedgerRepo, FindByID: %w", common.ErrNotFound)
	case err != nil:
		return nil, fmt.Errorf("LedgerRepo, FindByID: %w", err)
	}

	return r.restore(ctx, id, name, currency, createdAt)
}

func (r *LedgerRepo) FindByMember(ctx context.Context, telegramID int64) (*ledger.Ledger, error) {
	q := db(ctx, r.pool)

	var (
		id, name, currency string
		createdAt          time.Time
	)
	err := q.QueryRow(ctx,
		`SELECT l.id, l.name, l.currency, l.created_at
		   FROM ledgers l JOIN members m ON m.ledger_id = l.id
		  WHERE m.telegram_id = $1`, telegramID,
	).Scan(&id, &name, &currency, &createdAt)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, fmt.Errorf("LedgerRepo, FindByMember: %w", common.ErrNotFound)
	case err != nil:
		return nil, fmt.Errorf("LedgerRepo, FindByMember: %w", err)
	}

	return r.restore(ctx, id, name, currency, createdAt)
}

// Save rewrites the membership wholesale. A ledger holds a household, not a
// crowd, so replacing two rows costs less than working out which one changed.
func (r *LedgerRepo) Save(ctx context.Context, l *ledger.Ledger) error {
	q := db(ctx, r.pool)

	if _, err := q.Exec(ctx,
		`UPDATE ledgers SET name = $1, currency = $2 WHERE id = $3`,
		l.Name(), l.Currency().String(), l.ID(),
	); err != nil {
		return fmt.Errorf("LedgerRepo, Save: %w", err)
	}

	if _, err := q.Exec(ctx, `DELETE FROM members WHERE ledger_id = $1`, l.ID()); err != nil {
		return fmt.Errorf("LedgerRepo, Save: %w", err)
	}

	return r.saveMembers(ctx, l)
}

func (r *LedgerRepo) saveMembers(ctx context.Context, l *ledger.Ledger) error {
	q := db(ctx, r.pool)

	for _, m := range l.Members() {
		if _, err := q.Exec(ctx,
			`INSERT INTO members (ledger_id, telegram_id, name, role, joined_at)
			 VALUES ($1, $2, $3, $4, $5)`,
			l.ID(), m.TelegramID(), m.Name(), string(m.Role()), m.JoinedAt(),
		); err != nil {
			return fmt.Errorf("LedgerRepo, saveMembers: %w", err)
		}
	}
	return nil
}

func (r *LedgerRepo) restore(ctx context.Context, id, name, currencyCode string, createdAt time.Time) (*ledger.Ledger, error) {
	currency, err := currencyFrom(currencyCode)
	if err != nil {
		return nil, fmt.Errorf("LedgerRepo, restore: %w", err)
	}

	rows, err := db(ctx, r.pool).Query(ctx,
		`SELECT telegram_id, name, role, joined_at FROM members WHERE ledger_id = $1 ORDER BY joined_at`, id)
	if err != nil {
		return nil, fmt.Errorf("LedgerRepo, restore: %w", err)
	}
	defer rows.Close()

	var members []ledger.Member
	for rows.Next() {
		var (
			telegramID       int64
			memberName, role string
			joinedAt         time.Time
		)
		if err := rows.Scan(&telegramID, &memberName, &role, &joinedAt); err != nil {
			return nil, fmt.Errorf("LedgerRepo, restore: %w", err)
		}

		m, err := ledger.NewMember(telegramID, memberName, ledger.Role(role), joinedAt)
		if err != nil {
			return nil, fmt.Errorf("LedgerRepo, restore: %w", err)
		}
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("LedgerRepo, restore: %w", err)
	}

	return ledger.RestoreLedger(id, name, currency, members, createdAt), nil
}

func currencyFrom(code string) (common.Currency, error) {
	state, ok := common.CurrencyConverterFromString[code]
	if !ok {
		return common.Currency{}, fmt.Errorf("unknown currency %q", code)
	}
	return common.NewCurrency(state)
}
