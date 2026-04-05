package middleware

import (
	"context"
	"errors"
	"strconv"

	"github.com/dsnikitin/sowhat/consts/platform"
	"github.com/dsnikitin/sowhat/internal/pkg/errx"
	"github.com/dsnikitin/sowhat/internal/pkg/logger"
	"github.com/dsnikitin/sowhat/internal/transport/telegram/consts/ctxkey"
	"github.com/dsnikitin/sowhat/internal/transport/telegram/message"
	"gopkg.in/telebot.v3"
)

type IdentityService interface {
	IdentityUser(ctx context.Context, platform platform.Type, externalUserID string) (int64, error)
}

func Identity(service IdentityService) telebot.MiddlewareFunc {
	return func(next telebot.HandlerFunc) telebot.HandlerFunc {
		return func(botCtx telebot.Context) error {
			ctx, ok := ctxkey.GetContext(botCtx)
			if !ok {
				logger.Log.Errorw("Context not found in telegram request")
				return botCtx.Send(message.OperationFailed, telebot.ModeMarkdown)
			}

			sender := botCtx.Sender()
			if sender == nil {
				logger.Log.Infow("Sender is nil in telegram request")
				return botCtx.Send(message.OperationFailed, telebot.ModeMarkdown)
			}

			userID, err := service.IdentityUser(ctx, platform.Telegram, strconv.FormatInt(sender.ID, 10))
			if err != nil {
				switch {
				case errors.Is(err, errx.ErrNotFound):
					return botCtx.Send(message.IdentificationFailed, telebot.ModeMarkdown)
				case errors.Is(err, context.DeadlineExceeded):
					logger.Log.Warnw("Failed to identity user", "error", err.Error(), "telegram_user_id", sender.ID)
					return botCtx.Send(message.OperationTimeout, telebot.ModeMarkdown)
				default:
					logger.Log.Errorw("Failed to identity user", "error", err.Error())
					return botCtx.Send(message.OperationFailed, telebot.ModeMarkdown)
				}
			}

			return next(ctxkey.SetUserID(botCtx, userID))
		}
	}
}
