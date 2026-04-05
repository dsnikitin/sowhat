package gigachat

import (
	"context"
	"io"
	"net/http"

	"github.com/google/uuid"
)

type Authorizer interface {
	GetAccessToken(authToken string) (string, error)
}

type GigaChat struct {
	globalCtx  context.Context
	cfg        *Config
	client     *http.Client
	authorizer Authorizer
}

func New(globalCtx context.Context, cfg *Config, a Authorizer) *GigaChat {
	return &GigaChat{
		globalCtx:  globalCtx,
		cfg:        cfg,
		client:     http.DefaultClient,
		authorizer: a,
	}
}

func (c *GigaChat) UploadFile(data io.Reader) (uuid.UUID, error) {
	// возвращает fileID
	return uuid.Nil, nil
}
