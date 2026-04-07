package repository

import (
	"context"

	"github.com/dsnikitin/sowhat/internal/infrastructure/db/postgres"
	"github.com/dsnikitin/sowhat/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
)

type ChatRepository struct {
	db *postgres.DB
}

func NewChatRepository(db *postgres.DB) *ChatRepository {
	return &ChatRepository{db: db}
}

const saveMessageSQL = `
	INSERT INTO sowhat.chat(user_id, id, query, answer)
	VALUES (@userID, @id, @query, @answer)
`

func (r *ChatRepository) SaveMessage(ctx context.Context, userID int64, msg models.ChatMessage) error {
	args := pgx.NamedArgs{
		"userID": userID,
		"id":     msg.ChatID,
		"query":  msg.Query,
		"answer": msg.Answer,
	}
	_, err := r.db.Exec(ctx, saveMessageSQL, args)
	return errors.Wrap(err, "exec")
}

const getMessagesSQL = `
	SELECT id, query, answer
	FROM sowhat.chat
	WHERE user_id = @userID
	ORDER BY created_at
`

func (r *ChatRepository) GetMessages(ctx context.Context, userID int64) ([]models.ChatMessage, error) {
	args := pgx.NamedArgs{"userID": userID}
	fieldsPointer := func(m *models.ChatMessage) []any { return m.ScanFields() }

	meetings, err := postgres.Query(ctx, r.db, getMessagesSQL, args, fieldsPointer)
	return meetings, errors.Wrap(err, "query")
}

const deleteMessagesSQL = `
	DELETE FROM sowhat.chat
	WHERE user_id = @userID
`

func (r *ChatRepository) DeleteMessages(ctx context.Context, userID int64) error {
	_, err := r.db.Exec(ctx, deleteMessagesSQL, pgx.NamedArgs{"userID": userID})
	return errors.Wrap(err, "exec")
}
