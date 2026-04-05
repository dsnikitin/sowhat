package models

import (
	"encoding/json"
	"time"
)

type Meeting struct {
	ID        int64 `json:"id"`
	CreatedAt time.Time
}

func (m *Meeting) ScanFields() []any {
	return []any{&m.ID, &m.CreatedAt}
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

type MeetingWithTranscript struct {
	Meeting
	Transcript string `json:"transcript"`
}

func (m *MeetingWithTranscript) ScanFields() []any {
	return append(m.Meeting.ScanFields(), &m.Transcript)
}

type MeetingWithSummary struct {
	Meeting
	Summary string `json:"summary"`
}

func (m *MeetingWithSummary) ScanFields() []any {
	return append(m.Meeting.ScanFields(), &m.Summary)
}
