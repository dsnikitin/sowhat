package httpx

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

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
	}(req.Body)

	if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "read response body")
		}

		return errors.Errorf("error response code received: code %d, error %s", resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return errors.Wrap(err, "decode response body")
	}

	return nil
}
