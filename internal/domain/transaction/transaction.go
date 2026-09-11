package transaction

import (
	"fmt"
	"github.com/tomq29/ourmoney/internal/domain/common"
	"time"
)

type Transaction struct {
	id              string
	userID          string
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
	UserID      string
	Amount      common.Money
	ToAccountID string
	CategoryID  string
	Comment     string
	Date        time.Time
}

func NewIncome(p NewIncomeParams) (*Transaction, error) {

	err := checkingTransaction(p.UserID, p.Amount, p.Date, "NewIncome")
	if err != nil {
		return nil, err
	}

	if len(p.ToAccountID) == 0 {
		return nil, fmt.Errorf("making NewIncome, %w", ErrMissingAccount)
	}

	return &Transaction{
		id:              p.ID,
		userID:          p.UserID,
		transactionType: Income,
		amount:          p.Amount,
		fromAccountID:   "",
		toAccountID:     p.ToAccountID,
		categoryID:      p.CategoryID,
		comment:         p.Comment,
		date:            p.Date,
	}, nil

}

func checkingTransaction(userID string, amount common.Money, date time.Time, operationName string) error {

	if len(userID) == 0 {
		return fmt.Errorf("making %v, %w", operationName, ErrMissingUserID)

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
	UserID        string
	Amount        common.Money
	FromAccountID string
	CategoryID    string
	Comment       string
	Date          time.Time
}

func NewExpense(p NewExpenseParams) (*Transaction, error) {

	err := checkingTransaction(p.UserID, p.Amount, p.Date, "NewExpense")
	if err != nil {
		return nil, err
	}

	if len(p.FromAccountID) == 0 {
		return nil, fmt.Errorf("making NewExpense, %w", ErrMissingAccount)
	}

	return &Transaction{
		id:              p.ID,
		userID:          p.UserID,
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
	UserID        string
	Amount        common.Money
	ToAccountID   string
	FromAccountID string
	CategoryID    string
	Comment       string
	Date          time.Time
}

func NewTransfer(p NewTransferParams) (*Transaction, error) {

	err := checkingTransaction(p.UserID, p.Amount, p.Date, "NewTransfer")
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
		userID:          p.UserID,
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

func (t *Transaction) UserID() string {
	return t.userID
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
