package app_test

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/tomq29/ourmoney/internal/app"
	"github.com/tomq29/ourmoney/internal/domain/account"
	"github.com/tomq29/ourmoney/internal/domain/common"
	"github.com/tomq29/ourmoney/internal/domain/ledger"
	"github.com/tomq29/ourmoney/internal/infra/postgres"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Short() {
		os.Exit(m.Run())
	}

	code, err := runWithPostgres(m)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(code)
}

func runWithPostgres(m *testing.M) (int, error) {
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgres:17-alpine",
		tcpostgres.WithDatabase("ourmoney"),
		tcpostgres.WithUsername("ourmoney"),
		tcpostgres.WithPassword("ourmoney"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		return 0, fmt.Errorf("start postgres: %w", err)
	}
	defer func() { _ = testcontainers.TerminateContainer(container) }()

	url, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return 0, fmt.Errorf("connection string: %w", err)
	}
	if err := postgres.Migrate(url); err != nil {
		return 0, err
	}

	testPool, err = postgres.NewPool(ctx, url)
	if err != nil {
		return 0, err
	}
	defer testPool.Close()

	return m.Run(), nil
}

func newService(t *testing.T) *app.Service {
	t.Helper()
	if testPool == nil {
		t.Skip("integration test needs docker; running with -short")
	}

	if _, err := testPool.Exec(context.Background(), `DELETE FROM ledgers`); err != nil {
		t.Fatalf("reset: %v", err)
	}

	return app.NewService(app.Deps{
		Tx:           postgres.NewTxManager(testPool),
		Ledgers:      postgres.NewLedgerRepo(testPool),
		Invites:      postgres.NewInviteRepo(testPool),
		Accounts:     postgres.NewAccountRepo(testPool),
		Categories:   postgres.NewCategoryRepo(testPool),
		Transactions: postgres.NewTransactionRepo(testPool),
	})
}

func start(t *testing.T, s *app.Service, telegramID int64) *ledger.Ledger {
	t.Helper()

	l, err := s.Start(context.Background(), app.StartParams{
		TelegramID: telegramID,
		Name:       "tom",
		LedgerName: "Home",
		Currency:   common.CurrencyKZT,
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	return l
}

func accountNamed(t *testing.T, s *app.Service, telegramID int64, name string) *account.Account {
	t.Helper()

	accounts, err := s.ListAccounts(context.Background(), telegramID)
	if err != nil {
		t.Fatalf("list accounts: %v", err)
	}
	for _, acc := range accounts {
		if acc.Name() == name {
			return acc
		}
	}
	t.Fatalf("no account named %q", name)
	return nil
}

func amount(t *testing.T, acc *account.Account) int64 {
	t.Helper()
	return acc.Balance().Amount().IntPart()
}

// A new ledger arrives with an account already in it, so the first message a
// person sends can be an expense.
func TestStartCreatesALedgerWithAnAccount(t *testing.T) {
	s := newService(t)

	start(t, s, 1)

	accounts, err := s.ListAccounts(context.Background(), 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(accounts) != 1 || accounts[0].Name() != "Наличные" {
		t.Fatalf("accounts = %+v", accounts)
	}
}

func TestStartRefusesASecondLedger(t *testing.T) {
	s := newService(t)
	start(t, s, 1)

	_, err := s.Start(context.Background(), app.StartParams{
		TelegramID: 1, Name: "tom", LedgerName: "Other", Currency: common.CurrencyKZT,
	})
	if !errors.Is(err, app.ErrAlreadyInLedger) {
		t.Fatalf("err = %v, want %v", err, app.ErrAlreadyInLedger)
	}
}

func TestStartHonoursTheAllowList(t *testing.T) {
	s := newService(t)

	_, err := s.Start(context.Background(), app.StartParams{
		TelegramID: 7, Name: "stranger", LedgerName: "Theirs",
		Currency: common.CurrencyKZT, Allowed: []int64{1, 2},
	})
	if !errors.Is(err, app.ErrNotAllowed) {
		t.Fatalf("err = %v, want %v", err, app.ErrNotAllowed)
	}
}

func TestNothingWorksWithoutALedger(t *testing.T) {
	s := newService(t)
	ctx := context.Background()

	if _, err := s.ListAccounts(ctx, 99); !errors.Is(err, app.ErrNoLedger) {
		t.Errorf("list accounts: %v, want %v", err, app.ErrNoLedger)
	}
	if _, err := s.RecordExpense(ctx, app.RecordParams{TelegramID: 99, Amount: decimal.NewFromInt(1)}); !errors.Is(err, app.ErrNoLedger) {
		t.Errorf("record expense: %v, want %v", err, app.ErrNoLedger)
	}
}

func TestInviteAndJoinShareOneLedger(t *testing.T) {
	s := newService(t)
	ctx := context.Background()

	l := start(t, s, 1)

	invite, err := s.Invite(ctx, 1)
	if err != nil {
		t.Fatalf("invite: %v", err)
	}

	joined, err := s.Join(ctx, 2, "partner", invite.Code())
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if joined.ID() != l.ID() {
		t.Fatal("the partner joined a different ledger")
	}

	// And now they see the same accounts.
	theirs, err := s.ListAccounts(ctx, 2)
	if err != nil {
		t.Fatalf("list for the partner: %v", err)
	}
	if len(theirs) != 1 {
		t.Fatalf("accounts = %d, want 1", len(theirs))
	}
}

func TestAnInviteWorksOnce(t *testing.T) {
	s := newService(t)
	ctx := context.Background()

	start(t, s, 1)
	invite, err := s.Invite(ctx, 1)
	if err != nil {
		t.Fatalf("invite: %v", err)
	}

	if _, err := s.Join(ctx, 2, "partner", invite.Code()); err != nil {
		t.Fatalf("join: %v", err)
	}
	if _, err := s.Join(ctx, 3, "gatecrasher", invite.Code()); !errors.Is(err, ledger.ErrInviteUsed) {
		t.Fatalf("second join: %v, want %v", err, ledger.ErrInviteUsed)
	}
}

func TestUnknownInvite(t *testing.T) {
	s := newService(t)

	if _, err := s.Join(context.Background(), 2, "partner", "NOPE"); !errors.Is(err, app.ErrInviteNotFound) {
		t.Fatalf("err = %v, want %v", err, app.ErrInviteNotFound)
	}
}

func TestExpenseMovesTheBalanceAndNamesTheCategory(t *testing.T) {
	s := newService(t)
	ctx := context.Background()

	start(t, s, 1)
	if _, err := s.RecordIncome(ctx, app.RecordParams{
		TelegramID: 1, Amount: decimal.NewFromInt(200_000), Category: "Зарплата",
	}); err != nil {
		t.Fatalf("income: %v", err)
	}

	recorded, err := s.RecordExpense(ctx, app.RecordParams{
		TelegramID: 1, Amount: decimal.NewFromInt(5_000), Category: "Такси", Comment: "в аэропорт",
	})
	if err != nil {
		t.Fatalf("expense: %v", err)
	}
	if recorded.CreatedBy() != 1 {
		t.Errorf("author = %d, want 1", recorded.CreatedBy())
	}

	cash := accountNamed(t, s, 1, "Наличные")
	if got := amount(t, cash); got != 195_000 {
		t.Errorf("balance = %d, want 195000", got)
	}

	// The category was created by being used.
	report, err := s.Report(ctx, 1, time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	if len(report.ByCategory) != 1 || report.ByCategory[0].Name != "Такси" {
		t.Fatalf("report = %+v", report.ByCategory)
	}
	if report.Spent.Amount().IntPart() != 5_000 || report.Earned.Amount().IntPart() != 200_000 {
		t.Errorf("spent %s, earned %s", report.Spent.Amount(), report.Earned.Amount())
	}
}

func TestExpenseRefusesToOverdraw(t *testing.T) {
	s := newService(t)
	ctx := context.Background()
	start(t, s, 1)

	_, err := s.RecordExpense(ctx, app.RecordParams{TelegramID: 1, Amount: decimal.NewFromInt(1)})
	if !errors.Is(err, account.ErrInsufficientFunds) {
		t.Fatalf("err = %v, want %v", err, account.ErrInsufficientFunds)
	}

	// And the refusal left nothing behind.
	entries, err := s.ListTransactions(ctx, 1, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("transactions = %d, want none", len(entries))
	}
}

func TestTransferKeepsTheTotal(t *testing.T) {
	s := newService(t)
	ctx := context.Background()

	start(t, s, 1)
	if _, err := s.RecordIncome(ctx, app.RecordParams{TelegramID: 1, Amount: decimal.NewFromInt(100_000)}); err != nil {
		t.Fatalf("income: %v", err)
	}

	card, err := s.CreateAccount(ctx, app.CreateAccountParams{
		TelegramID: 1, Name: "Карта", Type: account.BankCard, InitialBalance: decimal.Zero,
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}

	cash := accountNamed(t, s, 1, "Наличные")
	if _, err := s.Transfer(ctx, app.TransferParams{
		TelegramID: 1, FromAccountID: cash.ID(), ToAccountID: card.ID(), Amount: decimal.NewFromInt(30_000),
	}); err != nil {
		t.Fatalf("transfer: %v", err)
	}

	if got := amount(t, accountNamed(t, s, 1, "Наличные")); got != 70_000 {
		t.Errorf("cash = %d, want 70000", got)
	}
	if got := amount(t, accountNamed(t, s, 1, "Карта")); got != 30_000 {
		t.Errorf("card = %d, want 30000", got)
	}

	total, err := s.Total(ctx, 1)
	if err != nil {
		t.Fatalf("total: %v", err)
	}
	if total.Amount().IntPart() != 100_000 {
		t.Errorf("total = %s, want 100000: a transfer must not create or destroy money", total.Amount())
	}
}

func TestUndoPutsTheMoneyBack(t *testing.T) {
	s := newService(t)
	ctx := context.Background()

	start(t, s, 1)
	if _, err := s.RecordIncome(ctx, app.RecordParams{TelegramID: 1, Amount: decimal.NewFromInt(50_000)}); err != nil {
		t.Fatalf("income: %v", err)
	}
	if _, err := s.RecordExpense(ctx, app.RecordParams{TelegramID: 1, Amount: decimal.NewFromInt(5_000), Category: "Еда"}); err != nil {
		t.Fatalf("expense: %v", err)
	}

	undone, err := s.UndoLast(ctx, 1)
	if err != nil {
		t.Fatalf("undo: %v", err)
	}
	if undone.Amount().Amount().IntPart() != 5_000 {
		t.Errorf("undid the wrong entry: %s", undone.Amount().Amount())
	}
	if got := amount(t, accountNamed(t, s, 1, "Наличные")); got != 50_000 {
		t.Errorf("balance = %d, want 50000", got)
	}

	entries, _ := s.ListTransactions(ctx, 1, 10)
	if len(entries) != 1 {
		t.Errorf("transactions = %d, want 1", len(entries))
	}

	// Undoing the income too empties the ledger.
	if _, err := s.UndoLast(ctx, 1); err != nil {
		t.Fatalf("second undo: %v", err)
	}
	if got := amount(t, accountNamed(t, s, 1, "Наличные")); got != 0 {
		t.Errorf("balance = %d, want 0", got)
	}
	if _, err := s.UndoLast(ctx, 1); !errors.Is(err, app.ErrNothingToUndo) {
		t.Errorf("third undo: %v, want %v", err, app.ErrNothingToUndo)
	}
}

// Two people moving money in opposite directions at the same time: the pair of
// accounts is always locked in the same order, so neither waits for the other.
func TestOppositeTransfersDoNotDeadlock(t *testing.T) {
	s := newService(t)
	ctx := context.Background()

	start(t, s, 1)
	if _, err := s.RecordIncome(ctx, app.RecordParams{TelegramID: 1, Amount: decimal.NewFromInt(100_000)}); err != nil {
		t.Fatalf("income: %v", err)
	}
	card, err := s.CreateAccount(ctx, app.CreateAccountParams{
		TelegramID: 1, Name: "Карта", Type: account.BankCard, InitialBalance: decimal.NewFromInt(100_000),
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	cash := accountNamed(t, s, 1, "Наличные")

	const rounds = 20
	var wg sync.WaitGroup
	errs := make(chan error, rounds*2)

	for range rounds {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, err := s.Transfer(ctx, app.TransferParams{
				TelegramID: 1, FromAccountID: cash.ID(), ToAccountID: card.ID(), Amount: decimal.NewFromInt(100),
			})
			errs <- err
		}()
		go func() {
			defer wg.Done()
			_, err := s.Transfer(ctx, app.TransferParams{
				TelegramID: 1, FromAccountID: card.ID(), ToAccountID: cash.ID(), Amount: decimal.NewFromInt(100),
			})
			errs <- err
		}()
	}

	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("transfer: %v", err)
		}
	}

	total, err := s.Total(ctx, 1)
	if err != nil {
		t.Fatalf("total: %v", err)
	}
	if total.Amount().IntPart() != 200_000 {
		t.Errorf("total = %s, want 200000", total.Amount())
	}
}
