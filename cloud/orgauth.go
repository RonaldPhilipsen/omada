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

// OrgTokenProvider fetches and caches an on-prem org-level Open API token
// via the client-credentials grant against the controller's own
// /openapi/authorize/token endpoint. The response is enveloped
// (errorCode/msg/result); a non-zero errorCode is an error. The returned
// refreshToken is not used: re-authentication always re-runs the
// client-credentials grant.
type OrgTokenProvider struct {
	baseURL      string
	omadacID     string
	clientID     string
	clientSecret string
	hc           *http.Client

	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

func NewOrgTokenProvider(baseURL, omadacID, clientID, clientSecret string, hc *http.Client) *OrgTokenProvider {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &OrgTokenProvider{
		baseURL:      baseURL,
		omadacID:     omadacID,
		clientID:     clientID,
		clientSecret: clientSecret,
		hc:           hc,
	}
}

type orgTokenResponse struct {
	ErrorCode int32  `json:"errorCode"`
	Msg       string `json:"msg"`
	Result    struct {
		AccessToken  string `json:"accessToken"`
		TokenType    string `json:"tokenType"`
		ExpiresIn    int64  `json:"expiresIn"`
		RefreshToken string `json:"refreshToken"`
	} `json:"result"`
}

// Token returns the cached token if it is still valid, otherwise fetches a
// new one from the on-prem org-level token endpoint.
func (p *OrgTokenProvider) Token(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.token != "" && time.Now().Before(p.expiresAt) {
		return p.token, nil
	}

	body, err := json.Marshal(map[string]string{
		"omadacId":      p.omadacID,
		"client_id":     p.clientID,
		"client_secret": p.clientSecret,
	})
	if err != nil {
		return "", fmt.Errorf("encode token request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/openapi/authorize/token?grant_type=client_credentials", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("new token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint: %s", resp.Status)
	}
	var tr orgTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if tr.ErrorCode != 0 {
		return "", fmt.Errorf("token endpoint: errorCode %d: %s", tr.ErrorCode, tr.Msg)
	}
	if tr.Result.AccessToken == "" {
		return "", fmt.Errorf("token response: missing accessToken")
	}

	p.token = tr.Result.AccessToken
	p.expiresAt = time.Now().Add(time.Duration(tr.Result.ExpiresIn)*time.Second - refreshSkew)
	return p.token, nil
}

// Invalidate drops the cached token so the next Token call refetches.
func (p *OrgTokenProvider) Invalidate() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.token = ""
	p.expiresAt = time.Time{}
}

// HeaderValue returns the full Authorization header value for a request:
// "AccessToken=<token>".
func (p *OrgTokenProvider) HeaderValue(ctx context.Context) (string, error) {
	tok, err := p.Token(ctx)
	if err != nil {
		return "", err
	}
	return "AccessToken=" + tok, nil
}
