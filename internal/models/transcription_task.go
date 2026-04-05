package models

type TranscriptionTaskStatus string

const (
	New      TranscriptionTaskStatus = "NEW"
	Running  TranscriptionTaskStatus = "RUNNING"
	Canceled TranscriptionTaskStatus = "CANCELED"
	Done     TranscriptionTaskStatus = "DONE"
	Error    TranscriptionTaskStatus = "ERROR"
)

type TranscriptionTask struct {
	ID     string                  `json:"id"`
	Status TranscriptionTaskStatus `json:"status"`
}
