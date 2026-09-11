package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/shopspring/decimal"

	"github.com/tomq29/ourmoney/internal/domain/account"
	"github.com/tomq29/ourmoney/internal/domain/common"
	"github.com/tomq29/ourmoney/internal/domain/ledger"
)

const inviteTTL = 24 * time.Hour

type StartParams struct {
	TelegramID int64
	Name       string
	LedgerName string
	Currency   common.CurrencyState
	// Allowed lists the Telegram accounts that may start a ledger at all. It
	// is not authentication -- Telegram already said who is writing -- only a
	// door on a bot that anyone can find and nobody else should be funding.
	Allowed []int64
}

// Start creates a ledger with one cash account already in it, so the first
// thing a new user does can be recording a coffee rather than filling forms.
func (s *Service) Start(ctx context.Context, p StartParams) (*ledger.Ledger, error) {
	if len(p.Allowed) > 0 && !slices.Contains(p.Allowed, p.TelegramID) {
		return nil, ErrNotAllowed
	}

	if _, err := s.ledgers.FindByMember(ctx, p.TelegramID); err == nil {
		return nil, ErrAlreadyInLedger
	} else if !errors.Is(err, common.ErrNotFound) {
		return nil, fmt.Errorf("check membership: %w", err)
	}

	currency, err := common.NewCurrency(p.Currency)
	if err != nil {
		return nil, err
	}

	now := s.now()
	owner, err := ledger.NewMember(p.TelegramID, p.Name, ledger.RoleOwner, now)
	if err != nil {
		return nil, err
	}

	l, err := ledger.NewLedger(s.newID(), p.LedgerName, currency, owner)
	if err != nil {
		return nil, err
	}

	zero, err := common.NewMoney(decimal.Zero, p.Currency)
	if err != nil {
		return nil, err
	}
	cash, err := account.NewAccount(s.newID(), l.ID(), "Наличные", zero, account.Cash)
	if err != nil {
		return nil, err
	}

	err = s.tx.Do(ctx, func(ctx context.Context) error {
		if err := s.ledgers.Create(ctx, l); err != nil {
			return err
		}
		return s.accounts.Create(ctx, cash)
	})
	if err != nil {
		return nil, err
	}
	return l, nil
}

func (s *Service) Invite(ctx context.Context, telegramID int64) (*ledger.Invite, error) {
	l, err := s.ledgerOf(ctx, telegramID)
	if err != nil {
		return nil, err
	}

	invite, err := ledger.NewInvite(s.invite(), l.ID(), telegramID, s.now(), inviteTTL)
	if err != nil {
		return nil, err
	}
	if err := s.invites.Create(ctx, invite); err != nil {
		return nil, fmt.Errorf("create invite: %w", err)
	}
	return invite, nil
}

// Join spends the code and adds the member in one transaction: if the ledger
// refuses the member, the code stays unused rather than being burnt for nothing.
func (s *Service) Join(ctx context.Context, telegramID int64, name, code string) (*ledger.Ledger, error) {
	var joined *ledger.Ledger

	err := s.tx.Do(ctx, func(ctx context.Context) error {
		invite, err := s.invites.FindByCode(ctx, code)
		if errors.Is(err, common.ErrNotFound) {
			return ErrInviteNotFound
		}
		if err != nil {
			return fmt.Errorf("find invite: %w", err)
		}

		if err := invite.Use(telegramID, s.now()); err != nil {
			return err
		}

		l, err := s.ledgers.FindByID(ctx, invite.LedgerID())
		if err != nil {
			return fmt.Errorf("find ledger: %w", err)
		}

		member, err := ledger.NewMember(telegramID, name, ledger.RoleMember, s.now())
		if err != nil {
			return err
		}
		if err := l.Join(member); err != nil {
			return err
		}

		if err := s.ledgers.Save(ctx, l); err != nil {
			return err
		}
		if err := s.invites.Save(ctx, invite); err != nil {
			return err
		}

		joined = l
		return nil
	})

	return joined, err
}

func (s *Service) Ledger(ctx context.Context, telegramID int64) (*ledger.Ledger, error) {
	return s.ledgerOf(ctx, telegramID)
}
