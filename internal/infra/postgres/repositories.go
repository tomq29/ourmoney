package postgres

import (
	"github.com/tomq29/ourmoney/internal/domain/account"
	"github.com/tomq29/ourmoney/internal/domain/category"
	"github.com/tomq29/ourmoney/internal/domain/ledger"
	"github.com/tomq29/ourmoney/internal/domain/transaction"
)

// The domain declares what it needs from storage; these lines fail the build
// the moment an implementation drifts from it.
var (
	_ account.AccountRepository         = (*AccountRepo)(nil)
	_ category.CategoryRepository       = (*CategoryRepo)(nil)
	_ ledger.LedgerRepository           = (*LedgerRepo)(nil)
	_ ledger.InviteRepository           = (*InviteRepo)(nil)
	_ transaction.TransactionRepository = (*TransactionRepo)(nil)
)
