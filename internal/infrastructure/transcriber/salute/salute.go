package salute

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/dsnikitin/sowhat/internal/consts/format"
	"github.com/dsnikitin/sowhat/internal/pkg/errx"
	"github.com/dsnikitin/sowhat/internal/pkg/httpx"
	"github.com/pkg/errors"
)

type taskStatus string

const (
	new      taskStatus = "NEW"
	running  taskStatus = "RUNNING"
	canceled taskStatus = "CANCELED"
	done     taskStatus = "DONE"
	failed   taskStatus = "ERROR"
)

type Authorizer interface {
	GetAccessToken(authToken string) (string, error)
}

type SaluteSpeech struct {
	appCtx     context.Context
	cfg        *Config
	client     *httpx.Client
	authorizer Authorizer
}

func New(appCtx context.Context, cfg *Config, cleint *httpx.Client, a Authorizer) *SaluteSpeech {
	return &SaluteSpeech{
		appCtx:     appCtx,
		cfg:        cfg,
		client:     cleint,
		authorizer: a,
	}
}

func (s *SaluteSpeech) SupportedFormats() []format.Type {
	return format.SaluteSpeechSupported()
}

func (s *SaluteSpeech) MinAndMaxFileSize() (int64, int64) {
	return s.cfg.MinFileSize, s.cfg.MaxFileSize
}

func (s *SaluteSpeech) UploadFile(fileContent io.Reader, mime string) (string, error) {
	accessToken, err := s.authorizer.GetAccessToken(s.cfg.OAuth.AuthToken)
	if err != nil {
		return "", errors.Wrap(err, "get access token")
	}

	headers := map[string]string{
		"Content-Type":  mime,
		"Accept":        "application/json",
		"Authorization": "Bearer " + accessToken,
	}

	var res UploadResponse
	err = s.client.DoRequestWithContext(
		s.appCtx, http.MethodPost, s.cfg.RestAPI.UploadData, headers, fileContent, &res)
	if err != nil {
		return "", errors.Wrap(err, "do http request with context")
	}

	return res.Result.FileId.String(), nil
}

func (s *SaluteSpeech) AsyncRecognize(fileID, mime string) (string, error) {
	accessToken, err := s.authorizer.GetAccessToken(s.cfg.OAuth.AuthToken)
	if err != nil {
		return "", errors.Wrap(err, "get access token")
	}

	pld, err := json.Marshal(NewRequstByAudioFormat(fileID, format.FromMIME(mime)))
	if err != nil {
		return "", errors.Wrap(err, "marshal request body")
	}

	headers := map[string]string{
		"Content-Type":  "application/json",
		"Accept":        "application/json",
		"Authorization": "Bearer " + accessToken,
	}

	var res RecognizeResponse
	err = s.client.DoRequestWithContext(
		s.appCtx, http.MethodPost, s.cfg.RestAPI.AsyncRecognize, headers, bytes.NewReader(pld), &res)
	if err != nil {
		return "", errors.Wrap(err, "do http request with context")
	}

	return res.Result.TaksId, nil
}

func (s *SaluteSpeech) CheckTaskCompleted(taskID string) (string, error) {
	accessToken, err := s.authorizer.GetAccessToken(s.cfg.OAuth.AuthToken)
	if err != nil {
		return "", errors.Wrap(err, "get access token")
	}

	endpoint := s.cfg.RestAPI.GetTaskStatus + "?id=" + taskID
	headers := map[string]string{
		"Accept":        "application/json",
		"Authorization": "Bearer " + accessToken,
	}

	var res CheckTaskResponse
	err = s.client.DoRequestWithContext(
		s.appCtx, http.MethodGet, endpoint, headers, http.NoBody, &res)
	if err != nil {
		return "", errors.Wrap(err, "do http request with context")
	}

	switch taskStatus(res.Result.Status) {
	case new, running:
		return "", errx.ErrRecognitionTaskNotCompleted
	case canceled, failed:
		return "", errx.ErrRecognitionTaskFailed
	case done:
		fallthrough
	default:
		return res.Result.ResponseFileID, nil
	}
}

func (s *SaluteSpeech) DownloadTranscript(fileID string) (string, []string, error) {
	accessToken, err := s.authorizer.GetAccessToken(s.cfg.OAuth.AuthToken)
	if err != nil {
		return "", nil, errors.Wrap(err, "get access token")
	}

	endpoint := s.cfg.RestAPI.DownloadData + "?response_file_id=" + fileID
	headers := map[string]string{
		"Accept":        "application/json",
		"Authorization": "Bearer " + accessToken,
	}

	var res DownloadResponse
	err = s.client.DoRequestWithContext(
		s.appCtx, http.MethodGet, endpoint, headers, http.NoBody, &res)
	if err != nil {
		return "", nil, errors.Wrap(err, "do http request with context")
	}

	phrases := make([]string, 0, len(res))
	for i := range res {
		for j := range res[i].Results {
			phrases = append(phrases, res[i].Results[j].Text)
		}
	}

	return strings.Join(phrases, ". "), phrases, nil
}
