package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/tomq29/ourmoney/internal/domain/account"
	"github.com/tomq29/ourmoney/internal/domain/common"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type AccountRepo struct {
	pool *pgxpool.Pool
}

func NewAccountRepo(pool *pgxpool.Pool) *AccountRepo {
	return &AccountRepo{pool: pool}
}

func (r *AccountRepo) Create(ctx context.Context, acc *account.Account) error {
	query := "Insert INTO accounts (id, ledger_id, name, balance_amount, balance_currency, account_type, created_at, updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8) "

	_, err := db(ctx, r.pool).Exec(ctx, query,
		acc.ID(),
		acc.LedgerID(),
		acc.Name(),
		acc.Balance().Amount(),
		acc.Balance().CurrencyCode(),
		account.AccountTypeName[acc.AccountType()],
		acc.CreateAt(),
		acc.UpdateAt())

	if err != nil {
		return fmt.Errorf("AccountRepo, Create: %w", err)
	}

	return nil

}

func (r *AccountRepo) FindByID(ctx context.Context, id string) (*account.Account, error) {
	query := "Select id, ledger_id, name, balance_amount, balance_currency, account_type, is_archived, created_at, updated_at From accounts Where id = $1"

	row := db(ctx, r.pool).QueryRow(ctx, query, id)

	acc, err := converter(row, "FindByID")
	if err != nil {
		return nil, err
	}

	return acc, nil
}

func (r *AccountRepo) FindByLedgerID(ctx context.Context, ledgerID string) ([]*account.Account, error) {

	query := "Select id, ledger_id, name, balance_amount, balance_currency, account_type, is_archived, created_at, updated_at From accounts Where ledger_id = $1"

	rows, err := db(ctx, r.pool).Query(ctx, query, ledgerID)
	if err != nil {
		return nil, fmt.Errorf("AccountRepo, FindByLedgerID: %w", err)
	}
	defer rows.Close()

	var accounts []*account.Account

	for rows.Next() {
		acc, err := converter(rows, "FindByLedgerID")
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, acc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("AccountRepo, FindByLedgerID: %w", err)
	}

	return accounts, nil
}

func (r *AccountRepo) Update(ctx context.Context, acc *account.Account) error {
	query := "Update accounts Set ledger_id = $1, name= $2 , balance_amount= $3, balance_currency= $4, account_type = $5, is_archived = $6, updated_at= $7 WHERE id = $8"

	tag, err := db(ctx, r.pool).Exec(ctx, query,
		acc.LedgerID(),
		acc.Name(),
		acc.Balance().Amount(),
		acc.Balance().CurrencyCode(),
		account.AccountTypeName[acc.AccountType()],
		acc.IsArchived(),
		acc.UpdateAt(),
		acc.ID())

	if err != nil {
		return fmt.Errorf("AccountRepo, Update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		err := errors.New("Zero rows RowsAffected")
		return fmt.Errorf("AccountRepo, Update: %w", err)
	}

	return nil
}

type commonRow interface {
	Scan(dest ...any) error
}

func converter(row commonRow, method string) (*account.Account, error) {
	var id, ledgerID, name, balanceCurrency, accountType string
	var balanceAmount decimal.Decimal
	var isArchived bool
	var createdAt time.Time
	var updatedAt time.Time

	err := row.Scan(&id, &ledgerID, &name, &balanceAmount, &balanceCurrency, &accountType, &isArchived, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("AccountRepo, %v: %w", method, err)
	}

	currency, ok := common.CurrencyConverterFromString[balanceCurrency]
	if !ok {
		err := errors.New("currency dismatch")
		return nil, fmt.Errorf("AccountRepo, %v: %w", method, err)
	}

	money, err := common.NewMoney(balanceAmount, currency)
	if err != nil {
		return nil, fmt.Errorf("AccountRepo, %v: %w", method, err)
	}

	accType, ok := account.AccountTypeConverterFromString[accountType]
	if !ok {
		err := errors.New("AccountType dismatch")
		return nil, fmt.Errorf("AccountRepo, %v: %w", method, err)
	}
	acc := account.RestoreAccount(id, ledgerID, name, money, accType, isArchived, createdAt, updatedAt)

	return acc, nil
}

// LockByID reads an account for update. Two accounts are always locked in a
// fixed order by the caller, so a transfer and its mirror image cannot sit
// waiting for each other.
func (r *AccountRepo) LockByID(ctx context.Context, id string) (*account.Account, error) {
	query := "Select id, ledger_id, name, balance_amount, balance_currency, account_type, is_archived, created_at, updated_at From accounts Where id = $1 For Update"

	return converter(db(ctx, r.pool).QueryRow(ctx, query, id), "LockByID")
}
