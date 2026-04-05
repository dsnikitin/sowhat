package service

type ChatAI interface {
}

type ChatService struct {
	ai ChatAI
}

func NewChatService(ai ChatAI) *ChatService {
	return &ChatService{ai: ai}
}

func (s *ChatService) Chat(query string) (string, error) {
	return "Не реализоввано", nil
}
