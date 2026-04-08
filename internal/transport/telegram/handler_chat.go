package telegram

import (
	"context"
	"strings"

	"github.com/dsnikitin/sowhat/internal/pkg/errx"
	"github.com/dsnikitin/sowhat/internal/pkg/logger"
	"github.com/dsnikitin/sowhat/internal/transport/telegram/message"
	"github.com/pkg/errors"
	"gopkg.in/telebot.v3"
)

type ChatService interface {
	NewChat(ctx context.Context, userID int64, query string) (string, error)
	ContinueChat(ctx context.Context, userID int64, query string) (string, error)
}

// новый разговор
func (h *Bot) OnChat(botCtx telebot.Context) error {
	ctx, userID, err := getContextAndUserID(botCtx)
	if err != nil {
		logger.Log.Errorw("Failed to get context and user_id", "error", err.Error(), "telegram_user_id", botCtx.Sender().ID)
		return botCtx.Send(message.OperationFailed, telebot.ModeMarkdown)
	}

	args := botCtx.Args()
	if len(args) == 0 {
		return botCtx.Send(message.EmptyChatQuery, telebot.ModeMarkdown)
	}

	answer, err := h.service.NewChat(ctx, userID, strings.Join(args, " "))
	if err != nil {
		switch {
		case errors.Is(err, errx.ErrTooLarge):
			return botCtx.Send(message.TooLargeChat, telebot.ModeMarkdown)
		case errors.Is(err, errx.ErrNoFilesForQuestion):
			return botCtx.Send(message.NoFilesForQuestion, telebot.ModeMarkdown)
		case errors.Is(err, context.DeadlineExceeded):
			logger.Log.Warnw("Failed to get meeting", "error", err.Error(), "user_id", userID)
			return botCtx.Send(message.TooBusy, telebot.ModeMarkdown)
		default:
			logger.Log.Errorw("Failed to get meeting", "error", err.Error(), "user_id", userID)
			return botCtx.Send(message.OperationFailed, telebot.ModeMarkdown)
		}
	}

	return botCtx.Send("🤖 "+answer, telebot.ModeMarkdown)
}

// продолжение разговора
func (h *Bot) OnText(botCtx telebot.Context) error {
	ctx, userID, err := getContextAndUserID(botCtx)
	if err != nil {
		logger.Log.Errorw("Failed to get context and user_id", "error", err.Error(), "telegram_user_id", botCtx.Sender().ID)
		return botCtx.Send(message.OperationFailed, telebot.ModeMarkdown)
	}

	answer, err := h.service.ContinueChat(ctx, userID, botCtx.Message().Text)
	if err != nil {
		switch {
		case errors.Is(err, errx.ErrTooLarge):
			return botCtx.Send(message.TooLargeChat, telebot.ModeMarkdown)
		case errors.Is(err, errx.ErrNoFilesForQuestion):
			return botCtx.Send(message.NoFilesForQuestion, telebot.ModeMarkdown)
		case errors.Is(err, context.DeadlineExceeded):
			logger.Log.Warnw("Failed to get meeting", "error", err.Error(), "user_id", userID)
			return botCtx.Send(message.TooBusy, telebot.ModeMarkdown)
		default:
			logger.Log.Errorw("Failed to get meeting", "error", err.Error(), "user_id", userID)
			return botCtx.Send(message.OperationFailed, telebot.ModeMarkdown)
		}
	}

	return botCtx.Send("🤖 "+answer, telebot.ModeMarkdown)
}
