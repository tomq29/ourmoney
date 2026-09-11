package transaction

import "errors"

// Domain errors are identifiers, not text for a human to read: the bot decides
// how to say them, and in which language.
var (
	ErrZeroAmount      = errors.New("amount must be greater than zero")
	ErrMissingAccount  = errors.New("required account is missing")
	ErrSameAccount     = errors.New("transfer to the same account")
	ErrMissingDate     = errors.New("date is missing")
	ErrMissingLedgerID = errors.New("ledger id is missing")
	ErrMissingAuthor   = errors.New("author is missing")
)
