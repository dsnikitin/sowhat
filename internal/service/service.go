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
	*ChatService
}

func New(appCtx context.Context, cfg *TranscriptionConfig, r Repository, t Transcriber, sum Summarizer, ch Chatter) *Service {
	s := &Service{
		UserService:          NewUserService(r),
		TranscriptionService: NewTranscriptionService(appCtx, cfg, t, sum, r),
		ChatService:          NewChatService(ch, r),
	}

	s.MeetingService = NewMeetingService(s.TranscriptionService, r)

	return s
}
