package transaction

import (
	"fmt"
	"github.com/tomq29/ourmoney/internal/domain/common"
	"time"
)

type Transaction struct {
	id              string
	ledgerID        string
	createdBy       int64
	transactionType TransactionType
	amount          common.Money
	fromAccountID   string
	toAccountID     string
	categoryID      string
	comment         string
	date            time.Time
}

type NewIncomeParams struct {
	ID          string
	LedgerID    string
	CreatedBy   int64
	Amount      common.Money
	ToAccountID string
	CategoryID  string
	Comment     string
	Date        time.Time
}

func NewIncome(p NewIncomeParams) (*Transaction, error) {

	err := checkingTransaction(p.LedgerID, p.CreatedBy, p.Amount, p.Date, "NewIncome")
	if err != nil {
		return nil, err
	}

	if len(p.ToAccountID) == 0 {
		return nil, fmt.Errorf("making NewIncome, %w", ErrMissingAccount)
	}

	return &Transaction{
		id:              p.ID,
		ledgerID:        p.LedgerID,
		createdBy:       p.CreatedBy,
		transactionType: Income,
		amount:          p.Amount,
		fromAccountID:   "",
		toAccountID:     p.ToAccountID,
		categoryID:      p.CategoryID,
		comment:         p.Comment,
		date:            p.Date,
	}, nil

}

func checkingTransaction(ledgerID string, createdBy int64, amount common.Money, date time.Time, operationName string) error {

	if len(ledgerID) == 0 {
		return fmt.Errorf("making %v, %w", operationName, ErrMissingLedgerID)
	}

	if createdBy <= 0 {
		return fmt.Errorf("making %v, %w", operationName, ErrMissingAuthor)
	}

	if amount.IsNegative() || amount.IsZero() {
		return fmt.Errorf("making %v, %w", operationName, ErrZeroAmount)
	}

	if date.IsZero() {
		return fmt.Errorf("making %v, %w", operationName, ErrMissingDate)
	}

	return nil
}

type NewExpenseParams struct {
	ID            string
	LedgerID      string
	CreatedBy     int64
	Amount        common.Money
	FromAccountID string
	CategoryID    string
	Comment       string
	Date          time.Time
}

func NewExpense(p NewExpenseParams) (*Transaction, error) {

	err := checkingTransaction(p.LedgerID, p.CreatedBy, p.Amount, p.Date, "NewExpense")
	if err != nil {
		return nil, err
	}

	if len(p.FromAccountID) == 0 {
		return nil, fmt.Errorf("making NewExpense, %w", ErrMissingAccount)
	}

	return &Transaction{
		id:              p.ID,
		ledgerID:        p.LedgerID,
		createdBy:       p.CreatedBy,
		transactionType: Expense,
		amount:          p.Amount,
		fromAccountID:   p.FromAccountID,
		toAccountID:     "",
		categoryID:      p.CategoryID,
		comment:         p.Comment,
		date:            p.Date,
	}, nil

}

type NewTransferParams struct {
	ID            string
	LedgerID      string
	CreatedBy     int64
	Amount        common.Money
	ToAccountID   string
	FromAccountID string
	CategoryID    string
	Comment       string
	Date          time.Time
}

func NewTransfer(p NewTransferParams) (*Transaction, error) {

	err := checkingTransaction(p.LedgerID, p.CreatedBy, p.Amount, p.Date, "NewTransfer")
	if err != nil {
		return nil, err
	}

	if len(p.FromAccountID) == 0 {
		return nil, fmt.Errorf("making NewTransfer, %w", ErrMissingAccount)

	}

	if p.FromAccountID == p.ToAccountID {
		return nil, fmt.Errorf("making NewTransfer, %w", ErrSameAccount)
	}

	return &Transaction{
		id:              p.ID,
		ledgerID:        p.LedgerID,
		createdBy:       p.CreatedBy,
		transactionType: Transfer,
		amount:          p.Amount,
		fromAccountID:   p.FromAccountID,
		toAccountID:     p.ToAccountID,
		categoryID:      p.CategoryID,
		comment:         p.Comment,
		date:            p.Date,
	}, nil

}

func (t *Transaction) ID() string {
	return t.id
}

func (t *Transaction) LedgerID() string {
	return t.ledgerID
}

func (t *Transaction) Amount() common.Money {
	return t.amount
}

func (t *Transaction) FromAccountID() string {
	return t.fromAccountID
}

func (t *Transaction) ToAccountID() string {
	return t.toAccountID
}

func (t *Transaction) CreatedBy() int64 {
	return t.createdBy
}

func (t *Transaction) Type() TransactionType {
	return t.transactionType
}

func (t *Transaction) CategoryID() string {
	return t.categoryID
}

func (t *Transaction) Comment() string {
	return t.comment
}

func (t *Transaction) Date() time.Time {
	return t.date
}
