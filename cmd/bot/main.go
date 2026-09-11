package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	// The month a report covers depends on the household's timezone, and the
	// image has no system one to read: carry the database in the binary.
	_ "time/tzdata"

	"github.com/tomq29/ourmoney/internal/app"
	"github.com/tomq29/ourmoney/internal/bot"
	"github.com/tomq29/ourmoney/internal/config"
	"github.com/tomq29/ourmoney/internal/infra/postgres"
)

func main() {
	if err := run(); err != nil {
		slog.Error("bot stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := postgres.Migrate(cfg.DatabaseURL); err != nil {
		return err
	}

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	service := app.NewService(app.Deps{
		Tx:           postgres.NewTxManager(pool),
		Ledgers:      postgres.NewLedgerRepo(pool),
		Invites:      postgres.NewInviteRepo(pool),
		Accounts:     postgres.NewAccountRepo(pool),
		Categories:   postgres.NewCategoryRepo(pool),
		Transactions: postgres.NewTransactionRepo(pool),
	})

	telegram := bot.NewTelegram(cfg.TelegramToken)
	b := bot.New(telegram, service, bot.Config{
		Allowed:    cfg.Allowed,
		Currency:   cfg.Currency,
		LedgerName: cfg.LedgerName,
	}, log)

	log.Info("listening for updates", "allowed", len(cfg.Allowed), "currency", cfg.Currency)
	return b.Run(ctx)
}
