package repository

import (
	"context"
	"strings"

	"github.com/dsnikitin/sowhat/internal/infrastructure/db/postgres"
	"github.com/dsnikitin/sowhat/internal/models"
	"github.com/dsnikitin/sowhat/internal/pkg/errx"
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
	fieldsPointer := func(m *models.Meeting) []any { return m.ScanFields() }

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

func (r *MeetingRepository) ListMeetings(ctx context.Context, userID int64, limit, offset int) ([]models.Meeting, int, error) {
	var total int
	args := pgx.NamedArgs{"userID": userID, "limit": limit, "offset": offset}
	fieldsPointer := func(m *models.Meeting) []any { return append(m.ScanFields(), &total) }

	meetings, err := postgres.Query(ctx, r.db, listMeetingsSQL, args, fieldsPointer)
	return meetings, total, errors.Wrap(err, "query")
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
) ([]models.Meeting, int, error) {
	query = strings.TrimSpace(query)

	if query == "" {
		return []models.Meeting{}, 0, nil
	}

	var total int
	args := pgx.NamedArgs{"userID": userID, "query": query, "limit": limit, "offset": offset}
	fieldsPointer := func(m *models.Meeting) []any { return append(m.ScanFields(), &total) }

	meetings, err := postgres.Query(ctx, r.db, findMeetingsSQL, args, fieldsPointer)
	return meetings, total, errors.Wrap(err, "query")
}

const updateMeetingSQL = `
	UPDATE sowhat.meetings
	SET transcript = @transcript,
		summary = @summary,
		chatter_file_id = @chatterFileID,
		is_transcription_failed = @isTranscriptionFailed,
		raw_transcript = @rawTranscript
	WHERE id = @id
`

func (r *MeetingRepository) UpdateMeeting(ctx context.Context, meeting models.Meeting) error {
	args := pgx.NamedArgs{
		"id":                    meeting.ID,
		"transcript":            meeting.Transcript,
		"summary":               meeting.Summary,
		"chatterFileID":         meeting.ChatterFileId,
		"isTranscriptionFailed": meeting.IsTranscriptionFailed,
		"rawTranscript":         meeting.RawTranscript,
	}

	res, err := r.db.Exec(ctx, updateMeetingSQL, args)
	if err != nil {
		return errors.Wrap(err, "exec")
	}

	if res.RowsAffected() == 0 {
		return errx.ErrNotFound
	}

	return nil
}

const getFileIDsSQL = `
	SELECT chatter_file_id
	FROM sowhat.meetings
	WHERE user_id = @userID AND chatter_file_id IS NOT NULL
`

func (r *MeetingRepository) GetFileIDs(ctx context.Context, userID int64) ([]string, error) {
	args := pgx.NamedArgs{"userID": userID}
	fieldsPointer := func(id *string) []any { return []any{id} }

	ids, err := postgres.Query(ctx, r.db, getFileIDsSQL, args, fieldsPointer)
	return ids, errors.Wrap(err, "query")
}
