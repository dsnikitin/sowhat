package service

import "context"

type Repository interface {
	UserRepository
	MeetingRepository
	TranscriptionRepository
	ChatRepository
}

type Service struct {
	*UserService
	*MeetingService
	*TranscriptionService
	*PublisherService
	*ChatService
}

func New(
	appCtx context.Context, cfg *TranscriptionConfig,
	r Repository, t Transcriber, sum Summarizer,
	chUp TranscriptUploader, ch Chatter,
) *Service {
	s := &Service{
		UserService:      NewUserService(r),
		PublisherService: NewPublisher(),
		ChatService:      NewChatService(ch, r),
	}
	s.TranscriptionService = NewTranscriptionService(appCtx, cfg, t, sum, chUp, s.PublisherService, r)
	s.MeetingService = NewMeetingService(s.TranscriptionService, r)

	return s
}
