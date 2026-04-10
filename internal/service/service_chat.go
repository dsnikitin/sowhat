package service

import (
	"context"
	"iter"

	"github.com/dsnikitin/sowhat/internal/models"
	"github.com/pkg/errors"
)

type Chatter interface {
	Chat(
		ctx context.Context, query string, fileIDs iter.Seq2[string, error], history iter.Seq2[models.ChatMessage, error],
	) (models.ChatMessage, error)
}

type ChatRepository interface {
	GetFileIDs(ctx context.Context, userID int64) iter.Seq2[string, error]
	SaveMessage(ctx context.Context, userID int64, msg models.ChatMessage) error
	GetMessages(ctx context.Context, userID int64) iter.Seq2[models.ChatMessage, error]
	DeleteMessages(ctx context.Context, userID int64) error
}

type ChatService struct {
	ch Chatter
	r  ChatRepository
}

func NewChatService(ch Chatter, r ChatRepository) *ChatService {
	return &ChatService{ch: ch, r: r}
}

func (s *ChatService) NewChat(ctx context.Context, userID int64, query string) (string, error) {
	fileIDs := s.r.GetFileIDs(ctx, userID)

	msg, err := s.ch.Chat(ctx, query, fileIDs, func(func(models.ChatMessage, error) bool) {})
	if err != nil {
		return "", errors.Wrap(err, "new chat with chatter")
	}

	if err := s.r.DeleteMessages(ctx, userID); err != nil {
		return "", errors.Wrap(err, "delete messages")
	}

	if err := s.r.SaveMessage(ctx, userID, msg); err != nil {
		return "", errors.Wrap(err, "save message")
	}

	return msg.Answer, nil
}

func (s *ChatService) ContinueChat(ctx context.Context, userID int64, query string) (string, error) {
	fileIDs := s.r.GetFileIDs(ctx, userID)
	history := s.r.GetMessages(ctx, userID)

	msg, err := s.ch.Chat(ctx, query, fileIDs, history)
	if err != nil {
		return "", errors.Wrap(err, "continue chat with chatter")
	}

	if err := s.r.SaveMessage(ctx, userID, msg); err != nil {
		return "", errors.Wrap(err, "save message")
	}

	return msg.Answer, nil
}
