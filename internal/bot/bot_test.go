package bot_test

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/tomq29/ourmoney/internal/app"
	"github.com/tomq29/ourmoney/internal/bot"
	"github.com/tomq29/ourmoney/internal/domain/common"
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

// fakeTelegram stands in for the Bot API: it hands the bot a fixed script of
// messages and records what it answers.
type fakeTelegram struct {
	mu       sync.Mutex
	updates  []bot.Update
	replies  []string
	drained  chan struct{}
	finished bool
}

func newFakeTelegram(messages ...string) *fakeTelegram {
	f := &fakeTelegram{drained: make(chan struct{})}
	for i, text := range messages {
		f.updates = append(f.updates, bot.Update{
			UpdateID: int64(i + 1),
			Message: &bot.Message{
				MessageID: int64(i + 1),
				From:      &bot.User{ID: 7, FirstName: "Tom"},
				Chat:      &bot.Chat{ID: 7},
				Text:      text,
			},
		})
	}
	return f
}

func (f *fakeTelegram) GetUpdates(ctx context.Context, offset int64, _ time.Duration) ([]bot.Update, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var pending []bot.Update
	for _, u := range f.updates {
		if u.UpdateID >= offset {
			pending = append(pending, u)
		}
	}

	if len(pending) == 0 {
		if !f.finished {
			f.finished = true
			close(f.drained)
		}
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return pending, nil
}

func (f *fakeTelegram) Send(_ context.Context, _ int64, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.replies = append(f.replies, text)
	return nil
}

func (f *fakeTelegram) sent() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.replies...)
}

// converse runs the bot over a script and returns everything it said back.
func converse(t *testing.T, messages ...string) []string {
	t.Helper()
	if testPool == nil {
		t.Skip("integration test needs docker; running with -short")
	}

	if _, err := testPool.Exec(context.Background(), `DELETE FROM ledgers`); err != nil {
		t.Fatalf("reset: %v", err)
	}

	service := app.NewService(app.Deps{
		Tx:           postgres.NewTxManager(testPool),
		Ledgers:      postgres.NewLedgerRepo(testPool),
		Invites:      postgres.NewInviteRepo(testPool),
		Accounts:     postgres.NewAccountRepo(testPool),
		Categories:   postgres.NewCategoryRepo(testPool),
		Transactions: postgres.NewTransactionRepo(testPool),
	})

	api := newFakeTelegram(messages...)
	b := bot.New(api, service, bot.Config{
		Currency:   common.CurrencyKZT,
		LedgerName: "Наш бюджет",
		PollWait:   time.Millisecond,
	}, slog.New(slog.DiscardHandler))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- b.Run(ctx) }()

	select {
	case <-api.drained:
	case <-ctx.Done():
		t.Fatal("the bot never worked through the script")
	}

	cancel()
	<-done

	return api.sent()
}

func TestTheWholeConversation(t *testing.T) {
	replies := converse(t,
		"/start",
		"+200000 зарплата",
		"-5000 такси в аэропорт",
		"-1.5к кофе",
		"/balance",
		"/report",
		"/undo",
		"/balance",
	)

	if len(replies) != 8 {
		t.Fatalf("replies = %d, want 8:\n%s", len(replies), strings.Join(replies, "\n---\n"))
	}

	if !strings.Contains(replies[0], "Бюджет заведён") {
		t.Errorf("start said: %s", replies[0])
	}
	if !strings.Contains(replies[1], "200 000 ₸") {
		t.Errorf("income said: %s", replies[1])
	}
	if !strings.Contains(replies[2], "5 000 ₸") || !strings.Contains(replies[2], "195 000 ₸") {
		t.Errorf("expense said: %s", replies[2])
	}
	if !strings.Contains(replies[3], "1 500 ₸") {
		t.Errorf("the thousands suffix said: %s", replies[3])
	}
	if !strings.Contains(replies[4], "193 500 ₸") {
		t.Errorf("balance said: %s", replies[4])
	}

	report := replies[5]
	for _, want := range []string{"Потрачено: 6 500 ₸", "Получено: 200 000 ₸", "такси", "кофе"} {
		if !strings.Contains(report, want) {
			t.Errorf("report is missing %q:\n%s", want, report)
		}
	}

	if !strings.Contains(replies[6], "Отменил") {
		t.Errorf("undo said: %s", replies[6])
	}
	if !strings.Contains(replies[7], "195 000 ₸") {
		t.Errorf("balance after undo said: %s", replies[7])
	}
}

func TestBotAsksForStartFirst(t *testing.T) {
	replies := converse(t, "-5000 еда")

	if len(replies) != 1 || !strings.Contains(replies[0], "/start") {
		t.Fatalf("replies = %v", replies)
	}
}

func TestBotExplainsItselfOnNonsense(t *testing.T) {
	replies := converse(t, "/start", "привет, как дела")

	if len(replies) != 2 || !strings.Contains(replies[1], "Запишите трату") {
		t.Fatalf("replies = %v", replies)
	}
}

func TestBotRefusesToOverspend(t *testing.T) {
	replies := converse(t, "/start", "-5000 еда")

	if len(replies) != 2 || !strings.Contains(replies[1], "столько нет") {
		t.Fatalf("replies = %v", replies)
	}
}

func TestBotTransfersBetweenNamedAccounts(t *testing.T) {
	replies := converse(t,
		"/start",
		"+100000 зарплата",
		"/transfer 30000 наличные наличные",
	)

	// Naming the same account twice is the domain's problem, not the parser's.
	if len(replies) != 3 {
		t.Fatalf("replies = %v", replies)
	}
	if strings.Contains(replies[2], "Перевёл") {
		t.Errorf("a transfer to the same account went through: %s", replies[2])
	}
}

func TestBotHandsOutAnInvite(t *testing.T) {
	replies := converse(t, "/start", "/invite")

	if len(replies) != 2 || !strings.Contains(replies[1], "/join ") {
		t.Fatalf("replies = %v", replies)
	}
}
