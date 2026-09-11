package account

import (
	"fmt"
	"github.com/tomq29/ourmoney/internal/domain/common"
	"time"
)

type Account struct {
	id          string
	userID      string
	name        string
	balance     common.Money
	accountType AccountType
	isArchived  bool
	createAt    time.Time
	updateAt    time.Time
}

func NewAccount(id, usedID, name string, balance common.Money, accountType AccountType) (*Account, error) {

	if len(name) == 0 {
		return nil, fmt.Errorf("making NewAccount %w", ErrEmptyName)
	}

	return &Account{
			id:          id,
			userID:      usedID,
			name:        name,
			balance:     balance,
			accountType: accountType,
			isArchived:  false,
			createAt:    time.Now(),
			updateAt:    time.Now(),
		},
		nil

}

func RestoreAccount(id, usedID, name string, balance common.Money, accountType AccountType, isArchived bool, createAt, updateAt time.Time) *Account {

	return &Account{id, usedID, name, balance, accountType, isArchived, createAt, updateAt}

}

func (a *Account) Deposit(amount common.Money) error {
	if a.isArchived {
		return fmt.Errorf("deposit operation in accountID %v %w", a.id, ErrAccountArchived)
	}

	newBalance, err := a.balance.Add(amount)
	if err != nil {
		return fmt.Errorf("deposit operation in accountID %v %w", a.id, err)
	}

	a.balance = newBalance
	a.updateAt = time.Now()

	return nil

}

func (a *Account) Withdraw(amount common.Money) error {
	if a.isArchived {
		return fmt.Errorf("withdraw operation in accountID %v %w", a.id, ErrAccountArchived)
	}

	newBalance, err := a.balance.Subtract(amount)
	if err != nil {
		return fmt.Errorf("withdraw operation in accountID %v %w", a.id, err)
	}

	if newBalance.IsNegative() {
		return fmt.Errorf("withdraw operation in accountID %v %w", a.id, ErrInsufficientFunds)
	}

	a.balance = newBalance
	a.updateAt = time.Now()

	return nil
}

func (a *Account) Archive() {
	a.isArchived = true
}

func (a *Account) Unarchive() {
	a.isArchived = false
}

func (a *Account) ID() string {
	return a.id
}

func (a *Account) UserID() string {
	return a.userID
}

func (a *Account) Name() string {
	return a.name
}

func (a *Account) Balance() common.Money {
	return a.balance
}

func (a *Account) AccountType() AccountType {
	return a.accountType
}

func (a *Account) IsArchived() bool {
	return a.isArchived
}

func (a *Account) CreateAt() time.Time {
	return a.createAt
}

func (a *Account) UpdateAt() time.Time {
	return a.updateAt
}
