package transaction

import (
	"errors"
	"github.com/tomq29/ourmoney/internal/domain/common"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestMakingIncome(t *testing.T) {

	a1 := decimal.NewFromInt(100)
	c := common.CurrencyKZT

	id := "id"
	userID := "userID"
	money, _ := common.NewMoney(a1, c)

	accountToID := "account2"
	categoryID := "salary"
	comment := "mySalary"
	date := time.Now()

	_, err := NewIncome(NewIncomeParams{
		ID:          id,
		UserID:      userID,
		Amount:      money,
		ToAccountID: accountToID,
		CategoryID:  categoryID,
		Comment:     comment,
		Date:        date,
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMakingExpense(t *testing.T) {

	a1 := decimal.NewFromInt(100)
	c := common.CurrencyKZT

	id := "id"
	userID := "userID"
	money, _ := common.NewMoney(a1, c)

	accountFromID := "account2"
	categoryID := "salary"
	comment := "mySalary"
	date := time.Now()

	_, err := NewExpense(NewExpenseParams{
		ID:            id,
		UserID:        userID,
		Amount:        money,
		FromAccountID: accountFromID,
		CategoryID:    categoryID,
		Comment:       comment,
		Date:          date,
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMakingTransfer(t *testing.T) {

	a1 := decimal.NewFromInt(100)
	c := common.CurrencyKZT

	id := "id"
	userID := "userID"
	money, _ := common.NewMoney(a1, c)
	accountFromID := "account1"
	accountToID := "account2"
	categoryID := "salary"
	comment := "mySalary"
	date := time.Now()

	_, err := NewTransfer(NewTransferParams{
		ID:            id,
		UserID:        userID,
		Amount:        money,
		ToAccountID:   accountToID,
		FromAccountID: accountFromID,
		CategoryID:    categoryID,
		Comment:       comment,
		Date:          date,
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestZeroMoney(t *testing.T) {

	a1 := decimal.NewFromInt(0)
	c := common.CurrencyKZT

	id := "id"
	userID := "userID"

	money, _ := common.NewMoney(a1, c)

	accountToID := "account2"
	categoryID := "salary"
	comment := "mySalary"
	date := time.Now()

	_, err := NewIncome(NewIncomeParams{
		ID:          id,
		UserID:      userID,
		Amount:      money,
		ToAccountID: accountToID,
		CategoryID:  categoryID,
		Comment:     comment,
		Date:        date,
	})

	if !errors.Is(err, ErrZeroAmount) {
		t.Errorf("expected ErrZeroAmount, got %v", err)
	}
}

func TestMissingAccountID(t *testing.T) {

	a1 := decimal.NewFromInt(100)
	c := common.CurrencyKZT

	id := "id"
	userID := "userID"

	money, _ := common.NewMoney(a1, c)

	accountToID := ""
	categoryID := "salary"
	comment := "mySalary"
	date := time.Now()

	_, err := NewIncome(NewIncomeParams{
		ID:          id,
		UserID:      userID,
		Amount:      money,
		ToAccountID: accountToID,
		CategoryID:  categoryID,
		Comment:     comment,
		Date:        date,
	})

	if !errors.Is(err, ErrMissingAccount) {
		t.Errorf("expected ErrMissingAccount, got %v", err)
	}
}

func TestErrSameAccount(t *testing.T) {

	a1 := decimal.NewFromInt(100)
	c := common.CurrencyKZT

	id := "id"
	userID := "userID"

	money, _ := common.NewMoney(a1, c)
	accountFromID := "account1"
	accountToID := "account1"
	categoryID := "salary"
	comment := "mySalary"
	date := time.Now()

	_, err := NewTransfer(NewTransferParams{
		ID:            id,
		UserID:        userID,
		Amount:        money,
		ToAccountID:   accountToID,
		FromAccountID: accountFromID,
		CategoryID:    categoryID,
		Comment:       comment,
		Date:          date,
	})

	if !errors.Is(err, ErrSameAccount) {
		t.Errorf("expected ErrSameAccount, got %v", err)
	}
}

func TestMissingDate(t *testing.T) {
	a1 := decimal.NewFromInt(100)
	c := common.CurrencyKZT

	id := "id"
	userID := "userID"

	money, _ := common.NewMoney(a1, c)

	accountToID := "account1"
	categoryID := "salary"
	comment := "mySalary"
	date := time.Time{}

	_, err := NewIncome(NewIncomeParams{
		ID:          id,
		UserID:      userID,
		Amount:      money,
		ToAccountID: accountToID,
		CategoryID:  categoryID,
		Comment:     comment,
		Date:        date,
	})

	if !errors.Is(err, ErrMissingDate) {
		t.Errorf("expected ErrMissingDate, got %v", err)
	}
}

func TestMissingUserID(t *testing.T) {
	a1 := decimal.NewFromInt(100)
	c := common.CurrencyKZT

	id := "id"
	userID := ""
	money, _ := common.NewMoney(a1, c)

	accountToID := "account2"
	categoryID := "salary"
	comment := "mySalary"
	date := time.Now()

	_, err := NewIncome(NewIncomeParams{
		ID:          id,
		UserID:      userID,
		Amount:      money,
		ToAccountID: accountToID,
		CategoryID:  categoryID,
		Comment:     comment,
		Date:        date,
	})

	if !errors.Is(err, ErrMissingUserID) {
		t.Errorf("expected ErrMissingUserID, got %v", err)
	}
}
