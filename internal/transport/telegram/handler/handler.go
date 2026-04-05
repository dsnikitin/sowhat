package handler

import (
	"context"

	"github.com/dsnikitin/sowhat/internal/transport/telegram/consts/ctxkey"
	"github.com/dsnikitin/sowhat/internal/transport/telegram/message"
	"github.com/pkg/errors"
	"gopkg.in/telebot.v3"
)

type Service interface {
	UserService
	MeetingService
	ChatService
}

type Handler struct {
	*UserHandler
	*MeetingHandler
	*ChatHandler
}

func New(cfg *UIConfig, s Service) *Handler {
	return &Handler{
		UserHandler:    NewUserHandler(s),
		MeetingHandler: NewMeetingHandler(cfg, s),
		ChatHandler:    NewChatHandler(s),
	}
}

func (h *Handler) OnHelp(botCtx telebot.Context) error {
	return botCtx.Send(message.Help, telebot.ModeMarkdown)
}

func getContextAndUserID(botCtx telebot.Context) (context.Context, int64, error) {
	ctx, ok := ctxkey.GetContext(botCtx)
	if !ok {
		return nil, 0, errors.New("context not found")
	}

	userID, ok := ctxkey.GetUserID(botCtx)
	if !ok {
		return nil, 0, errors.New("userID not found")
	}

	return ctx, userID, nil
}
