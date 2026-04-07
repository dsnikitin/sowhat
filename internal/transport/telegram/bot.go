package telegram

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/dsnikitin/sowhat/internal/models"
	"github.com/dsnikitin/sowhat/internal/pkg/errx"
	"github.com/dsnikitin/sowhat/internal/pkg/logger"
	"github.com/dsnikitin/sowhat/internal/transport/telegram/consts/ctxkey"
	"github.com/dsnikitin/sowhat/internal/transport/telegram/middleware"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"golang.org/x/net/proxy"
	"golang.org/x/sync/errgroup"
	"gopkg.in/telebot.v3"
)

type Service interface {
	UserService
	MeetingService
	ChatService
}

type Bot struct {
	*telebot.Bot
	appCtx       context.Context
	cfg          *Config
	subscriberID uuid.UUID
	service      Service
	outMsgs      chan (models.TranscriptionCompletedMsg)
	stopCh       chan struct{}
	eg           errgroup.Group
}

func New(
	appCtx context.Context, cfg *Config, s middleware.IdentityService, service Service,
) (*Bot, error) {
	logger.Log.Info("Connecting to Telegram API...")

	socksProxy := "127.0.0.1:9050" // Стандартный порт Tor

	dialer, err := proxy.SOCKS5("tcp", socksProxy, nil, proxy.Direct)
	if err != nil {
		log.Fatal("Ошибка создания прокси-диалера:", err)
	}

	// 2. Создаем HTTP-транспорт с прокси
	httpTransport := &http.Transport{
		Dial:                  dialer.Dial,
		TLSHandshakeTimeout:   30 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second,
	}

	// 3. Создаем HTTP-клиент с этим транспортом
	httpClient := &http.Client{
		Transport: httpTransport,
		Timeout:   90 * time.Second,
	}

	tbot, err := telebot.NewBot(telebot.Settings{
		Token:  cfg.AuthToken,
		Poller: &telebot.LongPoller{Timeout: cfg.PollerTimeout},
		Client: httpClient,
	})
	if err != nil {
		return nil, err
	}

	logger.Log.Info("Successfuly connected to Telegram API")

	bot := &Bot{
		appCtx:       appCtx,
		Bot:          tbot,
		cfg:          cfg,
		subscriberID: uuid.New(),
		service:      service,
		outMsgs:      make(chan models.TranscriptionCompletedMsg, 100),
		stopCh:       make(chan struct{}),
	}

	bot.router(appCtx, cfg, s)
	bot.eg.Go(bot.listen)

	return bot, nil
}

func (b *Bot) Notify(msg models.TranscriptionCompletedMsg) error {
	select {
	case b.outMsgs <- msg:
		return nil
	default:
		return errx.ErrAllWorkersBusy
	}
}

func (b *Bot) GetID() uuid.UUID {
	return b.subscriberID
}

func (b *Bot) Stop(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		close(b.stopCh)
		b.eg.Wait()

		b.Bot.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		select {
		case <-done:
		default:
			return errors.Wrap(ctx.Err(), "stop bot")
		}
	}

	return nil
}

func getContextAndUserID(botCtx telebot.Context) (context.Context, int64, error) {
	ctx, ok := ctxkey.GetContext(botCtx)
	if !ok {
		return nil, 0, errors.New("context not found")
	}

	userID, ok := ctxkey.GetUserID(botCtx)
	if !ok {
		return nil, 0, errors.New("userID not found")
	}

	return ctx, userID, nil
}
