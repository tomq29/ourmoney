package account

import "errors"

// Domain errors are identifiers, not text for a human to read: the bot decides
// how to say them, and in which language.
var (
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrAccountArchived   = errors.New("operation on an archived account")
	ErrEmptyName         = errors.New("account name is empty")
	ErrCurrencyMismatch  = errors.New("operation currency does not match the account")
)
