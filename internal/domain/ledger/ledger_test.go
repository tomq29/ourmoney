package ledger

import (
	"errors"
	"testing"
	"time"

	"github.com/tomq29/ourmoney/internal/domain/common"
)

func kzt(t *testing.T) common.Currency {
	t.Helper()

	c, err := common.NewCurrency(common.CurrencyKZT)
	if err != nil {
		t.Fatalf("currency: %v", err)
	}
	return c
}

func owner(t *testing.T, telegramID int64) Member {
	t.Helper()

	m, err := NewMember(telegramID, "owner", RoleOwner, time.Now())
	if err != nil {
		t.Fatalf("member: %v", err)
	}
	return m
}

func member(t *testing.T, telegramID int64) Member {
	t.Helper()

	m, err := NewMember(telegramID, "partner", RoleMember, time.Now())
	if err != nil {
		t.Fatalf("member: %v", err)
	}
	return m
}

func newLedger(t *testing.T) *Ledger {
	t.Helper()

	l, err := NewLedger("id", "Home", kzt(t), owner(t, 1))
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	return l
}

func TestNewLedgerStartsWithItsOwner(t *testing.T) {
	l := newLedger(t)

	if len(l.Members()) != 1 {
		t.Fatalf("members = %d, want 1", len(l.Members()))
	}
	if !l.HasMember(1) {
		t.Error("the owner is not a member of their own ledger")
	}

	m, _ := l.Member(1)
	if !m.IsOwner() {
		t.Error("the first member is not an owner")
	}
}

// Whoever creates the ledger owns it, whatever role they asked for.
func TestNewLedgerPromotesTheCreator(t *testing.T) {
	l, err := NewLedger("id", "Home", kzt(t), member(t, 7))
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}

	m, _ := l.Member(7)
	if !m.IsOwner() {
		t.Error("the creator did not become an owner")
	}
}

func TestNewLedgerRejectsEmptyName(t *testing.T) {
	if _, err := NewLedger("id", "", kzt(t), owner(t, 1)); !errors.Is(err, ErrEmptyName) {
		t.Fatalf("err = %v, want %v", err, ErrEmptyName)
	}
}

func TestJoin(t *testing.T) {
	l := newLedger(t)

	if err := l.Join(member(t, 2)); err != nil {
		t.Fatalf("join: %v", err)
	}
	if len(l.Members()) != 2 {
		t.Fatalf("members = %d, want 2", len(l.Members()))
	}

	if err := l.Join(member(t, 2)); !errors.Is(err, ErrAlreadyMember) {
		t.Errorf("joining twice: %v, want %v", err, ErrAlreadyMember)
	}
}

func TestLeave(t *testing.T) {
	l := newLedger(t)
	if err := l.Join(member(t, 2)); err != nil {
		t.Fatalf("join: %v", err)
	}

	if err := l.Leave(2); err != nil {
		t.Fatalf("leave: %v", err)
	}
	if l.HasMember(2) {
		t.Error("the member is still there")
	}

	if err := l.Leave(2); !errors.Is(err, ErrNotMember) {
		t.Errorf("leaving twice: %v, want %v", err, ErrNotMember)
	}
}

// A ledger nobody owns can never invite anyone, so it would be lost for good.
func TestTheLastOwnerCannotLeave(t *testing.T) {
	l := newLedger(t)
	if err := l.Join(member(t, 2)); err != nil {
		t.Fatalf("join: %v", err)
	}

	if err := l.Leave(1); !errors.Is(err, ErrLastOwner) {
		t.Fatalf("err = %v, want %v", err, ErrLastOwner)
	}

	if err := l.Join(owner(t, 3)); err != nil {
		t.Fatalf("join second owner: %v", err)
	}
	if err := l.Leave(1); err != nil {
		t.Errorf("leaving with another owner present: %v", err)
	}
}

// Members() hands out a copy: reaching into it must not edit the ledger.
func TestMembersAreNotAliased(t *testing.T) {
	l := newLedger(t)

	members := l.Members()
	members[0] = member(t, 99)

	if !l.HasMember(1) {
		t.Error("the ledger was changed through the slice it returned")
	}
}

func TestNewMemberRejectsBadInput(t *testing.T) {
	if _, err := NewMember(0, "nobody", RoleOwner, time.Now()); !errors.Is(err, ErrMissingTelegramID) {
		t.Errorf("err = %v, want %v", err, ErrMissingTelegramID)
	}
	if _, err := NewMember(1, "nobody", "accountant", time.Now()); !errors.Is(err, ErrUnknownRole) {
		t.Errorf("err = %v, want %v", err, ErrUnknownRole)
	}
}
