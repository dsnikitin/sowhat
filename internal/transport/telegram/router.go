package telegram

import (
	"context"

	"github.com/dsnikitin/sowhat/internal/pkg/logger"
	"github.com/dsnikitin/sowhat/internal/transport/telegram/handler"
	"github.com/dsnikitin/sowhat/internal/transport/telegram/middleware"

	"gopkg.in/telebot.v3"
	telebotmiddleware "gopkg.in/telebot.v3/middleware"
)

func (b *Bot) router(appCtx context.Context, cfg *Config, h *handler.Handler, s middleware.IdentityService) {
	b.Use(
		middleware.Logger(logger.Log),
		telebotmiddleware.AutoRespond(),
		middleware.Context(appCtx, cfg.RequestTimeout),
	)

	b.Handle("/start", h.OnStart)
	b.Handle("/help", h.OnHelp)

	protected := b.Group()
	protected.Use(middleware.Identity(s))

	// обработка встреч
	protected.Handle("/get", h.OnGet)
	protected.Handle("/list", h.OnList)
	protected.Handle("/find", h.OnFind)
	protected.Handle(telebot.OnVoice, h.OnVoice)
	protected.Handle(telebot.OnAudio, h.OnAudio)

	// обработка вопросов
	protected.Handle("/chat", h.OnChat)
	protected.Handle(telebot.OnText, h.OnText)
}
