package common

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestAddMoney(t *testing.T) {

	a1 := decimal.NewFromInt(100)
	a2 := decimal.NewFromInt(200)
	want := decimal.NewFromInt(300)

	c := CurrencyKZT

	m1, err := NewMoney(a1, c)
	m2, err := NewMoney(a2, c)

	if err != nil {
		t.Errorf("error in init")
		return
	}

	m1, err = m1.Add(m2)

	if err != nil {
		t.Errorf("error in add")
		return
	}

	if !m1.amount.Equal(want) {
		t.Errorf("failed test")
	}

}

func TestAddError(t *testing.T) {

	c1 := CurrencyKZT
	c2 := CurrencyRUB
	a1 := decimal.NewFromInt(100)

	m1, err := NewMoney(a1, c1)
	m2, err := NewMoney(a1, c2)

	if err != nil {
		t.Errorf("error in init")
		return
	}

	m1, err = m1.Add(m2)

	if err == nil {
		t.Errorf("failed test")
	}

}

func TestMultiply(t *testing.T) {

	a1 := decimal.NewFromInt(100)
	factor := decimal.NewFromInt(2)

	want := decimal.NewFromInt(200)

	c := CurrencyKZT

	m1, err := NewMoney(a1, c)

	if err != nil {
		t.Errorf("error in init")
		return
	}

	m1 = m1.Multiply(factor)

	if !m1.amount.Equal(want) {
		t.Errorf("failed test")
	}

	if m1.CurrencyCode() != "kzt" {
		t.Errorf("multiply changed the currency")
	}

}

func TestImmutability(t *testing.T) {

	a1 := decimal.NewFromInt(100)
	a2 := decimal.NewFromInt(50)

	c := CurrencyKZT

	m1, err := NewMoney(a1, c)
	oldM1, err := NewMoney(a1, c)
	m2, err := NewMoney(a2, c)

	if err != nil {
		t.Errorf("error in init")
		return
	}

	_, err = m1.Add(m2)

	if !m1.IsEqual(oldM1) {
		t.Errorf("failed test")
	}

}
