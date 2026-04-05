package repository

import (
	"context"
	"strings"

	"github.com/dsnikitin/sowhat/internal/infrastructure/db/postgres"
	"github.com/dsnikitin/sowhat/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
)

type MeetingRepository struct {
	db *postgres.DB
}

func NewMeetingRepository(db *postgres.DB) *MeetingRepository {
	return &MeetingRepository{db: db}
}

const createMeetingSQL = `
	INSERT INTO sowhat.meetings(id, user_id)
	VALUES(@id, userID)
`

func (r *MeetingRepository) CreateMeeting(ctx context.Context, userID int64) error {
	_, err := r.db.Exec(ctx, createMeetingSQL, pgx.NamedArgs{"userID": userID})
	return errors.Wrap(err, "exec")
}

const setTranscriptionSQL = `
	UPDATE sowhat.meetings
	SET transcript = @transcript
	WHERE id = @id
`

func (r *MeetingRepository) SetTranscription(ctx context.Context, id int64, transcript string) error {
	_, err := r.db.Exec(ctx, setTranscriptionSQL, pgx.NamedArgs{"id": id, "transcript": transcript})
	return errors.Wrap(err, "exec")
}

const setSummarySQL = `
	UPDATE sowhat.meetings
	SET summary = @summary
	WHERE id = @id
`

func (r *MeetingRepository) SetSummary(ctx context.Context, id int64, summary string) error {
	_, err := r.db.Exec(ctx, setSummarySQL, pgx.NamedArgs{"id": id, "summary": summary})
	return errors.Wrap(err, "exec")
}

const getMeetingSQL = `
	SELECT id, transcript
	FROM sowhat.meetings
	WHERE id = @id AND user_id = @userID
`

func (r *MeetingRepository) GetMeeting(ctx context.Context, id, userID int64) (models.MeetingWithTranscript, error) {
	args := pgx.NamedArgs{"id": id, "userID": userID}
	fieldsPointer := func(m *models.MeetingWithTranscript) []any { return m.ScanFields() }

	resp, err := postgres.QueryRow(ctx, r.db, getMeetingSQL, args, fieldsPointer)
	return resp, errors.Wrap(err, "query one")
}

const listMeetingsSQL = `
	SELECT id, summary 
	FROM sowhat.meetings
	WHERE user_id = @userID
	ORDER BY created_at DESC
`

func (r *MeetingRepository) ListMeetings(ctx context.Context, userID int64) ([]models.MeetingWithSummary, error) {
	args := pgx.NamedArgs{"userID": userID}
	fieldsPointer := func(m *models.MeetingWithSummary) []any { return m.ScanFields() }

	items, err := postgres.Query(ctx, r.db, listMeetingsSQL, args, fieldsPointer)
	return items, errors.Wrap(err, "query")
}

func (r *MeetingRepository) FindMeetings(ctx context.Context, userID int64, query string) ([]models.MeetingWithSummary, error) {
	if strings.TrimSpace(query) == "" {
		return []models.MeetingWithSummary{}, nil
	}

	return nil, errors.New("not implemented")
}
