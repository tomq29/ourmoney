package common

import (
	"fmt"

	"github.com/shopspring/decimal"
)

type Money struct {
	amount   decimal.Decimal
	currency Currency
}

func NewMoney(a decimal.Decimal, c CurrencyState) (Money, error) {

	currency, err := NewCurrency(c)
	if err != nil {
		return Money{}, err
	}

	return Money{amount: a, currency: currency}, nil

}

func (m Money) Add(external Money) (Money, error) {
	err := currencyCheck(m, external)
	if err != nil {
		return Money{}, fmt.Errorf("Add method; %w", err)
	}

	amount := m.amount.Add(external.amount)

	return Money{amount, m.currency}, nil
}

func (m Money) Subtract(external Money) (Money, error) {
	err := currencyCheck(m, external)
	if err != nil {
		return Money{}, fmt.Errorf("Subtract method; %w", err)
	}

	amount := m.amount.Sub(external.amount)

	return Money{amount, m.currency}, nil
}

func currencyCheck(m1, m2 Money) error {
	if m1.currency.code != m2.currency.code {
		return fmt.Errorf("currency not equal %v and %v", m1.currency, m2.currency)
	}
	return nil
}

// Multiply takes a plain number, not Money: money times money is money squared,
// which is not a thing anyone can hold.
func (m Money) Multiply(factor decimal.Decimal) Money {
	return Money{amount: m.amount.Mul(factor), currency: m.currency}
}

func (m Money) IsZero() bool {
	return m.amount.IsZero()
}

func (m Money) IsNegative() bool {
	return m.amount.IsNegative()
}

func (m Money) IsEqual(external Money) bool {

	if !m.amount.Equal(external.amount) || !m.currency.Equals(external.currency) {
		return false
	}

	return true
}

func (m Money) Amount() decimal.Decimal {
	return m.amount
}

func (m Money) CurrencyCode() string {
	return m.currency.code
}
