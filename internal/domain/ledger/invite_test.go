package ledger

import (
	"errors"
	"testing"
	"time"
)

func TestInviteIsUsedOnce(t *testing.T) {
	now := time.Now()
	invite, err := NewInvite("ABC123", "ledger", 1, now, time.Hour)
	if err != nil {
		t.Fatalf("invite: %v", err)
	}

	if invite.IsUsed() {
		t.Fatal("a fresh invite is already used")
	}

	if err := invite.Use(2, now.Add(time.Minute)); err != nil {
		t.Fatalf("use: %v", err)
	}
	if invite.UsedBy() != 2 {
		t.Errorf("used by %d, want 2", invite.UsedBy())
	}

	if err := invite.Use(3, now.Add(2*time.Minute)); !errors.Is(err, ErrInviteUsed) {
		t.Errorf("second use: %v, want %v", err, ErrInviteUsed)
	}
	if invite.UsedBy() != 2 {
		t.Error("the second attempt overwrote who used it")
	}
}

func TestInviteExpires(t *testing.T) {
	now := time.Now()
	invite, err := NewInvite("ABC123", "ledger", 1, now, time.Hour)
	if err != nil {
		t.Fatalf("invite: %v", err)
	}

	if err := invite.Use(2, now.Add(time.Hour+time.Second)); !errors.Is(err, ErrInviteExpired) {
		t.Fatalf("err = %v, want %v", err, ErrInviteExpired)
	}
	if invite.IsUsed() {
		t.Error("an expired invite was marked used")
	}
}

func TestNewInviteRejectsBadInput(t *testing.T) {
	if _, err := NewInvite("", "ledger", 1, time.Now(), time.Hour); !errors.Is(err, ErrEmptyCode) {
		t.Errorf("err = %v, want %v", err, ErrEmptyCode)
	}
	if _, err := NewInvite("ABC123", "ledger", 0, time.Now(), time.Hour); !errors.Is(err, ErrMissingTelegramID) {
		t.Errorf("err = %v, want %v", err, ErrMissingTelegramID)
	}
}
