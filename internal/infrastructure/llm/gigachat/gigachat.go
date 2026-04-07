package gigachat

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/dsnikitin/sowhat/internal/pkg/httpx"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type Authorizer interface {
	GetAccessToken(authToken string) (string, error)
}

type GigaChat struct {
	appCtx     context.Context
	cfg        *Config
	client     *httpx.Client
	authorizer Authorizer
}

func New(appCtx context.Context, cfg *Config, client *httpx.Client, a Authorizer) *GigaChat {
	return &GigaChat{
		appCtx:     appCtx,
		cfg:        cfg,
		client:     client,
		authorizer: a,
	}
}

func (g *GigaChat) Summarize(transcript string) (string, error) {
	msgs := []Message{
		{Role: "system", Content: summarizeSystemPrompt},
		{Role: "user", Content: transcript},
	}

	return "ЗАГЛУШКА", nil

	return g.complete(msgs)
}

func (g *GigaChat) complete(msgs []Message) (string, error) {
	accessToken, err := g.authorizer.GetAccessToken(g.cfg.OAuth.AuthToken)
	if err != nil {
		return "", errors.Wrap(err, "get access token")
	}

	pld, err := json.Marshal(NewRequest(msgs))
	if err != nil {
		return "", errors.Wrap(err, "marshal request body")
	}

	headers := map[string]string{
		"Content-Type":  "application/json",
		"Accept":        "application/json",
		"Authorization": "Bearer " + accessToken,
	}

	var res CompletionsResponse
	err = g.client.DoRequestWithContext(
		g.appCtx, http.MethodPost, g.cfg.RestAPI.Completions, headers, bytes.NewReader(pld), &res)
	if err != nil {
		return "", errors.Wrap(err, "do http request with context")
	}

	return res.Choices[0].Message.Content, nil
}

func (g *GigaChat) UploadFile(data io.Reader) (uuid.UUID, error) {
	// возвращает fileID
	return uuid.Nil, nil
}
