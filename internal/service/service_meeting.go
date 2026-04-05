package service

import (
	"context"
	"io"

	"github.com/dsnikitin/sowhat/internal/models"
	"github.com/google/uuid"
)

type MeetingRepository interface {
	CreateMeeting(ctx context.Context, userID int64) error
	GetMeeting(ctx context.Context, id, userID int64) (models.MeetingWithTranscript, error)
	ListMeetings(ctx context.Context, userID int64) ([]models.MeetingWithSummary, error)
	FindMeetings(ctx context.Context, userID int64, query string) ([]models.MeetingWithSummary, error)
}

type MeetingAI interface {
	UploadFile(file io.Reader) (uuid.UUID, error)
}

type Transcriber interface {
	UploadFile(file io.Reader) (uuid.UUID, error)
}

type MeetingService struct {
	ai MeetingAI
	t  Transcriber
	r  MeetingRepository
}

func NewMeetingService(ai MeetingAI, t Transcriber, r MeetingRepository) *MeetingService {
	return &MeetingService{ai: ai, t: t, r: r}
}

func (s *MeetingService) RegisterMeeting(ctx context.Context, userID int64) error {
	return s.r.CreateMeeting(ctx, userID)
}

func (s *MeetingService) GetMeeting(ctx context.Context, userID, meetingID int64) (models.MeetingWithTranscript, error) {
	return s.r.GetMeeting(ctx, meetingID, userID)
}

func (s *MeetingService) ListMeetings(ctx context.Context, userID int64) ([]models.MeetingWithSummary, error) {
	return s.r.ListMeetings(ctx, userID)
}

func (s *MeetingService) FindMeetings(ctx context.Context, userID int64, query string) ([]models.MeetingWithSummary, error) {
	return s.r.FindMeetings(ctx, userID, query)
}
