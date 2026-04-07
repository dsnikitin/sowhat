package service

type Chatter interface {
}

type ChatService struct {
	ch Chatter
}

func NewChatService(ch Chatter) *ChatService {
	return &ChatService{ch: ch}
}

func (s *ChatService) Chat(query string) (string, error) {
	return "Не реализоввано", nil
}
