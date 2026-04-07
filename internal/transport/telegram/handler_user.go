package telegram

import (
	"context"
	"fmt"
	"strconv"

	"github.com/dsnikitin/sowhat/internal/consts/platform"
	"github.com/dsnikitin/sowhat/internal/models"
	"github.com/dsnikitin/sowhat/internal/pkg/errx"
	"github.com/dsnikitin/sowhat/internal/pkg/logger"
	"github.com/dsnikitin/sowhat/internal/transport/telegram/consts/ctxkey"
	"github.com/dsnikitin/sowhat/internal/transport/telegram/message"
	"github.com/pkg/errors"
	"gopkg.in/telebot.v3"
)

type UserService interface {
	RegisterUser(ctx context.Context, externalID, name string, pt platform.Type) error
	GetUserByID(ctx context.Context, userID int64, pt platform.Type) (models.User, error)
}

func (b *Bot) OnStart(botCtx telebot.Context) error {
	ctx, ok := ctxkey.GetContext(botCtx)
	if !ok {
		logger.Log.Errorw("Context not found in onstart handler", "telegram_user_id", botCtx.Sender().ID)
		botCtx.Send(message.OperationFailed, telebot.ModeMarkdown)
	}

	user := botCtx.Sender()

	err := b.service.RegisterUser(ctx, strconv.FormatInt(user.ID, 10), user.FirstName, platform.Telegram)
	if err != nil {
		switch {
		case errors.Is(err, errx.ErrAlreadyExists):
			return botCtx.Send(fmt.Sprintf(message.WelcomeBack, user.FirstName), telebot.ModeMarkdown)
		case errors.Is(err, context.DeadlineExceeded):
			logger.Log.Warnw("Failed to register user", "error", err.Error(), "telegram_user_id", user.ID)
			return botCtx.Send(message.TooBusy, telebot.ModeMarkdown)
		default:
			logger.Log.Errorw("Failed to register user", "error", err.Error(), "telegram_user_id", user.ID)
			return botCtx.Send(message.OperationFailed, telebot.ModeMarkdown)
		}
	}

	return botCtx.Send(fmt.Sprintf(message.Introduction, user.FirstName), telebot.ModeMarkdown)
}

func (b *Bot) listen() error {
	for {
		select {
		case <-b.stopCh:
			return nil
		case msg := <-b.outMsgs:
			ctx, cancel := context.WithTimeout(b.appCtx, b.cfg.RequestTimeout)
			defer cancel()

			user, err := b.service.GetUserByID(ctx, msg.UserID, platform.Telegram)
			if err != nil {
				switch {
				case errors.Is(err, context.DeadlineExceeded):
					logger.Log.Warnw("Failed to get user for out message", "error", err.Error(), "user_id", msg.UserID)
					continue
				default:
					logger.Log.Errorw("Failed to get user for out message", "error", err.Error(), "user_id", msg.UserID)
					continue
				}
			}

			telegramUserID, err := strconv.ParseInt(user.ExternalID, 10, 64)
			if err != nil {
				logger.Log.Warnw("Failed to parst telegram user_id to int64", "error", err.Error(), "user_id", user.ID)
			}

			text := fmt.Sprintf(message.MeettingTranscriptionCompleted, msg.MeetingID)
			if _, err := b.Send(&telebot.User{ID: telegramUserID}, text); err != nil {
				switch {
				case errors.Is(err, telebot.ErrChatNotFound):
					logger.Log.Warnw("Failed to send out message to user", "error", err.Error(), "user_id", user.ID)
				default:
					logger.Log.Errorw("Failed to send out message to user", "error", err.Error(), "user_id", user.ID)
				}
			}
		}
	}
}
