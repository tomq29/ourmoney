package postgres

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/tomq29/ourmoney/internal/domain/account"
	"github.com/tomq29/ourmoney/internal/domain/common"
	"github.com/tomq29/ourmoney/internal/domain/ledger"
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

	if err := Migrate(url); err != nil {
		return 0, err
	}

	testPool, err = NewPool(ctx, url)
	if err != nil {
		return 0, err
	}
	defer testPool.Close()

	return m.Run(), nil
}

func reset(t *testing.T) {
	t.Helper()
	if testPool == nil {
		t.Skip("integration test needs docker; running with -short")
	}

	// Everything hangs off ledgers, so one delete empties the database.
	if _, err := testPool.Exec(context.Background(), `DELETE FROM ledgers`); err != nil {
		t.Fatalf("reset: %v", err)
	}
}

func kzt(t *testing.T) common.Currency {
	t.Helper()

	c, err := common.NewCurrency(common.CurrencyKZT)
	if err != nil {
		t.Fatalf("currency: %v", err)
	}
	return c
}

func money(t *testing.T, amount int64) common.Money {
	t.Helper()

	m, err := common.NewMoney(decimal.NewFromInt(amount), common.CurrencyKZT)
	if err != nil {
		t.Fatalf("money: %v", err)
	}
	return m
}

// seedLedger creates a ledger owned by telegram user 1 and returns it.
func seedLedger(t *testing.T) *ledger.Ledger {
	t.Helper()

	owner, err := ledger.NewMember(1, "tom", ledger.RoleOwner, time.Now().UTC().Truncate(time.Millisecond))
	if err != nil {
		t.Fatalf("member: %v", err)
	}

	l, err := ledger.NewLedger(uuid.NewString(), "Home", kzt(t), owner)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	if err := NewLedgerRepo(testPool).Create(context.Background(), l); err != nil {
		t.Fatalf("create ledger: %v", err)
	}
	return l
}

func seedAccount(t *testing.T, ledgerID, name string, balance int64) *account.Account {
	t.Helper()

	acc, err := account.NewAccount(uuid.NewString(), ledgerID, name, money(t, balance), account.Cash)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	if err := NewAccountRepo(testPool).Create(context.Background(), acc); err != nil {
		t.Fatalf("create account: %v", err)
	}
	return acc
}
