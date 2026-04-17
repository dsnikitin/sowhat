package telegram

import (
	"github.com/dsnikitin/sowhat/internal/transport/telegram/message"
	"gopkg.in/telebot.v3"
)

func (b *Bot) OnHelp(botCtx telebot.Context) error {
	return botCtx.Send(message.Help, telebot.ModeMarkdown)
}
