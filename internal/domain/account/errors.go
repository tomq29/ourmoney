package account

import "errors"

var ErrInsufficientFunds = errors.New("недостаточно средств")
var ErrAccountArchived = errors.New("операция с архивным счётом")
var ErrEmptyName = errors.New("пустое название счёта")
var ErrCurrencyMismatch = errors.New("валюта операции не совпадает с валютой счёта")
