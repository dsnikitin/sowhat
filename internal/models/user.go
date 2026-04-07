package models

import (
	"time"

	"github.com/dsnikitin/sowhat/internal/consts/platform"
)

type User struct {
	ID           int64
	ExternalID   string
	Name         string
	Platform     platform.Type
	RegisteredAt time.Time
}

func (u *User) ScanFields() []any {
	return []any{&u.ID, &u.ExternalID, &u.Name, &u.Platform, &u.RegisteredAt}
}
