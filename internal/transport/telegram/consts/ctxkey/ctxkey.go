package ctxkey

import (
	"context"

	"gopkg.in/telebot.v3"
)

const (
	userIDKey  string = "user_id"
	contextKey string = "context"
)

// GetUserID получает userID из контекста бота.
func GetUserID(botCtx telebot.Context) (int64, bool) {
	userID, ok := botCtx.Get(userIDKey).(int64)
	return userID, ok
}

// SetUserID добавляет userID в контекст бота.
func SetUserID(botCtx telebot.Context, userID int64) telebot.Context {
	botCtx.Set(userIDKey, userID)
	return botCtx
}

// GetContext получает context.Context из контекста бота.
func GetContext(botCtx telebot.Context) (context.Context, bool) {
	userID, ok := botCtx.Get(contextKey).(context.Context)
	return userID, ok
}

// SetContext добавляет context.Context в контекст бота.
func SetContext(botCtx telebot.Context, ctx context.Context) telebot.Context {
	botCtx.Set(contextKey, ctx)
	return botCtx
}
