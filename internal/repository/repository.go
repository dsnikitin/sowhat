package repository

import (
	"github.com/dsnikitin/sowhat/internal/infrastructure/db/postgres"
)

type Repository struct {
	*UserRepository
	*MeetingRepository
	*TranscriptionRepository
}

func New(db *postgres.DB) *Repository {
	return &Repository{
		UserRepository:          NewUserRepository(db),
		MeetingRepository:       NewMeetingRepository(db),
		TranscriptionRepository: NewTranscriptionRepository(db),
	}
}
