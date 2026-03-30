package handler

import (
	"github.com/dsnikitin/sowhat/internal/models"
	"github.com/google/uuid"
	"gopkg.in/telebot.v3"
)

type MeetingUseCases struct {
	GetMeetingUseCase
	ListMeetingsUseCase
	FindMeetingsUseCase
}

type GetMeetingUseCase interface {
	GetMeeting(id uuid.UUID) (models.Meeting, error)
}

type ListMeetingsUseCase interface {
	ListMeetings() ([]models.Meeting, error)
}

type FindMeetingsUseCase interface {
	FindMeetings(query string) ([]models.Meeting, error)
}

func (h *Handler) OnGet(botCtx telebot.Context) error {
	return botCtx.Send("On Get!")
}

func (h *Handler) OnList(botCtx telebot.Context) error {
	return botCtx.Send("On List!")
}

func (h *Handler) OnFind(botCtx telebot.Context) error {
	return botCtx.Send("On Find!")
}

func (h *Handler) OnVoice(botCtx telebot.Context) error {
	return botCtx.Send("On Voice!")
}

func (h *Handler) OnAudio(botCtx telebot.Context) error {
	return botCtx.Send("On Audio!")
}
