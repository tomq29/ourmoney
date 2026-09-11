package bot

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/tomq29/ourmoney/internal/app"
	"github.com/tomq29/ourmoney/internal/domain/account"
	"github.com/tomq29/ourmoney/internal/domain/transaction"
)

const help = `Запишите трату одной строкой:

<code>-5000 такси в аэропорт</code>
<code>5000 продукты</code> — минус можно не ставить
<code>+200000 зарплата</code> — это доход
<code>-3.5к кофе @карта</code> — «к» это тысячи, @ выбирает счёт

/accounts — счета и остатки
/balance — сколько всего
/report — траты за месяц по категориям
/transfer 30000 наличные карта — перевод между счетами
/undo — отменить последнюю запись
/invite — позвать второго человека в общий бюджет`

func (b *Bot) command(ctx context.Context, from *User, cmd Command) string {
	switch cmd.Name {
	case "start":
		return b.start(ctx, from)
	case "help":
		return help
	case "join":
		return b.join(ctx, from, cmd.Args)
	case "invite":
		return b.inviteMember(ctx, from)
	case "accounts":
		return b.accounts(ctx, from)
	case "balance":
		return b.balance(ctx, from)
	case "report":
		return b.report(ctx, from)
	case "transfer":
		return b.transfer(ctx, from, cmd.Args)
	case "undo":
		return b.undo(ctx, from)
	}
	return "Не знаю такой команды. /help покажет, что я умею."
}

func (b *Bot) start(ctx context.Context, from *User) string {
	if _, err := b.service.Ledger(ctx, from.ID); err == nil {
		return "С возвращением. " + help
	}

	_, err := b.service.Start(ctx, app.StartParams{
		TelegramID: from.ID,
		Name:       from.Name(),
		LedgerName: b.cfg.LedgerName,
		Currency:   b.cfg.Currency,
		Allowed:    b.cfg.Allowed,
	})
	switch {
	case errors.Is(err, app.ErrNotAllowed):
		return "Этот бот заведён для одной семьи. Попросите код приглашения и пришлите /join КОД."
	case err != nil:
		return b.fail("start", err)
	}

	return "Бюджет заведён, счёт «Наличные» уже в нём.\n\n" + help
}

func (b *Bot) join(ctx context.Context, from *User, args []string) string {
	if len(args) != 1 {
		return "Нужен код: /join ABCD1234"
	}

	l, err := b.service.Join(ctx, from.ID, from.Name(), strings.ToUpper(args[0]))
	switch {
	case errors.Is(err, app.ErrInviteNotFound):
		return "Такого кода нет."
	case err != nil:
		return b.fail("join", err)
	}
	return fmt.Sprintf("Готово, теперь вы в бюджете «%s». %s", l.Name(), "Запишите первую трату: -5000 продукты")
}

func (b *Bot) inviteMember(ctx context.Context, from *User) string {
	invite, err := b.service.Invite(ctx, from.ID)
	if err != nil {
		return b.fail("invite", err)
	}

	return fmt.Sprintf("Перешлите это второму человеку — код живёт сутки:\n\n<code>/join %s</code>", invite.Code())
}

func (b *Bot) accounts(ctx context.Context, from *User) string {
	accounts, err := b.service.ListAccounts(ctx, from.ID)
	if err != nil {
		return b.fail("accounts", err)
	}
	if len(accounts) == 0 {
		return "Счетов пока нет."
	}

	var out strings.Builder
	for _, acc := range accounts {
		if acc.IsArchived() {
			continue
		}
		fmt.Fprintf(&out, "%s — %s\n", acc.Name(), formatMoney(acc.Balance().Amount(), acc.Balance().CurrencyCode()))
	}
	return strings.TrimRight(out.String(), "\n")
}

func (b *Bot) balance(ctx context.Context, from *User) string {
	total, err := b.service.Total(ctx, from.ID)
	if err != nil {
		return b.fail("balance", err)
	}
	return "Всего: " + formatMoney(total.Amount(), total.CurrencyCode())
}

func (b *Bot) report(ctx context.Context, from *User) string {
	now := time.Now()
	from_ := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	report, err := b.service.Report(ctx, from.ID, from_, now)
	if err != nil {
		return b.fail("report", err)
	}

	var out strings.Builder
	fmt.Fprintf(&out, "С %s\n\n", from_.Format("2 January"))
	fmt.Fprintf(&out, "Потрачено: %s\n", formatMoney(report.Spent.Amount(), report.Spent.CurrencyCode()))
	fmt.Fprintf(&out, "Получено: %s\n", formatMoney(report.Earned.Amount(), report.Earned.CurrencyCode()))

	if len(report.ByCategory) > 0 {
		out.WriteString("\n")
		for _, line := range report.ByCategory {
			fmt.Fprintf(&out, "%s — %s\n", line.Name, formatMoney(line.Total.Amount(), line.Total.CurrencyCode()))
		}
	}
	return strings.TrimRight(out.String(), "\n")
}

func (b *Bot) transfer(ctx context.Context, from *User, args []string) string {
	if len(args) != 3 {
		return "Нужно так: /transfer 30000 наличные карта"
	}

	amount, err := decimal.NewFromString(strings.Replace(args[0], ",", ".", 1))
	if err != nil || !amount.IsPositive() {
		return "Сумма должна быть числом больше нуля."
	}

	fromAcc, err := b.accountNamed(ctx, from.ID, args[1])
	if err != nil {
		return b.fail("transfer", err)
	}
	toAcc, err := b.accountNamed(ctx, from.ID, args[2])
	if err != nil {
		return b.fail("transfer", err)
	}

	if _, err := b.service.Transfer(ctx, app.TransferParams{
		TelegramID:    from.ID,
		FromAccountID: fromAcc.ID(),
		ToAccountID:   toAcc.ID(),
		Amount:        amount,
	}); err != nil {
		return b.fail("transfer", err)
	}

	return fmt.Sprintf("Перевёл %s: %s → %s",
		formatMoney(amount, fromAcc.Balance().CurrencyCode()), fromAcc.Name(), toAcc.Name())
}

func (b *Bot) undo(ctx context.Context, from *User) string {
	undone, err := b.service.UndoLast(ctx, from.ID)
	if errors.Is(err, app.ErrNothingToUndo) {
		return "Отменять нечего."
	}
	if err != nil {
		return b.fail("undo", err)
	}

	return fmt.Sprintf("Отменил: %s %s",
		transactionWord(undone.Type()),
		formatMoney(undone.Amount().Amount(), undone.Amount().CurrencyCode()))
}

func (b *Bot) record(ctx context.Context, from *User, entry Entry) string {
	params := app.RecordParams{
		TelegramID: from.ID,
		Amount:     entry.Amount,
		Category:   entry.Category,
		Comment:    entry.Comment,
	}

	if entry.Account != "" {
		acc, err := b.accountNamed(ctx, from.ID, entry.Account)
		if err != nil {
			return b.fail("record", err)
		}
		params.AccountID = acc.ID()
	}

	var (
		recorded *transaction.Transaction
		err      error
	)
	if entry.Income {
		recorded, err = b.service.RecordIncome(ctx, params)
	} else {
		recorded, err = b.service.RecordExpense(ctx, params)
	}
	if err != nil {
		return b.fail("record", err)
	}

	accounts, err := b.service.ListAccounts(ctx, from.ID)
	if err != nil {
		return b.fail("record", err)
	}

	amount := formatMoney(recorded.Amount().Amount(), recorded.Amount().CurrencyCode())
	balance := ""
	for _, acc := range accounts {
		if acc.ID() == recorded.FromAccountID() || acc.ID() == recorded.ToAccountID() {
			balance = fmt.Sprintf("\n%s: %s", acc.Name(), formatMoney(acc.Balance().Amount(), acc.Balance().CurrencyCode()))
		}
	}

	if entry.Income {
		return fmt.Sprintf("+%s%s", amount, balance)
	}
	return fmt.Sprintf("−%s%s", amount, balance)
}

// accountNamed matches what a person types against account names, so /transfer
// takes "карта" rather than a uuid nobody can read off a phone.
func (b *Bot) accountNamed(ctx context.Context, telegramID int64, name string) (*account.Account, error) {
	accounts, err := b.service.ListAccounts(ctx, telegramID)
	if err != nil {
		return nil, err
	}

	for _, acc := range accounts {
		if strings.EqualFold(acc.Name(), name) {
			return acc, nil
		}
	}
	for _, acc := range accounts {
		if strings.HasPrefix(strings.ToLower(acc.Name()), strings.ToLower(name)) {
			return acc, nil
		}
	}
	return nil, app.ErrAccountNotFound
}
