package app

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/shopspring/decimal"

	"github.com/tomq29/ourmoney/internal/domain/common"
	"github.com/tomq29/ourmoney/internal/domain/transaction"
)

type CategoryTotal struct {
	Name  string
	Total common.Money
}

type Report struct {
	From       time.Time
	To         time.Time
	Spent      common.Money
	Earned     common.Money
	ByCategory []CategoryTotal
}

// Report answers the only question a household actually asks of a tracker:
// where did it go. Transfers are left out -- moving money between your own
// pockets is not spending it.
//
// The sum is done here rather than in SQL: a household's month is hundreds of
// rows, and keeping it in Go keeps the repository free of report shapes. A
// ledger with years of history would want the aggregate pushed down.
func (s *Service) Report(ctx context.Context, telegramID int64, from, to time.Time) (Report, error) {
	l, err := s.ledgerOf(ctx, telegramID)
	if err != nil {
		return Report{}, err
	}

	entries, err := s.transactions.Find(ctx, transaction.Filter{
		LedgerID: l.ID(),
		From:     from,
		To:       to,
	})
	if err != nil {
		return Report{}, fmt.Errorf("list transactions: %w", err)
	}

	names, err := s.categoryNames(ctx, l.ID())
	if err != nil {
		return Report{}, err
	}

	zero, err := s.moneyOf(l, decimal.Zero)
	if err != nil {
		return Report{}, err
	}

	report := Report{From: from, To: to, Spent: zero, Earned: zero}
	totals := map[string]common.Money{}

	for _, t := range entries {
		switch t.Type() {
		case transaction.Expense:
			if report.Spent, err = report.Spent.Add(t.Amount()); err != nil {
				return Report{}, ErrDifferentMoney
			}

			name := names[t.CategoryID()]
			if name == "" {
				name = "Без категории"
			}
			running, ok := totals[name]
			if !ok {
				running = zero
			}
			if totals[name], err = running.Add(t.Amount()); err != nil {
				return Report{}, ErrDifferentMoney
			}

		case transaction.Income:
			if report.Earned, err = report.Earned.Add(t.Amount()); err != nil {
				return Report{}, ErrDifferentMoney
			}
		}
	}

	for name, total := range totals {
		report.ByCategory = append(report.ByCategory, CategoryTotal{Name: name, Total: total})
	}
	slices.SortFunc(report.ByCategory, func(a, b CategoryTotal) int {
		if cmp := b.Total.Amount().Cmp(a.Total.Amount()); cmp != 0 {
			return cmp
		}
		return 0
	})

	return report, nil
}

func (s *Service) ListTransactions(ctx context.Context, telegramID int64, limit int) ([]*transaction.Transaction, error) {
	l, err := s.ledgerOf(ctx, telegramID)
	if err != nil {
		return nil, err
	}

	entries, err := s.transactions.Find(ctx, transaction.Filter{LedgerID: l.ID(), Limit: limit})
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	return entries, nil
}

func (s *Service) categoryNames(ctx context.Context, ledgerID string) (map[string]string, error) {
	categories, err := s.categories.FindByLedgerID(ctx, ledgerID)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}

	names := make(map[string]string, len(categories))
	for _, c := range categories {
		names[c.ID()] = c.Name()
	}
	return names, nil
}
