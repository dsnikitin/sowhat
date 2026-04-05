package service

type Repository interface {
	UserRepository
	MeetingRepository
}

type AI interface {
	MeetingAI
	ChatAI
}

type Service struct {
	*UserService
	*MeetingService
	*ChatService
}

func New(r Repository, t Transcriber, ai AI) *Service {
	return &Service{
		UserService:    NewUserService(r),
		MeetingService: NewMeetingService(ai, t, r),
		ChatService:    NewChatService(ai),
	}
}
