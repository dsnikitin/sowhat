package service

import (
	"context"
	"fmt"

	"github.com/dsnikitin/sowhat/internal/models"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type MeetingRepository interface {
	CreateMeeting(ctx context.Context, userID int64) (int64, error)
	GetMeeting(ctx context.Context, id, userID int64) (models.Meeting, error)
	ListMeetings(ctx context.Context, userID int64, limit, offset int) ([]models.Meeting, int, error)
	FindMeetings(ctx context.Context, userID int64, query string, limit, offset int) ([]models.Meeting, int, error)
}

type Transcription interface {
	AsyncTranscribe(userID int64, file models.File, subscriberID uuid.UUID) error
}

type MeetingService struct {
	t Transcription
	r MeetingRepository
}

func NewMeetingService(t Transcription, r MeetingRepository) *MeetingService {
	return &MeetingService{t: t, r: r}
}

func (s *MeetingService) RegisterMeeting(ctx context.Context, userID int64, file models.File, subscriberID uuid.UUID) (int64, error) {
	meetingID, err := s.r.CreateMeeting(ctx, userID)
	if err != nil {
		return 0, errors.Wrap(err, "create meeting")
	}
	fmt.Println("meetingID =", meetingID)

	file.MeetingID = meetingID
	if err = s.t.AsyncTranscribe(userID, file, subscriberID); err != nil {
		return 0, errors.Wrap(err, "async transcribe")
	}

	return meetingID, nil
}

func (s *MeetingService) GetMeeting(ctx context.Context, userID, meetingID int64) (models.Meeting, error) {
	return s.r.GetMeeting(ctx, meetingID, userID)
}

func (s *MeetingService) ListMeetings(ctx context.Context, userID int64, limit, offset int) ([]models.Meeting, int, error) {
	return s.r.ListMeetings(ctx, userID, limit, offset)
}

func (s *MeetingService) FindMeetings(ctx context.Context, userID int64, query string, limit, offset int) ([]models.Meeting, int, error) {
	return s.r.FindMeetings(ctx, userID, query, limit, offset)
}
