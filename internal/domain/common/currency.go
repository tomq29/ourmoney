package common

import (
	"errors"
)

type CurrencyState int

const (
	CurrencyUSD CurrencyState = iota
	CurrencyKZT
	CurrencyRUB
	CurrencyUSDT
)

var stateName = map[CurrencyState]string{
	CurrencyUSD:  "usd",
	CurrencyUSDT: "usdt",
	CurrencyKZT:  "kzt",
	CurrencyRUB:  "rub",
}

var CurrencyConverterFromString = map[string]CurrencyState{
	"usd":  CurrencyUSD,
	"usdt": CurrencyUSDT,
	"kzt":  CurrencyKZT,
	"rub":  CurrencyRUB,
}

type Currency struct {
	code string
}

func (c Currency) String() string {
	return c.code
}

func (c Currency) Equals(anotherC Currency) bool {
	return c.code == anotherC.code
}

func NewCurrency(c CurrencyState) (Currency, error) {
	cName, ok := stateName[c]
	if !ok {
		return Currency{}, errors.New("currency not found")
	}

	return Currency{code: cName}, nil
}
