package telegram

import "gopkg.in/telebot.v3"

type ChatService interface {
	Chat(query string) (string, error)
}

// новый разговор с AI
func (h *Bot) OnChat(botCtx telebot.Context) error {
	return botCtx.Send("On Chat!")
}

// продолжение разговора с AI
func (h *Bot) OnText(botCtx telebot.Context) error {
	return botCtx.Send("On Text!")
}

// func (h *ChatHandler)
