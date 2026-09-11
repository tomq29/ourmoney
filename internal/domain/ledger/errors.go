package ledger

import "errors"

// Domain errors are identifiers, not text for a human to read: the bot decides
// how to say them, and in which language.
var (
	ErrEmptyName         = errors.New("ledger name is empty")
	ErrMissingTelegramID = errors.New("telegram id is missing")
	ErrUnknownRole       = errors.New("unknown member role")
	ErrAlreadyMember     = errors.New("already a member of this ledger")
	ErrNotMember         = errors.New("not a member of this ledger")
	ErrLastOwner         = errors.New("cannot remove the last owner")
	ErrEmptyCode         = errors.New("invite code is empty")
	ErrInviteExpired     = errors.New("invite has expired")
	ErrInviteUsed        = errors.New("invite has already been used")
)
