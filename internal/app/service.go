package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/tomq29/ourmoney/internal/domain/account"
	"github.com/tomq29/ourmoney/internal/domain/category"
	"github.com/tomq29/ourmoney/internal/domain/common"
	"github.com/tomq29/ourmoney/internal/domain/ledger"
	"github.com/tomq29/ourmoney/internal/domain/transaction"
)

// TxManager is declared here, by the layer that needs it, and implemented in
// infrastructure. The use cases below never learn what a transaction is made of.
type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type Service struct {
	tx           TxManager
	ledgers      ledger.LedgerRepository
	invites      ledger.InviteRepository
	accounts     account.AccountRepository
	categories   category.CategoryRepository
	transactions transaction.TransactionRepository

	now    func() time.Time
	newID  func() string
	invite func() string
}

type Deps struct {
	Tx           TxManager
	Ledgers      ledger.LedgerRepository
	Invites      ledger.InviteRepository
	Accounts     account.AccountRepository
	Categories   category.CategoryRepository
	Transactions transaction.TransactionRepository

	// Now, NewID and NewInviteCode are injected so tests can pin them; every
	// caller in production leaves them empty and gets the real thing.
	Now           func() time.Time
	NewID         func() string
	NewInviteCode func() string
}

func NewService(d Deps) *Service {
	s := &Service{
		tx:           d.Tx,
		ledgers:      d.Ledgers,
		invites:      d.Invites,
		accounts:     d.Accounts,
		categories:   d.Categories,
		transactions: d.Transactions,
		now:          d.Now,
		newID:        d.NewID,
		invite:       d.NewInviteCode,
	}

	if s.now == nil {
		s.now = func() time.Time { return time.Now().UTC() }
	}
	if s.newID == nil {
		s.newID = newUUID
	}
	if s.invite == nil {
		s.invite = newInviteCode
	}
	return s
}

// ledgerOf is the authorisation check of this service: everything a member may
// touch hangs off the one ledger they belong to, so resolving it is both
// "who are you" and "what may you see".
func (s *Service) ledgerOf(ctx context.Context, telegramID int64) (*ledger.Ledger, error) {
	l, err := s.ledgers.FindByMember(ctx, telegramID)
	if errors.Is(err, common.ErrNotFound) {
		return nil, ErrNoLedger
	}
	if err != nil {
		return nil, fmt.Errorf("resolve ledger: %w", err)
	}
	return l, nil
}

// accountOf locks the account for update and refuses one from another ledger,
// which is the only place a stray id could cross a household boundary.
func (s *Service) accountOf(ctx context.Context, l *ledger.Ledger, accountID string) (*account.Account, error) {
	acc, err := s.accounts.LockByID(ctx, accountID)
	if errors.Is(err, common.ErrNotFound) {
		return nil, ErrAccountNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock account: %w", err)
	}
	if acc.LedgerID() != l.ID() {
		return nil, ErrAccountNotFound
	}
	return acc, nil
}

// categoryOf takes a name as typed and creates it on first use: a person
// entering "-5000 такси" is naming a category, not filling in a form.
func (s *Service) categoryOf(ctx context.Context, l *ledger.Ledger, name string) (*category.Category, error) {
	if name == "" {
		return nil, nil
	}

	found, err := s.categories.FindByName(ctx, l.ID(), name)
	switch {
	case err == nil:
		return found, nil
	case !errors.Is(err, common.ErrNotFound):
		return nil, fmt.Errorf("find category: %w", err)
	}

	created, err := category.NewCategory(s.newID(), l.ID(), name, nil, s.now())
	if err != nil {
		return nil, err
	}
	if err := s.categories.Create(ctx, created); err != nil {
		return nil, fmt.Errorf("create category: %w", err)
	}
	return created, nil
}

func categoryID(c *category.Category) string {
	if c == nil {
		return ""
	}
	return c.ID()
}
