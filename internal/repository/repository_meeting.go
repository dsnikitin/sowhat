package repository

import (
	"context"
	"iter"

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
	INSERT INTO sowhat.meetings(user_id)
	VALUES(@userID)
	RETURNING id
`

func (r *MeetingRepository) CreateMeeting(ctx context.Context, userID int64) (int64, error) {
	args := pgx.NamedArgs{"userID": userID}
	fieldPointer := func(id *int64) []any { return []any{id} }

	id, err := postgres.QueryRow(ctx, r.db, createMeetingSQL, args, fieldPointer)
	return id, errors.Wrap(err, "query row")
}

const getMeetingSQL = `
	SELECT id, user_id, transcript, summary, chatter_file_id, is_transcription_failed, created_at, raw_transcript
	FROM sowhat.meetings
	WHERE id = @id AND user_id = @userID
`

func (r *MeetingRepository) GetMeeting(ctx context.Context, id, userID int64) (models.Meeting, error) {
	args := pgx.NamedArgs{"id": id, "userID": userID}
	fieldsPointer := func(m *models.Meeting) []any { return m.FieldPointers() }

	resp, err := postgres.QueryRow(ctx, r.db, getMeetingSQL, args, fieldsPointer)
	return resp, errors.Wrap(err, "query one")
}

const listMeetingsSQL = `
	SELECT id, user_id, transcript, summary, chatter_file_id, is_transcription_failed,
		created_at, raw_transcript, COUNT(*) OVER() AS total
	FROM sowhat.meetings
	WHERE user_id = @userID
	ORDER BY created_at DESC
	LIMIT @limit OFFSET @offset
`

func (r *MeetingRepository) ListMeetings(ctx context.Context, userID int64, limit, offset int) iter.Seq2[models.MeetingWithTotal, error] {
	args := pgx.NamedArgs{"userID": userID, "limit": limit, "offset": offset}
	fieldsPointer := func(m *models.MeetingWithTotal) []any { return m.FieldPointers() }
	return postgres.Query(ctx, r.db, listMeetingsSQL, args, fieldsPointer)
}

const findMeetingsSQL = `
	SELECT id, user_id, transcript, summary, chatter_file_id, is_transcription_failed,
		created_at, raw_transcript, COUNT(*) OVER() AS total
	FROM sowhat.meetings
	WHERE user_id = @userID
		AND NOT is_transcription_failed
		AND transcript_tsv @@ websearch_to_tsquery('russian', @query)
	ORDER BY ts_rank(transcript_tsv, websearch_to_tsquery('russian', @query)) DESC, created_at DESC
	LIMIT @limit OFFSET @offset
`

func (r *MeetingRepository) FindMeetings(
	ctx context.Context, userID int64, query string, limit, offset int,
) iter.Seq2[models.MeetingWithTotal, error] {
	args := pgx.NamedArgs{"userID": userID, "query": query, "limit": limit, "offset": offset}
	fieldsPointer := func(m *models.MeetingWithTotal) []any { return m.FieldPointers() }
	return postgres.Query(ctx, r.db, findMeetingsSQL, args, fieldsPointer)
}
