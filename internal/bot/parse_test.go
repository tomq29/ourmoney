package bot

import (
	"errors"
	"testing"
)

func TestParseEntry(t *testing.T) {
	cases := map[string]struct {
		text     string
		income   bool
		amount   string
		category string
		comment  string
		account  string
	}{
		"expense with a comment": {
			text: "-5000 такси в аэропорт", amount: "5000", category: "такси", comment: "в аэропорт",
		},
		"bare number is an expense": {
			text: "5000 продукты", amount: "5000", category: "продукты",
		},
		"income": {
			text: "+200000 зарплата", income: true, amount: "200000", category: "зарплата",
		},
		"thousands in russian": {
			text: "-3.5к кофе", amount: "3500", category: "кофе",
		},
		"thousands in latin": {
			text: "-3k кофе", amount: "3000", category: "кофе",
		},
		"comma as a decimal point": {
			text: "-1234,50 обед", amount: "1234.5", category: "обед",
		},
		"named account": {
			text: "-5000 кофе @карта", amount: "5000", category: "кофе", account: "карта",
		},
		"account before the comment": {
			text: "-5000 кофе @карта с утра", amount: "5000", category: "кофе", comment: "с утра", account: "карта",
		},
		"amount alone": {
			text: "-700", amount: "700",
		},
		"spaces around the sign": {
			text: "- 700 такси", amount: "700", category: "такси",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			entry, err := ParseEntry(c.text)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}

			if entry.Income != c.income {
				t.Errorf("income = %v, want %v", entry.Income, c.income)
			}
			if entry.Amount.String() != c.amount {
				t.Errorf("amount = %s, want %s", entry.Amount, c.amount)
			}
			if entry.Category != c.category {
				t.Errorf("category = %q, want %q", entry.Category, c.category)
			}
			if entry.Comment != c.comment {
				t.Errorf("comment = %q, want %q", entry.Comment, c.comment)
			}
			if entry.Account != c.account {
				t.Errorf("account = %q, want %q", entry.Account, c.account)
			}
		})
	}
}

func TestParseEntryRejects(t *testing.T) {
	cases := map[string]string{
		"words only":    "привет",
		"empty":         "",
		"zero":          "0 еда",
		"a command":     "/report",
		"no amount":     "-такси",
		"amount inside": "купил за 5000",
	}

	for name, text := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseEntry(text); !errors.Is(err, ErrNotAnEntry) {
				t.Fatalf("err = %v, want %v", err, ErrNotAnEntry)
			}
		})
	}
}

func TestParseCommand(t *testing.T) {
	cases := map[string]struct {
		text string
		name string
		args []string
	}{
		"plain":              {"/report", "report", nil},
		"with arguments":     {"/transfer 30000 наличные карта", "transfer", []string{"30000", "наличные", "карта"}},
		"addressed in group": {"/undo@ourmoneybot", "undo", nil},
		"mixed case":         {"/Report", "report", nil},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			cmd, ok := ParseCommand(c.text)
			if !ok {
				t.Fatal("not recognised as a command")
			}
			if cmd.Name != c.name {
				t.Errorf("name = %q, want %q", cmd.Name, c.name)
			}
			if len(cmd.Args) != len(c.args) {
				t.Fatalf("args = %v, want %v", cmd.Args, c.args)
			}
			for i := range c.args {
				if cmd.Args[i] != c.args[i] {
					t.Errorf("arg %d = %q, want %q", i, cmd.Args[i], c.args[i])
				}
			}
		})
	}

	if _, ok := ParseCommand("-5000 еда"); ok {
		t.Error("an entry was taken for a command")
	}
}

func TestFormatMoney(t *testing.T) {
	cases := []struct {
		amount   string
		currency string
		want     string
	}{
		{"1234567", "kzt", "1 234 567 ₸"},
		{"1000", "kzt", "1 000 ₸"},
		{"999", "kzt", "999 ₸"},
		{"1234.50", "rub", "1 234,50 ₽"},
		{"0", "kzt", "0 ₸"},
		{"-500", "kzt", "−500 ₸"},
		{"100", "eur", "100 EUR"},
	}

	for _, c := range cases {
		t.Run(c.want, func(t *testing.T) {
			amount := decimalFrom(t, c.amount)
			if got := formatMoney(amount, c.currency); got != c.want {
				t.Errorf("formatMoney(%s) = %q, want %q", c.amount, got, c.want)
			}
		})
	}
}
