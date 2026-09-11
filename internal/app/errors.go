package app

import (
	"crypto/rand"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrNoLedger        = errors.New("not a member of any ledger")
	ErrAlreadyInLedger = errors.New("already a member of a ledger")
	ErrAccountNotFound = errors.New("account not found")
	ErrNoAccounts      = errors.New("the ledger has no accounts")
	ErrInviteNotFound  = errors.New("invite not found")
	ErrNotAllowed      = errors.New("this telegram account may not start a ledger")
	ErrNothingToUndo   = errors.New("there is nothing to undo")
	ErrDifferentMoney  = errors.New("accounts hold different currencies")
)

func newUUID() string {
	return uuid.NewString()
}

// Invite codes are read off one screen and typed into another, so the alphabet
// leaves out the characters people mistake for each other.
const inviteAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func newInviteCode() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand does not fail in practice, and a weaker code here would
		// be worse than not handing one out at all.
		panic(err)
	}

	for i := range b {
		b[i] = inviteAlphabet[int(b[i])%len(inviteAlphabet)]
	}
	return string(b)
}
