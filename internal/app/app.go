package app

import (
	"context"
	"time"

	"github.com/dsnikitin/sowhat/internal/bot/telegram"
	"github.com/dsnikitin/sowhat/internal/bot/telegram/handler"
	"github.com/dsnikitin/sowhat/internal/config"
	"github.com/dsnikitin/sowhat/internal/infra/db/postgres"
	"github.com/dsnikitin/sowhat/internal/pkg/logger"
	"github.com/dsnikitin/sowhat/internal/repository"
	"github.com/dsnikitin/sowhat/internal/service"
	"github.com/dsnikitin/sowhat/internal/usecase"
	"github.com/pkg/errors"
)

type App struct {
	TeleBot *telegram.Bot
	PgDB    *postgres.DB
}

func New(cfg *config.Config) *App {
	pgdb, err := postgres.New(cfg.PgDB)
	if err != nil {
		logger.Log.Fatalw("Failed to connect to postgres db", "error", err.Error())
	}

	if err := pgdb.ApplyMigrations(); err != nil {
		logger.Log.Fatalw("Failed to apply migrations", "error", err.Error())
	}

	tbot, err := initTelegramBot(cfg.TeleBot, pgdb)
	if err != nil {
		logger.Log.Fatalw("Failed to init telegram bot", "error", err.Error())
	}

	return &App{
		TeleBot: tbot,
		PgDB:    pgdb,
	}
}

func (a *App) Run() {
	a.TeleBot.Start()
}

func (a *App) Shutdown() {
	components := []struct {
		name    string
		timeout time.Duration
		stopFn  func(ctx context.Context) error
	}{
		{
			name:    "Telegram Bot",
			stopFn:  a.TeleBot.Stop,
			timeout: time.Second * 30,
		},
		{
			name:    "Postgres DB",
			stopFn:  a.PgDB.Close,
			timeout: time.Second * 30,
		},
	}

	for _, c := range components {
		ctx, cancel := context.WithTimeout(context.Background(), c.timeout)

		logger.Log.Infof("Stopping %s...", c.name)
		if err := c.stopFn(ctx); err != nil {
			logger.Log.Errorw("Failed to stop "+c.name, "error", err)
		} else {
			logger.Log.Infof("%s stopped gracefully", c.name)
		}

		cancel()
	}
}

func initTelegramBot(cfg *telegram.Config, db *postgres.DB) (*telegram.Bot, error) {
	repo := repository.NewUserRepository(db)
	userService := service.NewUserService(repo)

	botHandler := handler.New(&handler.UseCases{
		Users: handler.UserUseCases{
			RegisterUserUseCase: usecase.NewRegisterUserUseCase(userService),
		},
		Meetings: handler.MeetingUseCases{},
		Chat:     handler.ChatUseCases{},
	})

	bot, err := telegram.New(cfg, botHandler)
	return bot, errors.Wrap(err, "new bot")
}
