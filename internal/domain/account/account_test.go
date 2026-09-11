package account

import (
	"errors"
	"github.com/tomq29/ourmoney/internal/domain/common"
	"testing"

	"github.com/shopspring/decimal"
)

func TestValidNewAccount(t *testing.T) {

	a1 := decimal.NewFromInt(100)
	c := common.CurrencyKZT

	money, _ := common.NewMoney(a1, c)

	id := "id"
	userID := "userID"
	name := "name"
	balance := money
	accountType := BankCard

	account, err := NewAccount(id, userID, name, balance, accountType)

	if err != nil {
		t.Errorf("error in new Account, failed test")
	}

	if account.ID() != id ||
		account.UserID() != userID ||
		account.Name() != name ||
		!account.Balance().IsEqual(balance) ||
		account.accountType != accountType ||
		account.IsArchived() {
		t.Errorf("new Account field not equal to initial, failed test")
	}

}

func TestEmptyAccountName(t *testing.T) {

	a1 := decimal.NewFromInt(100)
	c := common.CurrencyKZT

	money, _ := common.NewMoney(a1, c)

	id := "id"
	userID := "userID"
	name := ""
	balance := money
	accountType := BankCard

	_, err := NewAccount(id, userID, name, balance, accountType)

	if !errors.Is(err, ErrEmptyName) {
		t.Errorf("expected ErrEmptyName, got %v", err)
	}
}

func TestAccountDeposit(t *testing.T) {

	a1 := decimal.NewFromInt(100)
	a2 := decimal.NewFromInt(11)
	a3 := decimal.NewFromInt(111)
	c := common.CurrencyKZT

	startMoney, _ := common.NewMoney(a1, c)
	addingMoney, _ := common.NewMoney(a2, c)
	expectMoney, _ := common.NewMoney(a3, c)

	id := "id"
	userID := "userID"
	name := "name"
	balance := startMoney
	accountType := BankCard

	account, err := NewAccount(id, userID, name, balance, accountType)

	if err != nil {
		t.Errorf("error in new Account, failed test")
	}

	err = account.Deposit(addingMoney)

	if err != nil {
		t.Errorf("error in Deposit func, failed test")
	}

	if !account.Balance().IsEqual(expectMoney) {
		t.Errorf("failed test")
	}

}

func TestArchiveAccountDeposit(t *testing.T) {

	a1 := decimal.NewFromInt(100)
	a2 := decimal.NewFromInt(100)

	c := common.CurrencyKZT

	startMoney, _ := common.NewMoney(a1, c)
	addingMoney, _ := common.NewMoney(a2, c)

	id := "id"
	userID := "userID"
	name := "name"
	balance := startMoney
	accountType := BankCard

	account, err := NewAccount(id, userID, name, balance, accountType)

	if err != nil {
		t.Errorf("error in new Account, failed test")
	}

	account.Archive()

	err = account.Deposit(addingMoney)

	if !errors.Is(err, ErrAccountArchived) {
		t.Errorf("expected ErrAccountArchived, got %v", err)
	}

}

func TestArchiveAccountChange(t *testing.T) {
	a1 := decimal.NewFromInt(100)

	c := common.CurrencyKZT

	startMoney, _ := common.NewMoney(a1, c)

	id := "id"
	userID := "userID"
	name := "name"
	balance := startMoney
	accountType := BankCard

	account, err := NewAccount(id, userID, name, balance, accountType)

	if err != nil {
		t.Errorf("error in new Account, failed test")
	}

	account.Archive()

	if !account.IsArchived() {
		t.Errorf("failed test")
	}

	account.Unarchive()

	if account.IsArchived() {
		t.Errorf("failed test")
	}

}

func TestWithdraw(t *testing.T) {
	a1 := decimal.NewFromInt(100)
	a2 := decimal.NewFromInt(25)
	a3 := decimal.NewFromInt(75)
	c := common.CurrencyKZT

	startMoney, _ := common.NewMoney(a1, c)
	substractMoney, _ := common.NewMoney(a2, c)
	expectMoney, _ := common.NewMoney(a3, c)

	id := "id"
	userID := "userID"
	name := "name"
	balance := startMoney
	accountType := BankCard

	account, err := NewAccount(id, userID, name, balance, accountType)

	if err != nil {
		t.Errorf("error in new Account, failed test")
	}

	err = account.Withdraw(substractMoney)

	if err != nil {
		t.Errorf("error in Deposit func, failed test")
	}

	if !account.Balance().IsEqual(expectMoney) {
		t.Errorf("failed test")
	}
}

func TestWithdrawError(t *testing.T) {
	a1 := decimal.NewFromInt(100)
	a2 := decimal.NewFromInt(125)

	c := common.CurrencyKZT

	startMoney, _ := common.NewMoney(a1, c)
	substractMoney, _ := common.NewMoney(a2, c)

	id := "id"
	userID := "userID"
	name := "name"
	balance := startMoney
	accountType := BankCard

	account, err := NewAccount(id, userID, name, balance, accountType)

	if err != nil {
		t.Errorf("error in new Account, failed test")
	}

	err = account.Withdraw(substractMoney)

	if !errors.Is(err, ErrInsufficientFunds) {
		t.Errorf("expected ErrInsufficientFunds, got %v", err)
	}

}
