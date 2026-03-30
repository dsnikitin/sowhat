package handler

import (
	"github.com/dsnikitin/sowhat/internal/bot/telegram/consts"
	"gopkg.in/telebot.v3"
)

type UseCases struct {
	Users    UserUseCases
	Meetings MeetingUseCases
	Chat     ChatUseCases
}

type Handler struct {
	users    UserUseCases
	meetings MeetingUseCases
	chat     ChatUseCases
}

func New(uc *UseCases) *Handler {
	return &Handler{
		users:    uc.Users,
		meetings: uc.Meetings,
		chat:     uc.Chat,
	}
}

func (h *Handler) OnHelp(botCtx telebot.Context) error {
	return botCtx.Send(consts.HelpMsg, telebot.ModeMarkdown)
}
