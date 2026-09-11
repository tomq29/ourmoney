package bot

import (
	"errors"
	"regexp"
	"strings"

	"github.com/shopspring/decimal"
)

var ErrNotAnEntry = errors.New("not an entry")

type Entry struct {
	Income   bool
	Amount   decimal.Decimal
	Category string
	Comment  string
	Account  string
}

// An entry is typed in a hurry, one-handed, while leaving a shop:
//
//	-5000 такси в аэропорт
//	5000 продукты
//	+200000 зарплата
//	-3.5к кофе @карта
//
// A bare number is an expense, because nearly every entry is one. The suffix
// "к" multiplies by a thousand, the way people say it out loud.
//
// The multiplier has to be followed by a space or the end of the line, spelled
// out rather than left to \b: Go's word boundary is ASCII-only, so "3.5к кофе"
// would otherwise read the "к" of "кофе" as a category.
var entryPattern = regexp.MustCompile(`^([-+])?\s*([0-9]+(?:[.,][0-9]+)?)\s*([кkКK])?(\s.*|)$`)

func ParseEntry(text string) (Entry, error) {
	fields := entryPattern.FindStringSubmatch(strings.TrimSpace(text))
	if fields == nil {
		return Entry{}, ErrNotAnEntry
	}

	amount, err := decimal.NewFromString(strings.Replace(fields[2], ",", ".", 1))
	if err != nil {
		return Entry{}, ErrNotAnEntry
	}
	if fields[3] != "" {
		amount = amount.Mul(decimal.NewFromInt(1000))
	}
	if !amount.IsPositive() {
		return Entry{}, ErrNotAnEntry
	}

	entry := Entry{Income: fields[1] == "+", Amount: amount}

	rest, account := splitAccount(fields[4])
	entry.Account = account

	words := strings.Fields(rest)
	if len(words) > 0 {
		entry.Category = words[0]
		entry.Comment = strings.Join(words[1:], " ")
	}
	return entry, nil
}

// splitAccount pulls a trailing "@name" off the text. Naming the account is
// the exception, so it sits out of the way of the words that always come.
func splitAccount(text string) (rest, account string) {
	words := strings.Fields(text)
	for i, w := range words {
		if strings.HasPrefix(w, "@") && len(w) > 1 {
			account = strings.TrimPrefix(w, "@")
			return strings.Join(append(words[:i:i], words[i+1:]...), " "), account
		}
	}
	return text, ""
}

type Command struct {
	Name string
	Args []string
}

// ParseCommand recognises "/undo" and "/undo@ourmoneybot" alike: Telegram
// appends the bot's name when a command is sent in a group.
func ParseCommand(text string) (Command, bool) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return Command{}, false
	}

	fields := strings.Fields(text)
	name := strings.TrimPrefix(fields[0], "/")
	if at := strings.IndexByte(name, '@'); at >= 0 {
		name = name[:at]
	}

	return Command{Name: strings.ToLower(name), Args: fields[1:]}, true
}
