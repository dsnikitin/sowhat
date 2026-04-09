package models

import (
	"encoding/json"
	"time"
)

type Meeting struct {
	ID                    int64
	UserID                int64
	Transcript            *string
	Summary               *string
	ChatterFileId         *string
	IsTranscriptionFailed bool
	CreatedAt             time.Time
	RawTranscript         *string
}

func (m *Meeting) ScanFields() []any {
	return []any{
		&m.ID, &m.UserID, &m.Transcript, &m.Summary, &m.ChatterFileId,
		&m.IsTranscriptionFailed, &m.CreatedAt, &m.RawTranscript,
	}
}

func (m Meeting) MarshalJSON() ([]byte, error) {
	type Alias Meeting
	resp := &struct {
		Alias
		CreatedAt string `json:"created_at"`
	}{
		Alias:     Alias(m),
		CreatedAt: m.CreatedAt.Truncate(time.Second).Format(time.RFC3339),
	}

	return json.Marshal(resp)
}

type MeetingWithTotal struct {
	Meeting
	Total int
}

func (m *MeetingWithTotal) ScanFields() []any {
	return append(m.Meeting.ScanFields(), &m.Total)
}
