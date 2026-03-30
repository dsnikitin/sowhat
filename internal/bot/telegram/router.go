package telegram

import (
	"github.com/dsnikitin/sowhat/internal/bot/telegram/handler"
	"gopkg.in/telebot.v3"
)

func (b *Bot) initRouter(h *handler.Handler) {
	b.Handle("/start", h.OnStart)
	b.Handle("/help", h.OnHelp)

	// обработка встреч
	b.Handle("/list", h.OnList)
	b.Handle("/find", h.OnFind)
	b.Handle("/get", h.OnGet)
	b.Handle(telebot.OnVoice, h.OnVoice)
	b.Handle(telebot.OnAudio, h.OnAudio)

	// обработка вопросов
	b.Handle("/chat", h.OnChat)
	b.Handle(telebot.OnText, h.OnText)
}
