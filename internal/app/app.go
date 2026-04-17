package app

import (
	"context"
	"time"

	"github.com/dsnikitin/sowhat/internal/config"
	"github.com/dsnikitin/sowhat/internal/infrastructure/db/postgres"
	"github.com/dsnikitin/sowhat/internal/infrastructure/llm/gigachat"
	"github.com/dsnikitin/sowhat/internal/infrastructure/oauth"
	"github.com/dsnikitin/sowhat/internal/infrastructure/transcriber/salute"
	"github.com/dsnikitin/sowhat/internal/pkg/httpx"
	"github.com/dsnikitin/sowhat/internal/pkg/logger"
	"github.com/dsnikitin/sowhat/internal/repository"
	"github.com/dsnikitin/sowhat/internal/repository/adapter"
	"github.com/dsnikitin/sowhat/internal/service"
	"github.com/dsnikitin/sowhat/internal/transport/telegram"
)

type App struct {
	pgDB         *postgres.DB
	authorizer   *oauth.Authorizer
	saluteSpeech *salute.SaluteSpeech
	gigachat     *gigachat.GigaChat
	teleBot      *telegram.Bot
	service      *service.Service
	ctxCancel    context.CancelFunc
}

func New(cfg *config.Config) *App {
	appCtx, appCtxCancel := context.WithCancel(context.Background())

	pgDB, err := postgres.New(cfg.PgDB)
	if err != nil {
		logger.Log.Fatalw("Failed to init postgres db", "error", err.Error())
	}

	if err := pgDB.ApplyMigrations(); err != nil {
		logger.Log.Fatalw("Failed to apply migrations", "error", err.Error())
	}

	httpClient := httpx.NewClient()

	authorizer, err := oauth.New(appCtx, cfg.OAuth, httpClient)
	if err != nil {
		logger.Log.Fatalw("Failed to init authorizer", "error", err.Error())
	}

	saluteSpeech := salute.New(appCtx, cfg.SaluteSpeech, httpClient, authorizer)
	gigachat := gigachat.New(appCtx, cfg.GigaChat, httpClient, authorizer)
	r := repository.New(pgDB)
	txProvider := adapter.NewTranscriptorTxAdapter(r.TranscriptionRepository)
	s := service.New(appCtx, cfg.Transcription, r, txProvider, saluteSpeech, gigachat, gigachat, gigachat)

	telebot, err := telegram.New(appCtx, cfg.TeleBot, s, s)
	if err != nil {
		logger.Log.Fatalw("Failed to init telegram bot", "error", err.Error())
	}

	return &App{
		pgDB:         pgDB,
		authorizer:   authorizer,
		saluteSpeech: saluteSpeech,
		gigachat:     gigachat,
		teleBot:      telebot,
		service:      s,
		ctxCancel:    appCtxCancel,
	}
}

func (a *App) Run() {
	a.service.PublisherService.Subscribe(a.teleBot)
	a.service.TranscriptionService.RestartNotCompleted()

	a.teleBot.Start()
}

func (a *App) Shutdown() {
	a.ctxCancel()

	components := []struct {
		name    string
		timeout time.Duration
		stopFn  func(ctx context.Context) error
	}{
		{
			name:    "Telegram Bot",
			stopFn:  a.teleBot.Stop,
			timeout: time.Second * 30,
		},
		{
			name:    "Postgres DB",
			stopFn:  a.pgDB.Close,
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
