package repository

import (
	"context"
	"iter"

	"github.com/dsnikitin/sowhat/internal/infrastructure/db/postgres"
	"github.com/dsnikitin/sowhat/internal/models"
	"github.com/dsnikitin/sowhat/internal/pkg/errx"
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
)

type TranscriptionRepository struct {
	db *postgres.DB
}

func NewTranscriptionRepository(db *postgres.DB) *TranscriptionRepository {
	return &TranscriptionRepository{db: db}
}

const createTranscriptionSQL = `
	INSERT INTO sowhat.transcriptions(meeting_id)
	VALUES(@meetingID)
`

func (r *TranscriptionRepository) CreateTranscription(ctx context.Context, meetingID int64) error {
	args := pgx.NamedArgs{"meetingID": meetingID}
	_, err := r.db.Exec(ctx, createTranscriptionSQL, args)
	return errors.Wrap(err, "exec")
}

const updateTranscriptionSQL = `
	UPDATE sowhat.transcriptions
	SET transcriber_rq_file_id = @TranscriberRqFileID,
		transcriber_task_id = @transcriberTaskID,
		transcriber_rs_file_id = @TranscriberRsFileID,
		is_completed = @isCompleted
	WHERE meeting_id = @meetingID
`

func (r *TranscriptionRepository) UpdateTranscription(ctx context.Context, tr models.Transcription) error {
	args := pgx.NamedArgs{
		"meetingID":           tr.Meeting.ID,
		"TranscriberRqFileID": tr.TranscriberRqFileID,
		"transcriberTaskID":   tr.TranscriberTaskID,
		"TranscriberRsFileID": tr.TranscriberRsFileID,
		"isCompleted":         tr.IsCompleted,
	}

	res, err := r.db.Exec(ctx, updateTranscriptionSQL, args)
	if err != nil {
		return errors.Wrap(err, "exec")
	}

	if res.RowsAffected() == 0 {
		return errx.ErrNotFound
	}

	return nil
}

const getNotCompletedTranscriptionsSQL = `
	SELECT m.id, m.user_id, m.transcript, m.summary, m.chatter_file_id, m.is_transcription_failed, m.created_at, m.raw_transcript,
		t.transcriber_rq_file_id, t.transcriber_task_id, t.transcriber_rs_file_id, t.is_completed
	FROM sowhat.transcriptions t
	INNER JOIN sowhat.meetings m ON t.meeting_id = m.id
	WHERE NOT t.is_completed
`

func (r *TranscriptionRepository) GetNotCompletedTranscriptions(ctx context.Context) iter.Seq2[models.Transcription, error] {
	fieldsPointer := func(m *models.Transcription) []any { return m.ScanFields() }
	return postgres.Query(ctx, r.db, getNotCompletedTranscriptionsSQL, pgx.NamedArgs{}, fieldsPointer)
}
