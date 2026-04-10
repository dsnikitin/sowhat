package models

import (
	"io"
	"time"
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

type File struct {
	Reader    io.Reader
	MIME      string
	Size      int64
	MeetingID int64
}

type TranscriptionCompletedMsg struct {
	MeetingID             int64
	UserID                int64
	IsTranscriptionFailed bool
}
