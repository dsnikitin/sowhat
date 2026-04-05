package middleware

import (
	"context"
	"time"

	"github.com/dsnikitin/sowhat/internal/transport/telegram/consts/ctxkey"
	"gopkg.in/telebot.v3"
)

const ContextKey = "request_context"

// ContextContext создает контекст с таймаутом и сохраняет его в tele.Context
func Context(appContext context.Context, timeout time.Duration) telebot.MiddlewareFunc {
	return func(next telebot.HandlerFunc) telebot.HandlerFunc {
		return func(botCtx telebot.Context) error {
			ctx, cancel := context.WithTimeout(appContext, timeout)
			defer cancel()

			return next(ctxkey.SetContext(botCtx, ctx))
		}
	}
}
