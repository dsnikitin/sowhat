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
