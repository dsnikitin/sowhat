package telegram

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/dsnikitin/sowhat/internal/consts/platform"
	"github.com/dsnikitin/sowhat/internal/models"
	"github.com/dsnikitin/sowhat/internal/pkg/errx"
	"github.com/dsnikitin/sowhat/internal/pkg/logger"
	"github.com/dsnikitin/sowhat/internal/transport/telegram/consts/ctxkey"
	"github.com/dsnikitin/sowhat/internal/transport/telegram/message"
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
	bot.eg.Go(bot.listenEvents)

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

func (b *Bot) listenEvents() error {
	for {
		select {
		case <-b.stopCh:
			return nil
		case msg := <-b.outMsgs:
			ctx, cancel := context.WithTimeout(b.appCtx, b.cfg.RequestTimeout)
			defer cancel()

			user, err := b.service.GetUserByID(ctx, msg.UserID, platform.Telegram)
			if err != nil {
				switch {
				case errors.Is(err, context.DeadlineExceeded):
					logger.Log.Warnw("Failed to get user for out message", "error", err.Error(), "user_id", msg.UserID)
					continue
				default:
					logger.Log.Errorw("Failed to get user for out message", "error", err.Error(), "user_id", msg.UserID)
					continue
				}
			}

			telegramUserID, err := strconv.ParseInt(user.ExternalID, 10, 64)
			if err != nil {
				logger.Log.Warnw("Failed to parst telegram user_id to int64", "error", err.Error(), "user_id", user.ID)
			}

			text := fmt.Sprintf(message.MeettingTranscriptionCompleted, msg.MeetingID)
			if _, err := b.Send(&telebot.User{ID: telegramUserID}, text); err != nil {
				switch {
				case errors.Is(err, telebot.ErrChatNotFound):
					logger.Log.Warnw("Failed to send out message to user", "error", err.Error(), "user_id", user.ID)
				default:
					logger.Log.Errorw("Failed to send out message to user", "error", err.Error(), "user_id", user.ID)
				}
			}
		}
	}
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
