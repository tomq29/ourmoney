package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/tomq29/ourmoney/internal/domain/common"
	"github.com/tomq29/ourmoney/internal/domain/transaction"
)

type TransactionRepo struct {
	pool *pgxpool.Pool
}

func NewTransactionRepo(pool *pgxpool.Pool) *TransactionRepo {
	return &TransactionRepo{pool: pool}
}

func (r *TransactionRepo) Create(ctx context.Context, t *transaction.Transaction) error {
	if _, err := db(ctx, r.pool).Exec(ctx,
		`INSERT INTO transactions
		     (id, ledger_id, created_by, transaction_type, amount, currency,
		      from_account_id, to_account_id, category_id, comment, date)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		t.ID(), t.LedgerID(), t.CreatedBy(),
		transaction.TransactionTypeName[t.Type()],
		t.Amount().Amount(), t.Amount().CurrencyCode(),
		nullableString(t.FromAccountID()), nullableString(t.ToAccountID()),
		nullableString(t.CategoryID()), t.Comment(), t.Date(),
	); err != nil {
		return fmt.Errorf("TransactionRepo, Create: %w", err)
	}
	return nil
}

func (r *TransactionRepo) FindByID(ctx context.Context, id string) (*transaction.Transaction, error) {
	row := db(ctx, r.pool).QueryRow(ctx, selectTransaction+` WHERE id = $1`, id)
	return scanTransaction(row, "FindByID")
}

// Last is what /undo works on: the newest entry in the ledger, whoever made it.
func (r *TransactionRepo) Last(ctx context.Context, ledgerID string) (*transaction.Transaction, error) {
	row := db(ctx, r.pool).QueryRow(ctx,
		selectTransaction+` WHERE ledger_id = $1 ORDER BY date DESC, id DESC LIMIT 1`, ledgerID)
	return scanTransaction(row, "Last")
}

func (r *TransactionRepo) Find(ctx context.Context, f transaction.Filter) ([]*transaction.Transaction, error) {
	where := []string{"ledger_id = $1"}
	args := []any{f.LedgerID}

	add := func(clause string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(clause, len(args)))
	}

	if f.AccountID != "" {
		args = append(args, f.AccountID)
		where = append(where, fmt.Sprintf("(from_account_id = $%d OR to_account_id = $%d)", len(args), len(args)))
	}
	if f.CategoryID != "" {
		add("category_id = $%d", f.CategoryID)
	}
	if f.Type != nil {
		add("transaction_type = $%d", transaction.TransactionTypeName[*f.Type])
	}
	if !f.From.IsZero() {
		add("date >= $%d", f.From)
	}
	if !f.To.IsZero() {
		add("date < $%d", f.To)
	}

	query := selectTransaction + " WHERE " + strings.Join(where, " AND ") + " ORDER BY date DESC, id DESC"
	if f.Limit > 0 {
		args = append(args, f.Limit)
		query += fmt.Sprintf(" LIMIT $%d", len(args))
	}
	if f.Offset > 0 {
		args = append(args, f.Offset)
		query += fmt.Sprintf(" OFFSET $%d", len(args))
	}

	rows, err := db(ctx, r.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("TransactionRepo, Find: %w", err)
	}
	defer rows.Close()

	var found []*transaction.Transaction
	for rows.Next() {
		t, err := scanTransaction(rows, "Find")
		if err != nil {
			return nil, err
		}
		found = append(found, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("TransactionRepo, Find: %w", err)
	}
	return found, nil
}

func (r *TransactionRepo) Delete(ctx context.Context, id string) error {
	tag, err := db(ctx, r.pool).Exec(ctx, `DELETE FROM transactions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("TransactionRepo, Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("TransactionRepo, Delete: %w", common.ErrNotFound)
	}
	return nil
}

const selectTransaction = `SELECT id, ledger_id, created_by, transaction_type, amount, currency,
	       from_account_id, to_account_id, category_id, comment, date
	  FROM transactions`

func scanTransaction(row commonRow, method string) (*transaction.Transaction, error) {
	var (
		id, ledgerID, kind, currencyCode string
		createdBy                        int64
		amount                           decimal.Decimal
		fromAccountID, toAccountID       *string
		categoryID, comment              *string
		date                             time.Time
	)

	err := row.Scan(&id, &ledgerID, &createdBy, &kind, &amount, &currencyCode,
		&fromAccountID, &toAccountID, &categoryID, &comment, &date)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, fmt.Errorf("TransactionRepo, %v: %w", method, common.ErrNotFound)
	case err != nil:
		return nil, fmt.Errorf("TransactionRepo, %v: %w", method, err)
	}

	currency, ok := common.CurrencyConverterFromString[currencyCode]
	if !ok {
		return nil, fmt.Errorf("TransactionRepo, %v: unknown currency %q", method, currencyCode)
	}
	money, err := common.NewMoney(amount, currency)
	if err != nil {
		return nil, fmt.Errorf("TransactionRepo, %v: %w", method, err)
	}

	return transaction.RestoreTransaction(transaction.RestoreParams{
		ID:            id,
		LedgerID:      ledgerID,
		CreatedBy:     createdBy,
		Type:          transaction.TypeFromString(kind),
		Amount:        money,
		FromAccountID: derefString(fromAccountID),
		ToAccountID:   derefString(toAccountID),
		CategoryID:    derefString(categoryID),
		Comment:       derefString(comment),
		Date:          date,
	}), nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
