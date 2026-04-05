package middleware

import (
	"time"

	"go.uber.org/zap"
	"gopkg.in/telebot.v3"
)

func Logger(logger *zap.SugaredLogger) telebot.MiddlewareFunc {
	return func(next telebot.HandlerFunc) telebot.HandlerFunc {
		return func(botCtx telebot.Context) error {
			start := time.Now()
			err := next(botCtx)

			msg := botCtx.Message()
			entities := botCtx.Entities()

			contentType := "text"
			switch {
			case len(entities) > 0 && entities[0].Type == telebot.EntityCommand:
				contentType = "command"
			case msg != nil && msg.Media() != nil:
				contentType = msg.Media().MediaType()
			}

			fields := []any{
				"telegram_user_id", botCtx.Sender().ID,
				"first_name", botCtx.Sender().FirstName,
				"username", botCtx.Sender().Username,
				"content_type", contentType,
				"duration", time.Since(start),
			}

			if contentType == "command" {
				fields = append(fields, "command", msg.Text)
			}

			if err != nil {
				fields = append(fields, "error", err.Error())
			}

			logger.Infow("Telegram bot request handled", fields...)
			return err
		}
	}
}
