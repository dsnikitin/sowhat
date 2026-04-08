package gigachat

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/dsnikitin/sowhat/internal/models"
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

	headers := make(map[string]string)
	return g.complete(msgs, headers)
}

func (g *GigaChat) Chat(
	ctx context.Context, query string, fileIDs []string, history []models.ChatMessage,
) (models.ChatMessage, error) {
	msgs := make([]Message, 0, len(history)+2)

	msgs = append(msgs, Message{Role: "system", Content: chatAboutMeetingsSystemPrompt})
	for _, m := range history {
		msgs = append(msgs,
			Message{Role: "user", Content: m.Query},
			Message{Role: "assistant", Content: m.Answer},
		)
	}
	msgs = append(msgs, Message{Role: "user", Content: query, Attachments: fileIDs})

	sessionId := uuid.New().String()
	headers := make(map[string]string)
	if len(history) > 0 {
		sessionId = history[0].ChatID
		headers["X-Session-ID"] = sessionId
	}

	answer, err := g.complete(msgs, headers)
	if err != nil {
		return models.ChatMessage{}, errors.Wrap(err, "complete")
	}

	return models.ChatMessage{ChatID: sessionId, Query: query, Answer: answer}, nil
}

func (g *GigaChat) complete(msgs []Message, headers map[string]string) (string, error) {
	accessToken, err := g.authorizer.GetAccessToken(g.cfg.OAuth.AuthToken)
	if err != nil {
		return "", errors.Wrap(err, "get access token")
	}

	pld, err := json.Marshal(NewRequest(msgs))
	if err != nil {
		return "", errors.Wrap(err, "marshal request body")
	}

	headers["Authorization"] = "Bearer " + accessToken
	headers["Content-Type"] = "application/json"
	headers["Accept"] = "application/json"

	var res CompletionsResponse
	err = g.client.DoRequestWithContext(
		g.appCtx, http.MethodPost, g.cfg.RestAPI.Completions, headers, bytes.NewReader(pld), &res)
	if err != nil {
		return "", errors.Wrap(err, "do http request with context")
	}

	return res.Choices[0].Message.Content, nil
}

func (g *GigaChat) UploadFile(fileContent io.Reader, contentType string) (string, error) {
	accessToken, err := g.authorizer.GetAccessToken(g.cfg.OAuth.AuthToken)
	if err != nil {
		return "", errors.Wrap(err, "get access token")
	}

	reqBody, err := buildMultipartBody(fileContent)
	if err != nil {
		return "", errors.Wrap(err, "build multipart body")
	}

	headers := map[string]string{
		"Content-Type":  "multipart/form-data",
		"Accept":        "application/json",
		"Authorization": "Bearer " + accessToken,
	}

	var res UploadResponse
	err = g.client.DoRequestWithContext(
		g.appCtx, http.MethodPost, g.cfg.RestAPI.UploadFile, headers, reqBody, &res)
	if err != nil {
		return "error_id", nil // TODO
		// return "", errors.Wrap(err, "do http request with context")
	}

	return "some_id", nil // TODO
}

func buildMultipartBody(fileContent io.Reader) (*bytes.Buffer, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	defer writer.Close()

	if err := writer.WriteField("purpose", "general"); err != nil {
		return nil, errors.Wrap(err, "write purpose filed")
	}

	part, err := writer.CreateFormFile("file", "file.txt")
	if err != nil {
		return nil, errors.Wrap(err, "create form-data")
	}

	if _, err := io.Copy(part, fileContent); err != nil {
		return nil, errors.Wrap(err, "copy file content")
	}

	return body, nil
}
