package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/tomq29/ourmoney/internal/domain/common"
)

type Config struct {
	DatabaseURL   string
	TelegramToken string
	Allowed       []int64
	Currency      common.CurrencyState
	LedgerName    string
	LogLevel      slog.Level
}

func Load() (Config, error) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		return Config{}, fmt.Errorf("TELEGRAM_TOKEN is required; get one from @BotFather")
	}

	allowed, err := ids("ALLOWED_TELEGRAM_IDS")
	if err != nil {
		return Config{}, err
	}

	currencyCode := envString("CURRENCY", "kzt")
	currency, ok := common.CurrencyConverterFromString[strings.ToLower(currencyCode)]
	if !ok {
		return Config{}, fmt.Errorf("CURRENCY: %q is not one of kzt, rub, usd, usdt", currencyCode)
	}

	var level slog.Level
	if raw := os.Getenv("LOG_LEVEL"); raw != "" {
		if err := level.UnmarshalText([]byte(raw)); err != nil {
			return Config{}, fmt.Errorf("LOG_LEVEL: %w", err)
		}
	}

	return Config{
		DatabaseURL:   url,
		TelegramToken: token,
		Allowed:       allowed,
		Currency:      currency,
		LedgerName:    envString("LEDGER_NAME", "Наш бюджет"),
		LogLevel:      level,
	}, nil
}

// ids reads a comma-separated allow list. Empty means the bot is open to
// whoever finds it, which is only sensible while nobody knows its name.
func ids(key string) ([]int64, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return nil, nil
	}

	var out []int64
	for _, field := range strings.Split(raw, ",") {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}

		id, err := strconv.ParseInt(field, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%s: %q is not a telegram id", key, field)
		}
		out = append(out, id)
	}
	return out, nil
}

func envString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
