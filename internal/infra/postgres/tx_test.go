package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/tomq29/ourmoney/internal/domain/account"
	"github.com/tomq29/ourmoney/internal/domain/category"
)

func TestTxCommitsEverythingTogether(t *testing.T) {
	reset(t)
	ctx := context.Background()

	l := seedLedger(t)
	accounts := NewAccountRepo(testPool)
	categories := NewCategoryRepo(testPool)

	err := NewTxManager(testPool).Do(ctx, func(ctx context.Context) error {
		acc, err := account.NewAccount(uuid.NewString(), l.ID(), "Cash", money(t, 1_000), account.Cash)
		if err != nil {
			return err
		}
		if err := accounts.Create(ctx, acc); err != nil {
			return err
		}

		c, err := category.NewCategory(uuid.NewString(), l.ID(), "Еда", nil, time.Now())
		if err != nil {
			return err
		}
		return categories.Create(ctx, c)
	})
	if err != nil {
		t.Fatalf("tx: %v", err)
	}

	stored, err := accounts.FindByLedgerID(ctx, l.ID())
	if err != nil || len(stored) != 1 {
		t.Fatalf("accounts = %d (err %v), want 1", len(stored), err)
	}
	if found, err := categories.FindByName(ctx, l.ID(), "Еда"); err != nil || found == nil {
		t.Fatalf("category: %v", err)
	}
}

// The point of the unit of work: two repositories, one failure, nothing left
// behind by either of them.
func TestTxRollsBackEverythingTogether(t *testing.T) {
	reset(t)
	ctx := context.Background()

	l := seedLedger(t)
	accounts := NewAccountRepo(testPool)
	categories := NewCategoryRepo(testPool)
	boom := errors.New("boom")

	err := NewTxManager(testPool).Do(ctx, func(ctx context.Context) error {
		acc, _ := account.NewAccount(uuid.NewString(), l.ID(), "Cash", money(t, 1_000), account.Cash)
		if err := accounts.Create(ctx, acc); err != nil {
			return err
		}

		c, _ := category.NewCategory(uuid.NewString(), l.ID(), "Еда", nil, time.Now())
		if err := categories.Create(ctx, c); err != nil {
			return err
		}
		return boom
	})

	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want %v", err, boom)
	}

	stored, err := accounts.FindByLedgerID(ctx, l.ID())
	if err != nil {
		t.Fatalf("accounts: %v", err)
	}
	if len(stored) != 0 {
		t.Errorf("accounts = %d, want none: the rollback left a row behind", len(stored))
	}
	if all, _ := categories.FindByLedgerID(ctx, l.ID()); len(all) != 0 {
		t.Errorf("categories = %d, want none", len(all))
	}
}

// A nested Do joins the transaction it is already in rather than opening a
// second one, so an inner failure still rolls the outer work back.
func TestNestedTxJoinsTheOuterOne(t *testing.T) {
	reset(t)
	ctx := context.Background()

	l := seedLedger(t)
	accounts := NewAccountRepo(testPool)
	manager := NewTxManager(testPool)
	boom := errors.New("boom")

	err := manager.Do(ctx, func(ctx context.Context) error {
		acc, _ := account.NewAccount(uuid.NewString(), l.ID(), "Cash", money(t, 1_000), account.Cash)
		if err := accounts.Create(ctx, acc); err != nil {
			return err
		}

		return manager.Do(ctx, func(ctx context.Context) error { return boom })
	})

	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want %v", err, boom)
	}
	if stored, _ := accounts.FindByLedgerID(ctx, l.ID()); len(stored) != 0 {
		t.Errorf("accounts = %d, want none", len(stored))
	}
}

// Without a transaction in the context the repositories still work, straight
// against the pool.
func TestRepositoriesWorkOutsideATransaction(t *testing.T) {
	reset(t)
	ctx := context.Background()

	l := seedLedger(t)
	acc := seedAccount(t, l.ID(), "Cash", 1_000)

	found, err := NewAccountRepo(testPool).FindByID(ctx, acc.ID())
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if found.Name() != "Cash" {
		t.Errorf("account = %q", found.Name())
	}
}
