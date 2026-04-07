package models

type ChatMessage struct {
	ChatID string
	Query  string
	Answer string
}

func (m ChatMessage) ScanFields() []any {
	return []any{&m.ChatID, &m.Query, &m.Answer}
}
