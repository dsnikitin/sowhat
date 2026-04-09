package oauth

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dsnikitin/sowhat/internal/pkg/errx"
	"github.com/dsnikitin/sowhat/internal/pkg/logger"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

type HTTPCleint interface {
	DoRequestWithContext(ctx context.Context, method, url string, headers map[string]string, body io.Reader, result any) error
}

type Authorizer struct {
	appCtx context.Context
	client HTTPCleint
	cfgs   []*Config
	tokens map[string]*accessToken
}

type accessToken struct {
	mu        sync.RWMutex
	Token     string    `json:"access_token"`
	ExpiresAt time.Time `json:"expires_at"`
}

func New(appCtx context.Context, cfgs []*Config, client HTTPCleint) (*Authorizer, error) {
	a := &Authorizer{
		appCtx: appCtx,
		client: client,
		cfgs:   cfgs,
		tokens: make(map[string]*accessToken, len(cfgs)),
	}

	eg := errgroup.Group{}
	for _, cfg := range cfgs {
		eg.Go(func() error {
			_, err := a.getAccessToken(cfg)
			return errors.Wrapf(err, "get access token for %s", cfg.Consumer)
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, errors.Wrap(err, "errgroup wait")
	}

	for _, cfg := range cfgs {
		go a.scheduleRefresh(cfg, a.tokens[cfg.AuthToken].ExpiresAt)
	}

	return a, nil
}

func (a *Authorizer) GetAccessToken(authToken string) (string, error) {
	token, ok := a.tokens[authToken]
	if !ok {
		return "", errx.ErrNotFound
	}

	token.mu.RLock()
	defer token.mu.RUnlock()

	if time.Now().After(token.ExpiresAt) {
		return "", errx.ErrAccessTokenExpired
	}

	return token.Token, nil
}

func (a *Authorizer) getAccessToken(cfg *Config) (time.Time, error) {
	logger.Log.Infow("Getting access token...", "consumer", cfg.Consumer)

	headers := map[string]string{
		"Content-Type":  "application/x-www-form-urlencoded",
		"Accept":        "application/json",
		"RqUID":         uuid.New().String(),
		"Authorization": "Basic " + cfg.AuthToken,
	}

	newToken := struct {
		Token     string `json:"access_token"`
		ExpiresAt int64  `json:"expires_at"`
	}{}

	err := a.client.DoRequestWithContext(
		a.appCtx, http.MethodPost, cfg.Endpoint, headers, strings.NewReader("scope="+cfg.Scope), &newToken)
	if err != nil {
		return time.Time{}, errors.Wrap(err, "do http request with context")
	}

	expiresAt := unixToTime(newToken.ExpiresAt)
	if existingToken, ok := a.tokens[cfg.AuthToken]; !ok {
		a.tokens[cfg.AuthToken] = &accessToken{
			Token:     newToken.Token,
			ExpiresAt: expiresAt,
		}
	} else {
		existingToken.mu.Lock()
		existingToken.Token = newToken.Token
		existingToken.ExpiresAt = expiresAt
		existingToken.mu.Unlock()
	}

	logger.Log.Infow("Successfully got access token", "consuner", cfg.Consumer)
	return expiresAt, nil
}

func (a *Authorizer) scheduleRefresh(cfg *Config, expiresAt time.Time) {
	if cfg.RefreshThreshold <= 0 {
		return
	}

	refreshTime := expiresAt.Add(-cfg.RefreshThreshold)
	waitTime := max(time.Until(refreshTime), 0)

	select {
	case <-a.appCtx.Done():
		return
	case <-time.After(waitTime):
		if newExpiresAt, err := a.getAccessToken(cfg); err != nil {
			logger.Log.Errorw("Failed to get access token", "consumer", cfg.Consumer, "error", err.Error())
			time.Sleep(time.Second)
			go a.scheduleRefresh(cfg, expiresAt)
		} else {
			go a.scheduleRefresh(cfg, newExpiresAt)
		}
	}
}

func unixToTime(unixtime int64) time.Time {
	switch {
	case unixtime >= 1e15:
		return time.UnixMicro(unixtime)
	case unixtime >= 1e12:
		return time.UnixMilli(unixtime)
	default:
		return time.Unix(unixtime, 0)
	}
}
