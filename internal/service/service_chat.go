package service

import (
	"context"

	"github.com/dsnikitin/sowhat/internal/models"
	"github.com/dsnikitin/sowhat/internal/pkg/errx"
	"github.com/pkg/errors"
)

type Chatter interface {
	Chat(ctx context.Context, query string, fileIDs []string, history []models.ChatMessage) (models.ChatMessage, error)
}

type ChatRepository interface {
	GetFileIDs(ctx context.Context, userID int64) ([]string, error)
	SaveMessage(ctx context.Context, userID int64, msg models.ChatMessage) error
	GetMessages(ctx context.Context, userID int64) ([]models.ChatMessage, error)
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
	fileIDs, err := s.r.GetFileIDs(ctx, userID)
	if err != nil {
		return "", errors.Wrap(err, "get file ids")
	}

	// if len(fileIDs) == 0 {
	// 	return "", errx.ErrNoFilesForQuestion
	// }

	msg, err := s.ch.Chat(ctx, query, fileIDs, nil)
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
	fileIDs, err := s.r.GetFileIDs(ctx, userID)
	if err != nil {
		return "", errors.Wrap(err, "get file ids")
	}

	if len(fileIDs) == 0 {
		return "", errx.ErrNoFilesForQuestion
	}

	history, err := s.r.GetMessages(ctx, userID)
	if err != nil {
		return "", errors.Wrap(err, "get history")
	}

	msg, err := s.ch.Chat(ctx, query, fileIDs, history)
	if err != nil {
		return "", errors.Wrap(err, "continue chat with chatter")
	}

	if err := s.r.SaveMessage(ctx, userID, msg); err != nil {
		return "", errors.Wrap(err, "save message")
	}

	return msg.Answer, nil
}
