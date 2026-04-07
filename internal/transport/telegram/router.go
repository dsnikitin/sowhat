package telegram

import (
	"context"

	"github.com/dsnikitin/sowhat/internal/pkg/logger"
	"github.com/dsnikitin/sowhat/internal/transport/telegram/middleware"

	"gopkg.in/telebot.v3"
	telebotmiddleware "gopkg.in/telebot.v3/middleware"
)

func (b *Bot) router(appCtx context.Context, cfg *Config, s middleware.IdentityService) {
	b.Use(
		middleware.Logger(logger.Log),
		telebotmiddleware.AutoRespond(),
		middleware.Context(appCtx, cfg.RequestTimeout),
	)

	b.Handle("/start", b.OnStart)
	b.Handle("/help", b.OnHelp)

	protected := b.Group()
	protected.Use(middleware.Identity(s))

	// обработка встреч
	protected.Handle("/get", b.OnGet)
	protected.Handle("/list", b.OnList)
	protected.Handle("/find", b.OnFind)
	protected.Handle(telebot.OnVoice, b.OnVoice)
	protected.Handle(telebot.OnAudio, b.OnAudio)

	// обработка вопросов
	protected.Handle("/chat", b.OnChat)
	protected.Handle(telebot.OnText, b.OnText)
}
