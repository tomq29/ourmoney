package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/tomq29/ourmoney/internal/domain/account"
	"github.com/tomq29/ourmoney/internal/domain/common"
	"github.com/tomq29/ourmoney/internal/domain/ledger"
	"github.com/tomq29/ourmoney/internal/domain/transaction"
)

type RecordParams struct {
	TelegramID int64
	AccountID  string
	Amount     decimal.Decimal
	Category   string
	Comment    string
}

// RecordExpense moves the balance and writes the entry in one transaction:
// a balance that disagrees with the history it came from is worse than no
// balance at all.
func (s *Service) RecordExpense(ctx context.Context, p RecordParams) (*transaction.Transaction, error) {
	var recorded *transaction.Transaction

	err := s.tx.Do(ctx, func(ctx context.Context) error {
		l, acc, amount, err := s.prepare(ctx, p)
		if err != nil {
			return err
		}

		cat, err := s.categoryOf(ctx, l, p.Category)
		if err != nil {
			return err
		}

		if err := acc.Withdraw(amount); err != nil {
			return err
		}

		recorded, err = transaction.NewExpense(transaction.NewExpenseParams{
			ID:            s.newID(),
			LedgerID:      l.ID(),
			CreatedBy:     p.TelegramID,
			Amount:        amount,
			FromAccountID: acc.ID(),
			CategoryID:    categoryID(cat),
			Comment:       p.Comment,
			Date:          s.now(),
		})
		if err != nil {
			return err
		}

		if err := s.accounts.Update(ctx, acc); err != nil {
			return err
		}
		return s.transactions.Create(ctx, recorded)
	})

	return recorded, err
}

func (s *Service) RecordIncome(ctx context.Context, p RecordParams) (*transaction.Transaction, error) {
	var recorded *transaction.Transaction

	err := s.tx.Do(ctx, func(ctx context.Context) error {
		l, acc, amount, err := s.prepare(ctx, p)
		if err != nil {
			return err
		}

		cat, err := s.categoryOf(ctx, l, p.Category)
		if err != nil {
			return err
		}

		if err := acc.Deposit(amount); err != nil {
			return err
		}

		recorded, err = transaction.NewIncome(transaction.NewIncomeParams{
			ID:          s.newID(),
			LedgerID:    l.ID(),
			CreatedBy:   p.TelegramID,
			Amount:      amount,
			ToAccountID: acc.ID(),
			CategoryID:  categoryID(cat),
			Comment:     p.Comment,
			Date:        s.now(),
		})
		if err != nil {
			return err
		}

		if err := s.accounts.Update(ctx, acc); err != nil {
			return err
		}
		return s.transactions.Create(ctx, recorded)
	})

	return recorded, err
}

type TransferParams struct {
	TelegramID    int64
	FromAccountID string
	ToAccountID   string
	Amount        decimal.Decimal
	Comment       string
}

func (s *Service) Transfer(ctx context.Context, p TransferParams) (*transaction.Transaction, error) {
	var recorded *transaction.Transaction

	err := s.tx.Do(ctx, func(ctx context.Context) error {
		l, err := s.ledgerOf(ctx, p.TelegramID)
		if err != nil {
			return err
		}

		from, to, err := s.lockPair(ctx, l, p.FromAccountID, p.ToAccountID)
		if err != nil {
			return err
		}

		amount, err := s.moneyOf(l, p.Amount)
		if err != nil {
			return err
		}

		if err := from.Withdraw(amount); err != nil {
			return err
		}
		if err := to.Deposit(amount); err != nil {
			return err
		}

		recorded, err = transaction.NewTransfer(transaction.NewTransferParams{
			ID:            s.newID(),
			LedgerID:      l.ID(),
			CreatedBy:     p.TelegramID,
			Amount:        amount,
			FromAccountID: from.ID(),
			ToAccountID:   to.ID(),
			Comment:       p.Comment,
			Date:          s.now(),
		})
		if err != nil {
			return err
		}

		if err := s.accounts.Update(ctx, from); err != nil {
			return err
		}
		if err := s.accounts.Update(ctx, to); err != nil {
			return err
		}
		return s.transactions.Create(ctx, recorded)
	})

	return recorded, err
}

// UndoLast removes the newest entry and puts the money back where it was. It
// exists because the fastest way to enter an expense is also the fastest way
// to enter the wrong one.
func (s *Service) UndoLast(ctx context.Context, telegramID int64) (*transaction.Transaction, error) {
	var undone *transaction.Transaction

	err := s.tx.Do(ctx, func(ctx context.Context) error {
		l, err := s.ledgerOf(ctx, telegramID)
		if err != nil {
			return err
		}

		last, err := s.transactions.Last(ctx, l.ID())
		if errors.Is(err, common.ErrNotFound) {
			return ErrNothingToUndo
		}
		if err != nil {
			return fmt.Errorf("find last transaction: %w", err)
		}

		if err := s.reverse(ctx, l, last); err != nil {
			return err
		}
		if err := s.transactions.Delete(ctx, last.ID()); err != nil {
			return err
		}

		undone = last
		return nil
	})

	return undone, err
}

// reverse applies the mirror image of a transaction to the balances it moved.
func (s *Service) reverse(ctx context.Context, l *ledger.Ledger, t *transaction.Transaction) error {
	switch t.Type() {
	case transaction.Income:
		acc, err := s.accountOf(ctx, l, t.ToAccountID())
		if err != nil {
			return err
		}
		if err := acc.Withdraw(t.Amount()); err != nil {
			return err
		}
		return s.accounts.Update(ctx, acc)

	case transaction.Expense:
		acc, err := s.accountOf(ctx, l, t.FromAccountID())
		if err != nil {
			return err
		}
		if err := acc.Deposit(t.Amount()); err != nil {
			return err
		}
		return s.accounts.Update(ctx, acc)

	case transaction.Transfer:
		from, to, err := s.lockPair(ctx, l, t.FromAccountID(), t.ToAccountID())
		if err != nil {
			return err
		}
		if err := to.Withdraw(t.Amount()); err != nil {
			return err
		}
		if err := from.Deposit(t.Amount()); err != nil {
			return err
		}
		if err := s.accounts.Update(ctx, from); err != nil {
			return err
		}
		return s.accounts.Update(ctx, to)
	}

	return fmt.Errorf("unknown transaction type %v", t.Type())
}

func (s *Service) prepare(ctx context.Context, p RecordParams) (*ledger.Ledger, *account.Account, common.Money, error) {
	l, err := s.ledgerOf(ctx, p.TelegramID)
	if err != nil {
		return nil, nil, common.Money{}, err
	}

	accountID := p.AccountID
	if accountID == "" {
		accountID, err = s.defaultAccount(ctx, l)
		if err != nil {
			return nil, nil, common.Money{}, err
		}
	}

	acc, err := s.accountOf(ctx, l, accountID)
	if err != nil {
		return nil, nil, common.Money{}, err
	}

	amount, err := s.moneyOf(l, p.Amount)
	if err != nil {
		return nil, nil, common.Money{}, err
	}
	return l, acc, amount, nil
}

// lockPair takes both accounts in a fixed order. Two people transferring in
// opposite directions at the same moment would otherwise each hold the row the
// other is waiting for.
func (s *Service) lockPair(ctx context.Context, l *ledger.Ledger, fromID, toID string) (from, to *account.Account, err error) {
	first, second := fromID, toID
	if first > second {
		first, second = second, first
	}

	firstAcc, err := s.accountOf(ctx, l, first)
	if err != nil {
		return nil, nil, err
	}
	secondAcc, err := s.accountOf(ctx, l, second)
	if err != nil {
		return nil, nil, err
	}

	if first == fromID {
		return firstAcc, secondAcc, nil
	}
	return secondAcc, firstAcc, nil
}

// defaultAccount is the one account a household usually spends from: the only
// live one, or the oldest when there are several.
func (s *Service) defaultAccount(ctx context.Context, l *ledger.Ledger) (string, error) {
	accounts, err := s.accounts.FindByLedgerID(ctx, l.ID())
	if err != nil {
		return "", fmt.Errorf("list accounts: %w", err)
	}

	for _, acc := range accounts {
		if !acc.IsArchived() {
			return acc.ID(), nil
		}
	}
	return "", ErrNoAccounts
}
