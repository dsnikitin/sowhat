package handler

import "gopkg.in/telebot.v3"

type ChatService interface {
	Chat(query string) (string, error)
}

type ChatHandler struct {
	service ChatService
}

func NewChatHandler(s ChatService) *ChatHandler {
	return &ChatHandler{service: s}
}

// новый разговор с AI
func (h *ChatHandler) OnChat(botCtx telebot.Context) error {
	return botCtx.Send("On Chat!")
}

// продолжение разговора с AI
func (h *ChatHandler) OnText(botCtx telebot.Context) error {
	return botCtx.Send("On Text!")
}
