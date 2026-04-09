package httpx

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/dsnikitin/sowhat/internal/pkg/errx"
	"github.com/dsnikitin/sowhat/internal/pkg/logger"
	"github.com/pkg/errors"
)

// TODO добавить конфиги и возможность настраивать клиент

type Client struct {
	client *http.Client
}

func NewClient() *Client {
	return &Client{
		client: http.DefaultClient,
	}
}

func (c *Client) DoRequestWithContext(
	ctx context.Context, method string, url string, headers map[string]string, body io.Reader, result any,
) error {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return errors.Wrap(err, "new http request")
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "do http request")
	}
	defer func(body io.ReadCloser) {
		if err := body.Close(); err != nil {
			logger.Log.Errorw("Failed to close response body", "error", err)
		}
	}(resp.Body)

	if resp.StatusCode != 200 {
		switch resp.StatusCode {
		case 400:
			err = errx.ErrBadRequest
		case 401:
			err = errx.ErrUnauthorized
		case 402:
			err = errx.ErrPaymentRequired
		case 403:
			err = errx.ErrPermissionDenied
		case 404:
			err = errx.ErrNotFound
		case 413:
			err = errx.ErrTooLarge
		case 422:
			err = errx.ErrUnprocessable
		case 429:
			err = errx.ErrTooManyRequests
		case 500:
			err = errx.ErrInternalServer
		}

		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return errors.Wrap(readErr, "read response body")
		}

		return errors.Wrapf(err, "error response code received: code %d, error %s", resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return errors.Wrap(err, "decode response body")
	}

	return nil
}
