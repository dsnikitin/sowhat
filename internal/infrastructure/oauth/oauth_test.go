package oauth

import (
	"context"
	"fmt"
	"io"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/dsnikitin/sowhat/internal/pkg/errx"
)

type mockHTTPClient struct {
	mu         sync.Mutex
	callCount  int
	failAlways bool
}

func (m *mockHTTPClient) DoRequestWithContext(
	ctx context.Context, method, url string, headers map[string]string, body io.Reader, result any,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++

	if m.failAlways {
		return fmt.Errorf("mock error")
	}

	if tokenResult, ok := result.(*struct {
		Token     string `json:"access_token"`
		ExpiresAt int64  `json:"expires_at"`
	}); ok {
		tokenResult.Token = fmt.Sprintf("token_%d", m.callCount)
		tokenResult.ExpiresAt = time.Now().Add(1 * time.Hour).Unix()
	}

	return nil
}

func testConfig() *Config {
	return &Config{
		Consumer:         "test",
		AuthToken:        "auth_token",
		Endpoint:         "https://example.com/token",
		Scope:            "read",
		RefreshThreshold: 10 * time.Minute,
	}
}

func TestNew_WithCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel() // отмена контекста, чтобы остановить горутины scheduleRefresh

		cfg := testConfig()
		mockClient := &mockHTTPClient{}

		authorizer, err := New(ctx, []*Config{cfg}, mockClient)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		token, err := authorizer.GetAccessToken(cfg.AuthToken)
		if err != nil {
			t.Fatalf("GetAccessToken() error = %v", err)
		}
		if token == "" {
			t.Error("Token is empty")
		}
	})
}

func TestGetAccessToken_Success(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cfg := testConfig()
		mockClient := &mockHTTPClient{}
		authorizer := &Authorizer{
			appCtx: context.Background(),
			client: mockClient,
			tokens: make(map[string]*accessToken),
		}

		if _, err := authorizer.getAccessToken(cfg); err != nil {
			t.Fatalf("getAccessToken() error = %v", err)
		}

		token, err := authorizer.GetAccessToken(cfg.AuthToken)
		if err != nil {
			t.Fatalf("GetAccessToken() error = %v", err)
		}
		if token == "" {
			t.Error("Token is empty")
		}
	})
}

func TestGetAccessToken_Error(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cfg := testConfig()
		mockClient := &mockHTTPClient{failAlways: true}
		authorizer := &Authorizer{
			appCtx: context.Background(),
			client: mockClient,
			tokens: make(map[string]*accessToken),
		}

		if _, err := authorizer.getAccessToken(cfg); err == nil {
			t.Error("getAccessToken() expected error, got nil")
		}
	})
}

func TestGetAccessToken_UpdateExisting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cfg := testConfig()
		mockClient := &mockHTTPClient{}
		authorizer := &Authorizer{
			appCtx: context.Background(),
			client: mockClient,
			tokens: make(map[string]*accessToken),
		}

		expiresAt1, err := authorizer.getAccessToken(cfg)
		if err != nil {
			t.Fatalf("First getAccessToken() error = %v", err)
		}
		token1, _ := authorizer.GetAccessToken(cfg.AuthToken)

		// Прокручиваем время вперед, чтобы токен устарел
		time.Sleep(2 * time.Hour)
		synctest.Wait()

		expiresAt2, err := authorizer.getAccessToken(cfg)
		if err != nil {
			t.Fatalf("Second getAccessToken() error = %v", err)
		}
		token2, _ := authorizer.GetAccessToken(cfg.AuthToken)

		if token1 == token2 {
			t.Error("Token should be updated")
		}
		if !expiresAt2.After(expiresAt1) {
			t.Error("ExpiresAt should be updated to a later time")
		}
	})
}

func TestGetAccessToken_NotFound(t *testing.T) {
	authorizer := &Authorizer{
		tokens: make(map[string]*accessToken),
	}

	if _, err := authorizer.GetAccessToken("unknown_auth_token"); err != errx.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestGetAccessToken_Expired(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		authorizer := &Authorizer{
			tokens: map[string]*accessToken{
				"token": {
					Token:     "old",
					ExpiresAt: time.Now().Add(-1 * time.Hour),
				},
			},
		}

		if _, err := authorizer.GetAccessToken("token"); err != errx.ErrAccessTokenExpired {
			t.Errorf("Expected ErrAccessTokenExpired, got %v", err)
		}
	})
}

func TestScheduleRefresh_Success(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cfg := testConfig()
		mockClient := &mockHTTPClient{}
		authorizer := &Authorizer{
			appCtx: ctx,
			client: mockClient,
			tokens: make(map[string]*accessToken),
		}

		expiresAt, err := authorizer.getAccessToken(cfg)
		if err != nil {
			t.Fatalf("getAccessToken() error = %v", err)
		}

		firstToken, _ := authorizer.GetAccessToken(cfg.AuthToken)
		initialCallCount := mockClient.callCount

		// Запускаем планировщик
		go authorizer.scheduleRefresh(cfg, expiresAt)
		synctest.Wait()

		// Прокручиваем время до обновления
		refreshTime := expiresAt.Add(-cfg.RefreshThreshold)
		time.Sleep(time.Until(refreshTime))
		synctest.Wait()

		newToken, _ := authorizer.GetAccessToken(cfg.AuthToken)
		if newToken == firstToken {
			t.Error("Token was not refreshed")
		}

		if mockClient.callCount != initialCallCount+1 {
			t.Errorf("Expected %d calls, got %d", initialCallCount+1, mockClient.callCount)
		}
	})
}

func TestScheduleRefresh_ContextCancelled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cfg := testConfig()
		mockClient := &mockHTTPClient{}
		authorizer := &Authorizer{
			appCtx: ctx,
			client: mockClient,
			tokens: make(map[string]*accessToken),
		}

		expiresAt, err := authorizer.getAccessToken(cfg)
		if err != nil {
			t.Fatalf("getAccessToken() error = %v", err)
		}

		go authorizer.scheduleRefresh(cfg, expiresAt)
		synctest.Wait()

		cancel()
		synctest.Wait()
	})
}

func TestScheduleRefreshThreshold_IsZero(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cfg := testConfig()
		cfg.RefreshThreshold = 0
		mockClient := &mockHTTPClient{}
		authorizer := &Authorizer{
			appCtx: context.Background(),
			client: mockClient,
			tokens: make(map[string]*accessToken),
		}

		expiresAt, err := authorizer.getAccessToken(cfg)
		if err != nil {
			t.Fatalf("getAccessToken() error = %v", err)
		}

		firstToken, _ := authorizer.GetAccessToken(cfg.AuthToken)
		initialCallCount := mockClient.callCount

		go authorizer.scheduleRefresh(cfg, expiresAt)
		synctest.Wait()

		// добавляем токену время жизни, чтобы не вернулась ошибка ErrAccessTokenExpired
		// когда будем ниже вызывать GetAccessToken
		authorizer.tokens[cfg.AuthToken].ExpiresAt = authorizer.tokens[cfg.AuthToken].ExpiresAt.Add(10 * time.Hour)

		// Прокручиваем время
		time.Sleep(2 * time.Hour)
		synctest.Wait()

		currentToken, _ := authorizer.GetAccessToken(cfg.AuthToken)
		if currentToken != firstToken {
			t.Errorf("Token should not be refreshed when threshold is 0, first token %s, current token %s", firstToken, currentToken)
		}

		if mockClient.callCount != initialCallCount {
			t.Error("No additional calls should be made")
		}
	})
}
