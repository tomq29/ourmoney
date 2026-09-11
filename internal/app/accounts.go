package app

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/tomq29/ourmoney/internal/domain/account"
	"github.com/tomq29/ourmoney/internal/domain/common"
	"github.com/tomq29/ourmoney/internal/domain/ledger"
)

type CreateAccountParams struct {
	TelegramID     int64
	Name           string
	Type           account.AccountType
	InitialBalance decimal.Decimal
}

func (s *Service) CreateAccount(ctx context.Context, p CreateAccountParams) (*account.Account, error) {
	l, err := s.ledgerOf(ctx, p.TelegramID)
	if err != nil {
		return nil, err
	}

	balance, err := s.moneyOf(l, p.InitialBalance)
	if err != nil {
		return nil, err
	}

	acc, err := account.NewAccount(s.newID(), l.ID(), p.Name, balance, p.Type)
	if err != nil {
		return nil, err
	}
	if err := s.accounts.Create(ctx, acc); err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}
	return acc, nil
}

func (s *Service) ListAccounts(ctx context.Context, telegramID int64) ([]*account.Account, error) {
	l, err := s.ledgerOf(ctx, telegramID)
	if err != nil {
		return nil, err
	}

	accounts, err := s.accounts.FindByLedgerID(ctx, l.ID())
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	return accounts, nil
}

func (s *Service) ArchiveAccount(ctx context.Context, telegramID int64, accountID string) error {
	return s.tx.Do(ctx, func(ctx context.Context) error {
		l, err := s.ledgerOf(ctx, telegramID)
		if err != nil {
			return err
		}

		acc, err := s.accountOf(ctx, l, accountID)
		if err != nil {
			return err
		}

		acc.Archive()
		return s.accounts.Update(ctx, acc)
	})
}

// Total adds up what the household has, which only means anything while every
// account is in the ledger's currency.
func (s *Service) Total(ctx context.Context, telegramID int64) (common.Money, error) {
	l, err := s.ledgerOf(ctx, telegramID)
	if err != nil {
		return common.Money{}, err
	}

	accounts, err := s.accounts.FindByLedgerID(ctx, l.ID())
	if err != nil {
		return common.Money{}, fmt.Errorf("list accounts: %w", err)
	}

	total, err := s.moneyOf(l, decimal.Zero)
	if err != nil {
		return common.Money{}, err
	}

	for _, acc := range accounts {
		if acc.IsArchived() {
			continue
		}
		total, err = total.Add(acc.Balance())
		if err != nil {
			return common.Money{}, ErrDifferentMoney
		}
	}
	return total, nil
}

// moneyOf builds an amount in the ledger's own currency, which is the only
// currency anything inside it is ever expressed in.
func (s *Service) moneyOf(l *ledger.Ledger, amount decimal.Decimal) (common.Money, error) {
	state, ok := common.CurrencyConverterFromString[l.Currency().String()]
	if !ok {
		return common.Money{}, fmt.Errorf("unknown currency %q", l.Currency().String())
	}
	return common.NewMoney(amount, state)
}
