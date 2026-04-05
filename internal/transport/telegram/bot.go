package telegram

import (
	"context"

	"github.com/dsnikitin/sowhat/internal/pkg/logger"
	"github.com/dsnikitin/sowhat/internal/transport/telegram/handler"
	"github.com/dsnikitin/sowhat/internal/transport/telegram/middleware"
	"github.com/pkg/errors"
	tele "gopkg.in/telebot.v3"
)

type Bot struct {
	*tele.Bot
}

func New(
	appCtx context.Context, cfg *Config, h *handler.Handler, s middleware.IdentityService,
) (*Bot, error) {
	logger.Log.Info("Connecting to Telegram API...")

	tbot, err := tele.NewBot(tele.Settings{
		Token:  cfg.AuthToken,
		Poller: &tele.LongPoller{Timeout: cfg.PollerTimeout},
	})

	logger.Log.Info("Successfuly connected to Telegram API")

	if err != nil {
		return nil, err
	}

	bot := &Bot{Bot: tbot}
	bot.router(appCtx, cfg, h, s)

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
