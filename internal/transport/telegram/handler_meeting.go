package telegram

import (
	"context"
	"fmt"
	"iter"
	"strconv"
	"strings"

	"github.com/dsnikitin/sowhat/internal/models"
	"github.com/dsnikitin/sowhat/internal/pkg/errx"
	"github.com/dsnikitin/sowhat/internal/pkg/logger"
	"github.com/dsnikitin/sowhat/internal/transport/telegram/message"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gopkg.in/telebot.v3"
)

type MeetingService interface {
	RegisterMeeting(ctx context.Context, userID int64, file models.MeetingFile, subscriberID uuid.UUID) (int64, error)
	GetMeeting(ctx context.Context, userID, meetingID int64) (models.Meeting, error)
	ListMeetings(ctx context.Context, userID int64, limit, offset int) iter.Seq2[models.MeetingWithTotal, error]
	FindMeetings(ctx context.Context, userID int64, query string, limit, offset int) iter.Seq2[models.MeetingWithTotal, error]
}

func (b *Bot) OnGet(botCtx telebot.Context) error {
	ctx, userID, err := getContextAndUserID(botCtx)
	if err != nil {
		logger.Log.Errorw("Failed to get context and user_id", "error", err.Error(), "telegram_user_id", botCtx.Sender().ID)
		return botCtx.Send(message.OperationFailed, telebot.ModeMarkdown)
	}

	args := botCtx.Args()
	if len(args) == 0 {
		return botCtx.Send(message.IncorrectMeetingID, telebot.ModeMarkdown)
	}

	if len(args) > 1 {
		return botCtx.Send(message.TooMuchArguments, telebot.ModeMarkdown)
	}

	meetingID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return botCtx.Send(message.IncorrectMeetingID, telebot.ModeMarkdown)
	}

	meeting, err := b.service.GetMeeting(ctx, userID, meetingID)
	if err != nil {
		switch {
		case errors.Is(err, errx.ErrNotFound):
			return botCtx.Send(message.MeetingNotFound, telebot.ModeMarkdown)
		case errors.Is(err, context.DeadlineExceeded):
			logger.Log.Warnw("Failed to get meeting", "error", err.Error(), "user_id", userID)
			return botCtx.Send(message.TooBusy, telebot.ModeMarkdown)
		default:
			logger.Log.Errorw("Failed to get meeting", "error", err.Error(), "user_id", userID)
			return botCtx.Send(message.OperationFailed, telebot.ModeMarkdown)
		}
	}

	msg := message.MeeetingWithTranscript(meeting, b.cfg.UI.DateFormat, b.cfg.UI.TranscriptMaxLength)
	return botCtx.Send(msg, telebot.ModeHTML)
}

func (b *Bot) OnList(botCtx telebot.Context) error {
	ctx, userID, err := getContextAndUserID(botCtx)
	if err != nil {
		logger.Log.Errorw("Failed to get context and user_id", "error", err.Error(), "telegram_user_id", botCtx.Sender().ID)
		return botCtx.Send(message.OperationFailed, telebot.ModeMarkdown)
	}

	args := botCtx.Args()
	if len(args) > 1 {
		return botCtx.Send(message.TooMuchArguments, telebot.ModeMarkdown)
	}

	offset := 0
	if len(args) == 1 {
		page, _ := strconv.Atoi(args[0])
		if page <= 0 {
			return botCtx.Send(message.IncorrectListPage, telebot.ModeMarkdown)
		}
		offset = (page - 1) * b.cfg.UI.MeetingsPerPage
	}

	iter := b.service.ListMeetings(ctx, userID, b.cfg.UI.MeetingsPerPage, offset)

	msg, err := message.MeetingsWithSummaryList(iter, b.cfg.UI.DateFormat, b.cfg.UI.SummaryMaxLength)
	if err != nil {
		switch {
		case errors.Is(err, errx.ErrEmptyList):
			return b.sendNoMeetings(botCtx, offset, message.NoMeetings)
		case errors.Is(err, context.DeadlineExceeded):
			logger.Log.Warnw("Failed to list meetings", "error", err.Error(), "user_id", userID)
			return botCtx.Send(message.TooBusy, telebot.ModeMarkdown)
		default:
			logger.Log.Errorw("Failed to list meetings", "error", err.Error(), "user_id", userID)
			return botCtx.Send(message.OperationFailed, telebot.ModeMarkdown)
		}
	}

	return botCtx.Send(msg, telebot.ModeMarkdown)
}

func (b *Bot) OnFind(botCtx telebot.Context) error {
	ctx, userID, err := getContextAndUserID(botCtx)
	if err != nil {
		logger.Log.Errorw("Failed to get context and user_id", "error", err.Error(), "telegram_user_id", botCtx.Sender().ID)
		return botCtx.Send(message.OperationFailed, telebot.ModeMarkdown)
	}

	args := botCtx.Args()
	argsCount := len(args)

	if argsCount < 1 {
		return botCtx.Send(message.EmptyFindQuery, telebot.ModeMarkdown)
	}

	offset := 0
	if argsCount > 1 {
		// если получится перевести в число, то считаем, что последний аргумент это номер страницы
		// если не получится, то offset останется 0 и последний аргумент - это тоже ключевое слово
		if page, err := strconv.Atoi(args[len(args)-1]); err == nil {
			if page <= 0 {
				return botCtx.Send(message.IncorrectListPage, telebot.ModeMarkdown)
			}
			offset = (page - 1) * b.cfg.UI.MeetingsPerPage
			argsCount--
		}
	}

	query := strings.Join(args[:argsCount], " ")
	iter := b.service.FindMeetings(ctx, userID, query, b.cfg.UI.MeetingsPerPage, offset)

	msg, err := message.MeetingsWithSummaryList(iter, b.cfg.UI.DateFormat, b.cfg.UI.SummaryMaxLength)
	if err != nil {
		switch {
		case errors.Is(err, errx.ErrEmptyList):
			return b.sendNoMeetings(botCtx, offset, message.MeetingsNotFound)
		case errors.Is(err, context.DeadlineExceeded):
			logger.Log.Warnw("Failed to find meetings", "error", err.Error(), "user_id", userID)
			return botCtx.Send(message.TooBusy, telebot.ModeMarkdown)
		default:
			logger.Log.Errorw("Failed to find meetings", "error", err.Error(), "user_id", userID)
			return botCtx.Send(message.OperationFailed, telebot.ModeMarkdown)
		}
	}

	return botCtx.Send(msg, telebot.ModeMarkdown)
}

func (b *Bot) OnVoice(botCtx telebot.Context) error {
	voice := botCtx.Message().Voice
	return b.registerMeeting(botCtx, &voice.File, voice.MIME)
}

func (b *Bot) OnAudio(botCtx telebot.Context) error {
	audio := botCtx.Message().Audio
	return b.registerMeeting(botCtx, &audio.File, audio.MIME)
}

func (b *Bot) registerMeeting(botCtx telebot.Context, teleFile *telebot.File, mime string) error {
	ctx, userID, err := getContextAndUserID(botCtx)
	if err != nil {
		logger.Log.Errorw("Failed to get context and user_id", "error", err.Error(), "telegram_user_id", botCtx.Sender().ID)
		return botCtx.Send(message.OperationFailed, telebot.ModeMarkdown)
	}

	fileRc, err := botCtx.Bot().File(teleFile)
	if err != nil {
		logger.Log.Errorw("Failed to get file from Telegram", "error", err.Error(), "user_id", userID)
		return botCtx.Send(message.OperationFailed, telebot.ModeMarkdown)
	}

	file := models.MeetingFile{
		Reader: fileRc,
		MIME:   mime,
		Size:   teleFile.FileSize,
	}

	meetingId, err := b.service.RegisterMeeting(ctx, userID, file, b.subscriberID)
	if err != nil {
		var ufErr *errx.ErrUnsupportedAudioFormat
		var usErr *errx.ErrUnsupportedFileSize

		switch {
		case errors.As(err, &ufErr):
			return botCtx.Send(fmt.Sprintf(message.UnsupportedAudioFormat, ufErr.SupportedFormats), telebot.ModeMarkdown)
		case errors.As(err, &usErr):
			return botCtx.Send(fmt.Sprintf(message.UnsupportedFileSize, usErr.MinSize, usErr.MaxSize), telebot.ModeMarkdown)
		case errors.Is(err, errx.ErrAllWorkersBusy):
			logger.Log.Warnw("Failed to register meeting", "error", err.Error(), "user_id", userID)
			return botCtx.Send(message.TooBusy, telebot.ModeMarkdown)
		case errors.Is(err, context.DeadlineExceeded):
			logger.Log.Warnw("Failed to register meeting", "error", err.Error(), "user_id", userID)
			return botCtx.Send(message.TooBusy, telebot.ModeMarkdown)
		default:
			logger.Log.Errorw("Failed to register meeting", "error", err.Error(), "user_id", userID)
			return botCtx.Send(message.OperationFailed, telebot.ModeMarkdown)
		}
	}

	return botCtx.Send(fmt.Sprintf(message.MeetingRegistered, meetingId), telebot.ModeMarkdown)
}

func (b *Bot) sendNoMeetings(botCtx telebot.Context, offset int, msgNoMeetings string) error {
	if offset == 0 {
		return botCtx.Send(msgNoMeetings, telebot.ModeMarkdown)
	}
	return botCtx.Send(message.NoMoreMeetings, telebot.ModeMarkdown)
}
