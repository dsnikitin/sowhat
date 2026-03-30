package handler

import "gopkg.in/telebot.v3"

type ChatUseCases struct {
	AnswerQuestionUseCase
}

type AnswerQuestionUseCase interface {
	Answer()
}

func (h *Handler) OnChat(botCtx telebot.Context) error {
	return botCtx.Send("On Chat!")
}

func (h *Handler) OnText(botCtx telebot.Context) error {
	return botCtx.Send("On Text!")
}
