package telegram

import (
	"context"
	"time"

	"github.com/dsnikitin/sowhat/internal/bot/telegram/handler"
	"github.com/pkg/errors"
	tele "gopkg.in/telebot.v3"
)

type Bot struct {
	*tele.Bot
}

type Config struct {
	AuthToken     string        `env:"AUTH_TOKEN" yaml:"-"`
	PollerTimeout time.Duration `env:"POLLER_TIMEOUT" yaml:"poller_timeout"`
}

func New(cfg *Config, h *handler.Handler) (*Bot, error) {
	tbot, err := tele.NewBot(tele.Settings{
		Token:  cfg.AuthToken,
		Poller: &tele.LongPoller{Timeout: cfg.PollerTimeout},
	})

	if err != nil {
		return nil, err
	}

	bot := &Bot{Bot: tbot}
	bot.initRouter(h)

	return bot, nil
}

func (b *Bot) Stop(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		b.Bot.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		select {
		case <-done:
		default:
			return errors.Wrap(ctx.Err(), "stop bot")
		}
	}

	return nil
}
