package main

import (
	"time"

	"github.com/dsnikitin/sowhatbot/internal/config"
	"github.com/dsnikitin/sowhatbot/internal/pkg/logger"
	tele "gopkg.in/telebot.v3"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		logger.Log.Fatalw("Failed to init config", "error", err.Error())
	}

	pref := tele.Settings{
		Token:  cfg.BotAuthToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := tele.NewBot(pref)
	if err != nil {
		logger.Log.Fatalw("Failed to init bot", "error", err.Error())
	}

	bot.Handle("/hello", func(c tele.Context) error {
		return c.Send("Hello!")
	})

	bot.Start()
}
