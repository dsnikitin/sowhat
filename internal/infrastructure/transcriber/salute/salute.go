package salute

import (
	"context"
	"io"
	"net/http"

	"github.com/google/uuid"
)

type Authorizer interface {
	GetAccessToken(authToken string) (string, error)
}

type SaluteSpeech struct {
	appCtx     context.Context
	cfg        *Config
	client     *http.Client
	authorizer Authorizer
}

func New(appCtx context.Context, cfg *Config, a Authorizer) *SaluteSpeech {
	return &SaluteSpeech{
		appCtx:     appCtx,
		cfg:        cfg,
		client:     http.DefaultClient,
		authorizer: a,
	}
}

func (c *SaluteSpeech) UploadFile(data io.Reader) (uuid.UUID, error) {
	// возвращает fileID
	return uuid.Nil, nil
}

func (c *SaluteSpeech) RecognizeAsync(fileID uuid.UUID) (string, error) {
	// возвращает task_id
	return "", nil
}

func (c *SaluteSpeech) GetTaskStatus(taskID string) (string, error) {
	// возвращает task_status - завести enum
	return "", nil
}

func (c *SaluteSpeech) DownloadResult(fileID uuid.UUID) error {
	return nil
}
