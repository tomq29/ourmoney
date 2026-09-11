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

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, acc *account.Account) error {
	query := "Insert INTO accounts (id, user_id, name, balance_amount, balance_currency, account_type) VALUES($1,$2,$3,$4,$5,$6) "

	_, err := r.pool.Exec(ctx, query,
		acc.ID(),
		acc.UserID(),
		acc.Name(),
		acc.Balance().Amount(),
		acc.Balance().CurrencyCode(),
		account.AccountTypeName[acc.AccountType()])

	if err != nil {
		return fmt.Errorf("AccountRepo, Create: %w", err)
	}

	return nil

}

func (r *Repository) FindByID(ctx context.Context, id string) (*account.Account, error) {
	query := "Select id, user_id, name, balance_amount, balance_currency, account_type, is_archived, created_at, updated_at From accounts Where id = $1"

	row := r.pool.QueryRow(ctx, query, id)

	acc, err := converter(row, "FindByID")
	if err != nil {
		return nil, err
	}

	return acc, nil
}

func (r *Repository) FindByUserID(ctx context.Context, userID string) ([]*account.Account, error) {

	query := "Select id, user_id, name, balance_amount, balance_currency, account_type, is_archived, created_at, updated_at From accounts Where user_id = $1"

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("AccountRepo, FindByUserID: %w", err)
	}
	defer rows.Close()

	var accounts []*account.Account

	for rows.Next() {
		acc, err := converter(rows, "FindByUserID")
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, acc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("AccountRepo, FindByUserID: %w", err)
	}

	return accounts, nil
}

func (r *Repository) Update(ctx context.Context, acc *account.Account) error {
	query := "Update accounts Set user_id = $1, name= $2 , balance_amount= $3, balance_currency= $4, account_type = $5, is_archived = $6, updated_at= $7 WHERE id = $8"

	tag, err := r.pool.Exec(ctx, query,
		acc.UserID(),
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
	var id, userID, name, balanceCurrency, accountType string
	var balanceAmount decimal.Decimal
	var isArchived bool
	var createdAt time.Time
	var updatedAt time.Time

	err := row.Scan(&id, &userID, &name, &balanceAmount, &balanceCurrency, &accountType, &isArchived, &createdAt, &updatedAt)
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
	acc := account.RestoreAccount(id, userID, name, money, accType, isArchived, createdAt, updatedAt)

	return acc, nil
}
