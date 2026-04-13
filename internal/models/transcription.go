package models

import (
	"io"
	"time"

	"github.com/dsnikitin/sowhat/internal/consts/stage"
)

type Transcription struct {
	Meeting             Meeting
	TranscriberRqFileID *string
	TranscriberTaskID   *string
	TranscriberRsFileID *string
	IsCompleted         bool
	StageAttemptsCount  int
	LastAttemptAt       time.Time
	FileContent         io.Reader
	FileMIME            string
}

func (t *Transcription) FieldPointers() []any {
	return append(t.Meeting.FieldPointers(),
		&t.TranscriberRqFileID, &t.TranscriberTaskID, &t.TranscriberRsFileID, &t.IsCompleted)
}

type TranscriptionStage struct {
	Name      stage.Name
	StageFn   func(tr *Transcription) error
	PersistFn func(tr Transcription) error
}

func NewTranscriptionStage(
	name stage.Name, processFn func(tr *Transcription) error, persistFn func(tr Transcription) error,
) TranscriptionStage {
	return TranscriptionStage{Name: name, StageFn: processFn, PersistFn: persistFn}
}

type TranscriptionCompleteEvent struct {
	MeetingID int64
	UserID    int64
	IsFailed  bool
}
