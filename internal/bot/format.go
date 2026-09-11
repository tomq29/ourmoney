package bot

import (
	"errors"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/tomq29/ourmoney/internal/app"
	"github.com/tomq29/ourmoney/internal/domain/account"
	"github.com/tomq29/ourmoney/internal/domain/transaction"
)

var currencySign = map[string]string{
	"kzt":  "₸",
	"rub":  "₽",
	"usd":  "$",
	"usdt": "USDT",
}

// formatMoney writes 1234567.00 as "1 234 567 ₸": the trailing zeros of a
// whole sum are noise on a phone screen.
func formatMoney(amount decimal.Decimal, currencyCode string) string {
	text := amount.StringFixed(2)
	text = strings.TrimSuffix(text, ".00")

	negative := strings.HasPrefix(text, "-")
	text = strings.TrimPrefix(text, "-")

	whole, fraction, _ := strings.Cut(text, ".")
	var grouped strings.Builder
	for i, digit := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			grouped.WriteString(" ")
		}
		grouped.WriteRune(digit)
	}

	out := grouped.String()
	if fraction != "" {
		out += "," + fraction
	}
	if negative {
		out = "−" + out
	}

	if sign, ok := currencySign[currencyCode]; ok {
		return out + " " + sign
	}
	return out + " " + strings.ToUpper(currencyCode)
}

func transactionWord(kind transaction.TransactionType) string {
	switch kind {
	case transaction.Income:
		return "доход"
	case transaction.Transfer:
		return "перевод"
	}
	return "трату"
}

// fail turns a domain error into something worth reading. Anything it does not
// recognise is logged in full and answered vaguely: a stack of wrapped
// database errors helps nobody holding a phone.
func (b *Bot) fail(op string, err error) string {
	switch {
	case errors.Is(err, app.ErrNoLedger):
		return "Сначала /start — или /join КОД, если вас позвали в общий бюджет."
	case errors.Is(err, app.ErrAlreadyInLedger):
		return "Вы уже в бюджете."
	case errors.Is(err, app.ErrAccountNotFound):
		return "Не нашёл такой счёт. /accounts покажет, какие есть."
	case errors.Is(err, app.ErrNoAccounts):
		return "Нет ни одного счёта."
	case errors.Is(err, app.ErrNothingToUndo):
		return "Отменять нечего."
	case errors.Is(err, account.ErrInsufficientFunds):
		return "На счёте столько нет."
	case errors.Is(err, account.ErrAccountArchived):
		return "Этот счёт в архиве."
	case errors.Is(err, app.ErrDifferentMoney):
		return "Счета в разных валютах — так не сходится."
	}

	b.log.Error(op, "error", err)
	return "Что-то сломалось у меня, попробуйте ещё раз."
}
