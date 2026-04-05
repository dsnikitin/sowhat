package handler

import (
	"context"
	"strconv"
	"strings"

	"github.com/dsnikitin/sowhat/internal/models"
	"github.com/dsnikitin/sowhat/internal/pkg/errx"
	"github.com/dsnikitin/sowhat/internal/pkg/logger"
	"github.com/dsnikitin/sowhat/internal/transport/telegram/message"
	"github.com/pkg/errors"
	"gopkg.in/telebot.v3"
)

type MeetingService interface {
	RegisterMeeting(ctx context.Context, userID int64) error
	GetMeeting(ctx context.Context, userID, meetingID int64) (models.MeetingWithTranscript, error)
	ListMeetings(ctx context.Context, userID int64) ([]models.MeetingWithSummary, error)               // TODO нужна релизация пагинации
	FindMeetings(ctx context.Context, userID int64, query string) ([]models.MeetingWithSummary, error) // TODO нужна релизация пагинации
}

type MeetingHandler struct {
	cfg     *UIConfig
	service MeetingService
}

func NewMeetingHandler(cfg *UIConfig, s MeetingService) *MeetingHandler {
	return &MeetingHandler{cfg: cfg, service: s}
}

func (h *MeetingHandler) OnGet(botCtx telebot.Context) error {
	ctx, userID, err := getContextAndUserID(botCtx)
	if err != nil {
		logger.Log.Errorw("Failed to get context and user_id", "error", err.Error(), "telegram_user_id", botCtx.Sender().ID)
		botCtx.Send(message.OperationFailed, telebot.ModeMarkdown)
	}

	args := botCtx.Args()
	if len(args) != 1 {
		return botCtx.Send(message.EmptyOrTooMuchMeetingID, telebot.ModeMarkdown)
	}

	meetingID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return botCtx.Send(message.IncorrectMeetingID, telebot.ModeMarkdown)
	}

	meeting, err := h.service.GetMeeting(ctx, userID, meetingID)
	if err != nil {
		switch {
		case errors.Is(err, errx.ErrNotFound):
			return botCtx.Send(message.MeetingNotFound, telebot.ModeMarkdown)
		case errors.Is(err, context.DeadlineExceeded):
			logger.Log.Warnw("Failed to get meeting", "error", err.Error(), "user_id", userID)
			return botCtx.Send(message.OperationTimeout, telebot.ModeMarkdown)
		default:
			logger.Log.Errorw("Failed to get meeting", "error", err.Error(), "user_id", userID)
			return botCtx.Send(message.OperationFailed, telebot.ModeMarkdown)
		}
	}

	msg := message.MeeetingWithTranscript(meeting, h.cfg.DateFormat, h.cfg.TranscriptMaxLength)
	return botCtx.Send(msg, telebot.ModeMarkdown)
}

func (h *MeetingHandler) OnList(botCtx telebot.Context) error {
	ctx, userID, err := getContextAndUserID(botCtx)
	if err != nil {
		logger.Log.Errorw("Failed to get context and user_id", "error", err.Error(), "telegram_user_id", botCtx.Sender().ID)
		botCtx.Send(message.OperationFailed, telebot.ModeMarkdown)
	}

	meetings, err := h.service.ListMeetings(ctx, userID)
	if err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			logger.Log.Warnw("Failed to list meetings", "error", err.Error(), "user_id", userID)
			return botCtx.Send(message.OperationTimeout, telebot.ModeMarkdown)
		default:
			logger.Log.Errorw("Failed to list meetings", "error", err.Error(), "user_id", userID)
			return botCtx.Send(message.OperationFailed, telebot.ModeMarkdown)
		}
	}

	if len(meetings) == 0 {
		return botCtx.Send(message.NoMeetings, telebot.ModeMarkdown)
	}

	msg := message.MeetingsWithSummaryList(meetings, h.cfg.DateFormat, h.cfg.SummaryMaxLength)
	return botCtx.Send(msg, telebot.ModeMarkdown)
}

func (h *MeetingHandler) OnFind(botCtx telebot.Context) error {
	ctx, userID, err := getContextAndUserID(botCtx)
	if err != nil {
		logger.Log.Errorw("Failed to get context and user_id", "error", err.Error(), "telegram_user_id", botCtx.Sender().ID)
		botCtx.Send(message.OperationFailed, telebot.ModeMarkdown)
	}

	query := strings.TrimSpace(botCtx.Data())
	if query == "" {
		return botCtx.Send(message.EmptyFindQuery, telebot.ModeMarkdown)
	}

	meetings, err := h.service.FindMeetings(ctx, userID, query)
	if err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			logger.Log.Warnw("Failed to find meetings", "error", err.Error(), "user_id", userID)
			return botCtx.Send(message.OperationTimeout, telebot.ModeMarkdown)
		default:
			logger.Log.Errorw("Failed to find meetings", "error", err.Error(), "user_id", userID)
			return botCtx.Send(message.OperationFailed, telebot.ModeMarkdown)
		}
	}

	if len(meetings) == 0 {
		return botCtx.Send(message.NoMeetings, telebot.ModeMarkdown)
	}

	msg := message.MeetingsWithSummaryList(meetings, h.cfg.DateFormat, h.cfg.SummaryMaxLength)
	return botCtx.Send(msg, telebot.ModeMarkdown)
}

func (h *MeetingHandler) OnVoice(botCtx telebot.Context) error {
	_, _, err := getContextAndUserID(botCtx)
	if err != nil {
		logger.Log.Errorw("Failed to get context and user_id", "error", err.Error(), "telegram_user_id", botCtx.Sender().ID)
		botCtx.Send(message.OperationFailed, telebot.ModeMarkdown)
	}

	return botCtx.Send("On Voice!")
}

func (h *MeetingHandler) OnAudio(botCtx telebot.Context) error {
	_, _, err := getContextAndUserID(botCtx)
	if err != nil {
		logger.Log.Errorw("Failed to get context and user_id", "error", err.Error(), "telegram_user_id", botCtx.Sender().ID)
		botCtx.Send(message.OperationFailed, telebot.ModeMarkdown)
	}

	return botCtx.Send("On Audio!")
}
