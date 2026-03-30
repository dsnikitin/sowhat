package handler

import (
	"context"
	"fmt"

	"github.com/dsnikitin/sowhat/internal/bot/telegram/consts"
	"github.com/dsnikitin/sowhat/internal/pkg/errx"
	"github.com/dsnikitin/sowhat/internal/pkg/logger"
	"github.com/pkg/errors"
	"gopkg.in/telebot.v3"
)

type UserUseCases struct {
	RegisterUserUseCase
}

type RegisterUserUseCase interface {
	Register(ctx context.Context, id int64, name string) error
}

func (h *Handler) OnStart(botCtx telebot.Context) error {
	ctx := context.Background() // TODO сделать глобальный контекст и от него создавать withTimeout

	user := botCtx.Sender()

	err := h.users.Register(ctx, user.ID, user.FirstName)
	if err != nil {
		if !errors.Is(err, errx.ErrAlreadyExists) {
			logger.Log.Error("Failed to register user", "error", err, "user_id", user.ID)
		}

		greeting := fmt.Sprintf("Рад снова видеть тебя, %s!\n%s", user.FirstName, consts.WelcomeBackMsg)
		return botCtx.Send(greeting, telebot.ModeMarkdown)
	}

	greeting := fmt.Sprintf("Привет, %s!\n%s", user.FirstName, consts.FirstStartMsg)
	return botCtx.Send(greeting, telebot.ModeMarkdown)
}
