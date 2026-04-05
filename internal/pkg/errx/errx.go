package errx

import "github.com/pkg/errors"

var (
	ErrInternalServer     = errors.New("internal server error")
	ErrAlreadyExists      = errors.New("already exists")
	ErrAlreadyAccepted    = errors.New("already accepted")
	ErrNotFound           = errors.New("not found")
	ErrAllWorkersBusy     = errors.New("all workers are busy")
	ErrToManyRequests     = errors.New("too many requests")
	ErrAccessTokenExpired = errors.New("access token expired")
	ErrIncorrectMeetingID = errors.New("incorrect meeting id")
)
