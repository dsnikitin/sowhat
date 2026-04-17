package gigachat

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"iter"
	"mime/multipart"
	"net/http"
	"net/textproto"

	"github.com/dsnikitin/sowhat/internal/models"
	"github.com/dsnikitin/sowhat/internal/pkg/errx"
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
	ctx context.Context, query string, fileIDs iter.Seq2[string, error], history iter.Seq2[models.ChatMessage, error],
) (models.ChatMessage, error) {
	fIDs := []string{}
	for id, err := range fileIDs {
		if err != nil {
			return models.ChatMessage{}, errors.Wrap(err, "iterate fileIDs")
		}
		fIDs = append(fIDs, id)
	}

	if len(fIDs) == 0 && !g.cfg.CanBeMyself {
		return models.ChatMessage{}, errx.ErrNoFilesForQuestion
	}

	msgs := make([]Message, 0, 2)
	sessionId := uuid.New().String()

	msgs = append(msgs, Message{Role: "system", Content: chatAboutMeetingsSystemPrompt})
	for m, err := range history {
		if err != nil {
			return models.ChatMessage{}, errors.Wrap(err, "iterate history")
		}

		sessionId = m.ChatID

		msgs = append(msgs,
			Message{Role: "user", Content: m.Query},
			Message{Role: "assistant", Content: m.Answer},
		)
	}
	msgs = append(msgs, Message{Role: "user", Content: query, Attachments: fIDs})

	headers := map[string]string{"X-Session-ID": sessionId}
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

func (g *GigaChat) Upload(content io.Reader, contentType string) (string, error) {
	accessToken, err := g.authorizer.GetAccessToken(g.cfg.OAuth.AuthToken)
	if err != nil {
		return "", errors.Wrap(err, "get access token")
	}

	reqBody := &bytes.Buffer{}
	writer := multipart.NewWriter(reqBody)

	if err := writeMultipartBody(content, writer, contentType); err != nil {
		return "", errors.Wrap(err, "build multipart body")
	}

	headers := map[string]string{
		"Content-Type":  writer.FormDataContentType(),
		"Accept":        "application/json",
		"Authorization": "Bearer " + accessToken,
	}

	if err := writer.Close(); err != nil {
		return "", errors.Wrap(err, "close multipart writer")
	}

	var res UploadResponse
	err = g.client.DoRequestWithContext(
		g.appCtx, http.MethodPost, g.cfg.RestAPI.UploadFile, headers, reqBody, &res)
	if err != nil {
		return "", errors.Wrap(err, "do http request with context")
	}

	return res.FileId, nil
}

func writeMultipartBody(fileContent io.Reader, writer *multipart.Writer, contentType string) error {
	if err := writer.WriteField("purpose", "general"); err != nil {
		return errors.Wrap(err, "write purpose filed")
	}

	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", `form-data; name="file"; filename="file.txt"`)
	partHeader.Set("Content-Type", contentType)
	part, err := writer.CreatePart(partHeader)
	if err != nil {
		return errors.Wrap(err, "create form-data")
	}

	_, err = io.Copy(part, fileContent)
	return errors.Wrap(err, "copy file content")
}
