package oauth

import (
	"context"
	"encoding/json"
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

type Authorizer struct {
	appCtx context.Context
	client *http.Client
	cfgs   []*Config
	tokens map[string]*accessToken
}

type accessToken struct {
	mu        sync.RWMutex
	Token     string    `json:"access_token"`
	ExpiresAt time.Time `json:"expires_at"`
}

func New(globalCtx context.Context, cfgs []*Config) (*Authorizer, error) {
	a := &Authorizer{
		appCtx: globalCtx,
		client: http.DefaultClient,
		cfgs:   cfgs,
		tokens: make(map[string]*accessToken, len(cfgs)),
	}

	eg := errgroup.Group{}
	for _, cfg := range cfgs {
		eg.Go(func() error {
			return a.getAccessToken(cfg)
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, errors.Wrap(err, "errgroup wait")
	}

	return a, nil
}

func (a *Authorizer) GetAccessToken(authToken string) (string, error) {
	token, ok := a.tokens[authToken]
	if !ok {
		return "", errx.ErrNotFound
	}

	token.mu.RLock()
	if time.Now().After(token.ExpiresAt) {
		return "", errx.ErrAccessTokenExpired
	}
	defer token.mu.RUnlock()

	return token.Token, nil
}

func (a *Authorizer) getAccessToken(cfg *Config) error {
	logger.Log.Infof("Getting access token to %s...", cfg.Consumer)

	req, err := http.NewRequestWithContext(a.appCtx, http.MethodPost, cfg.Endpoint, strings.NewReader("scope="+cfg.Scope))
	if err != nil {
		return errors.Wrap(err, "new http request")
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("RqUID", uuid.New().String())
	req.Header.Add("Authorization", "Basic "+cfg.AuthToken)

	resp, err := a.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "do http request")
	}
	defer resp.Body.Close()

	newToken := struct {
		Token     string `json:"access_token"`
		ExpiresAt int64  `json:"expires_at"`
	}{}

	if err := json.NewDecoder(resp.Body).Decode(&newToken); err != nil {
		return errors.Wrap(err, "decode access token")
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

	logger.Log.Infof("Successfuly got access token to %s", cfg.Consumer)

	go a.scheduleRefresh(cfg, expiresAt)

	return nil
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
		go a.getAccessToken(cfg)
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
