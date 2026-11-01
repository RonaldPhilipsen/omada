package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const refreshSkew = 60 * time.Second

// TokenProvider fetches and caches a northbound client-credentials token.
type TokenProvider struct {
	baseURL  string
	username string
	password string
	hc       *http.Client

	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

func NewTokenProvider(baseURL, username, password string, hc *http.Client) *TokenProvider {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &TokenProvider{
		baseURL:  baseURL,
		username: username,
		password: password,
		hc:       hc,
	}
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

// Token returns the cached token if it is still valid, otherwise fetches a
// new one from the northbound token endpoint.
func (tp *TokenProvider) Token(ctx context.Context) (string, error) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	if tp.token != "" && time.Now().Before(tp.expiresAt) {
		return tp.token, nil
	}

	body, err := json.Marshal(map[string]string{
		"username": tp.username,
		"password": tp.password,
	})
	if err != nil {
		return "", fmt.Errorf("encode token request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tp.baseURL+"/v1/token", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("new token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := tp.hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint: %s", resp.Status)
	}
	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("token response: missing access_token")
	}

	tp.token = tr.AccessToken
	tp.expiresAt = time.Now().Add(time.Duration(tr.ExpiresIn)*time.Second - refreshSkew)
	return tp.token, nil
}

// Invalidate drops the cached token so the next Token call refetches.
func (tp *TokenProvider) Invalidate() {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	tp.token = ""
	tp.expiresAt = time.Time{}
}

// HeaderValue returns the full Authorization header value for a request:
// "Bearer <token>".
func (tp *TokenProvider) HeaderValue(ctx context.Context) (string, error) {
	tok, err := tp.Token(ctx)
	if err != nil {
		return "", err
	}
	return "Bearer " + tok, nil
}
