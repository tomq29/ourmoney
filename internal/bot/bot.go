package bot

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/tomq29/ourmoney/internal/app"
	"github.com/tomq29/ourmoney/internal/domain/common"
)

type sender interface {
	GetUpdates(ctx context.Context, offset int64, timeout time.Duration) ([]Update, error)
	Send(ctx context.Context, chatID int64, text string) error
}

type Config struct {
	Allowed    []int64
	Currency   common.CurrencyState
	LedgerName string
	PollWait   time.Duration
}

type Bot struct {
	api     sender
	service *app.Service
	cfg     Config
	log     *slog.Logger
	offset  int64
}

func New(api sender, service *app.Service, cfg Config, log *slog.Logger) *Bot {
	if cfg.PollWait == 0 {
		cfg.PollWait = 30 * time.Second
	}
	if cfg.LedgerName == "" {
		cfg.LedgerName = "Наш бюджет"
	}
	return &Bot{api: api, service: service, cfg: cfg, log: log}
}

// Run polls until the context is cancelled. Long polling means Telegram never
// has to reach us, so the bot works from a laptop behind NAT with no public
// address and no certificate.
func (b *Bot) Run(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}

		updates, err := b.api.GetUpdates(ctx, b.offset, b.cfg.PollWait)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			// Telegram or the network hiccuped. Pause, then keep going: a
			// tracker that dies on one failed poll is a tracker nobody trusts.
			b.log.Error("poll", "error", err)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(3 * time.Second):
			}
			continue
		}

		for _, u := range updates {
			b.offset = u.UpdateID + 1
			b.handle(ctx, u)
		}
	}
}

func (b *Bot) handle(ctx context.Context, u Update) {
	if u.Message == nil || u.Message.From == nil || u.Message.Chat == nil {
		return
	}

	reply := b.reply(ctx, u.Message)
	if reply == "" {
		return
	}

	if err := b.api.Send(ctx, u.Message.Chat.ID, reply); err != nil && ctx.Err() == nil {
		b.log.Error("send", "error", err, "chat", u.Message.Chat.ID)
	}
}

func (b *Bot) reply(ctx context.Context, m *Message) string {
	if cmd, ok := ParseCommand(m.Text); ok {
		return b.command(ctx, m.From, cmd)
	}

	entry, err := ParseEntry(m.Text)
	if errors.Is(err, ErrNotAnEntry) {
		return help
	}
	return b.record(ctx, m.From, entry)
}
