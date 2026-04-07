package errx

import (
	"github.com/dsnikitin/sowhat/internal/consts/format"
	"github.com/pkg/errors"
)

var (
	ErrAlreadyExists               = errors.New("already exists")
	ErrNotFound                    = errors.New("not found")
	ErrAllWorkersBusy              = errors.New("all workers are busy")
	ErrAccessTokenExpired          = errors.New("access token expired")
	ErrRecognitionTaskNotCompleted = errors.New("recognition task is not completed")
	ErrRecognitionTaskFailed       = errors.New("recognition is failed")
	ErrTooEarly                    = errors.New("too early")
	ErrNoFilesForQuestion          = errors.New("no files for question")
)

type ErrUnsupportedAudioFormat struct {
	SupportedFormats []format.Type
	Err              error
}

func NewUnsupportedAudioFormatError(supportedFormats []format.Type, err error) error {
	return &ErrUnsupportedAudioFormat{
		SupportedFormats: supportedFormats,
		Err:              err,
	}
}

func (e *ErrUnsupportedAudioFormat) Error() string {
	return e.Err.Error()
}

func (e *ErrUnsupportedAudioFormat) Unwrap() error {
	return e.Err
}

type ErrUnsupportedFileSize struct {
	MinSize int64
	MaxSize int64
	Err     error
}

func NewUnsupportedFileSizeError(minSize, maxSize int64, err error) error {
	return &ErrUnsupportedFileSize{
		MinSize: minSize,
		MaxSize: maxSize,
		Err:     err,
	}
}

func (e *ErrUnsupportedFileSize) Error() string {
	return e.Err.Error()
}

func (e *ErrUnsupportedFileSize) Unwrap() error {
	return e.Err
}
