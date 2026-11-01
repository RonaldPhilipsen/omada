package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// orgTokenMock serves an on-prem org-level token endpoint at
// /openapi/authorize/token and records each request's path, query and body.
type orgTokenMock struct {
	srv   *httptest.Server
	hits  int
	path  string
	query string
	body  map[string]string
}

func newOrgTokenMock(t *testing.T, respond func(hits int, w http.ResponseWriter, r *http.Request)) *orgTokenMock {
	t.Helper()
	m := &orgTokenMock{}
	m.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.hits++
		if m.path == "" {
			m.path = r.URL.Path
			m.query = r.URL.RawQuery
			b, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("reading request body: %v", err)
				return
			}
			if err := json.Unmarshal(b, &m.body); err != nil {
				t.Errorf("decoding request body: %v", err)
			}
		}
		respond(m.hits, w, r)
	}))
	t.Cleanup(m.srv.Close)
	return m
}

func TestOrgTokenFetchAndCache(t *testing.T) {
	m := newOrgTokenMock(t, func(hits int, w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errorCode":0,"msg":"ok","result":{"accessToken":"org-tok-1","tokenType":"bearer","expiresIn":7200,"refreshToken":"rt-1"}}`))
	})

	p := NewOrgTokenProvider(m.srv.URL, "org-1", "cid", "csecret", nil)

	tok1, err := p.Token(context.Background())
	if err != nil {
		t.Fatalf("first Token() error: %v", err)
	}
	if tok1 != "org-tok-1" {
		t.Fatalf("got token %q, want %q", tok1, "org-tok-1")
	}
	tok2, err := p.Token(context.Background())
	if err != nil {
		t.Fatalf("second Token() error: %v", err)
	}
	if tok2 != tok1 {
		t.Fatalf("second Token() = %q, want cached %q", tok2, tok1)
	}
	if m.hits != 1 {
		t.Fatalf("token endpoint hits = %d, want 1", m.hits)
	}
	if m.path != "/openapi/authorize/token" {
		t.Errorf("path = %q, want /openapi/authorize/token", m.path)
	}
	if m.query != "grant_type=client_credentials" {
		t.Errorf("query = %q, want grant_type=client_credentials", m.query)
	}
	wantBody := map[string]string{
		"omadacId":      "org-1",
		"client_id":     "cid",
		"client_secret": "csecret",
	}
	for k, want := range wantBody {
		if got := m.body[k]; got != want {
			t.Errorf("body[%q] = %q, want %q", k, got, want)
		}
	}
}

func TestOrgTokenRefreshOnExpiry(t *testing.T) {
	m := newOrgTokenMock(t, func(hits int, w http.ResponseWriter, r *http.Request) {
		// expiresIn=1 is past the 60s refresh skew, so the token is
		// already expired on return.
		_, _ = w.Write([]byte(fmt.Sprintf(
			`{"errorCode":0,"msg":"ok","result":{"accessToken":"org-tok-%d","tokenType":"bearer","expiresIn":1,"refreshToken":"rt-%d"}}`, hits, hits)))
	})

	p := NewOrgTokenProvider(m.srv.URL, "org-1", "cid", "csecret", nil)

	tok1, err := p.Token(context.Background())
	if err != nil {
		t.Fatalf("first Token() error: %v", err)
	}
	tok2, err := p.Token(context.Background())
	if err != nil {
		t.Fatalf("second Token() error: %v", err)
	}
	if tok1 == tok2 {
		t.Fatalf("expected refresh after expiry, got same token %q", tok1)
	}
	if m.hits != 2 {
		t.Fatalf("token endpoint hits = %d, want 2", m.hits)
	}
}

func TestOrgTokenEndpointFailure(t *testing.T) {
	m := newOrgTokenMock(t, func(hits int, w http.ResponseWriter, r *http.Request) {
		if hits == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errorCode":0,"msg":"ok","result":{"accessToken":"org-tok-2","tokenType":"bearer","expiresIn":7200,"refreshToken":"rt-2"}}`))
	})

	p := NewOrgTokenProvider(m.srv.URL, "org-1", "cid", "csecret", nil)

	if _, err := p.Token(context.Background()); err == nil {
		t.Fatal("expected error from 500 response, got nil")
	}
	tok, err := p.Token(context.Background())
	if err != nil {
		t.Fatalf("expected retry to succeed, got error: %v", err)
	}
	if tok != "org-tok-2" {
		t.Fatalf("got token %q, want %q", tok, "org-tok-2")
	}
	if m.hits != 2 {
		t.Fatalf("token endpoint hits = %d, want 2 (no poisoned cache)", m.hits)
	}
}

func TestOrgTokenEnvelopeError(t *testing.T) {
	m := newOrgTokenMock(t, func(hits int, w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errorCode":-44106,"msg":"application not found"}`))
	})

	p := NewOrgTokenProvider(m.srv.URL, "org-1", "cid", "csecret", nil)

	_, err := p.Token(context.Background())
	if err == nil {
		t.Fatal("expected error from non-zero errorCode, got nil")
	}
	if !strings.Contains(err.Error(), "-44106") {
		t.Errorf("error = %v, want it to mention the envelope errorCode -44106", err)
	}
	if m.hits != 1 {
		t.Fatalf("token endpoint hits = %d, want 1", m.hits)
	}
}

func TestOrgTokenBadJSON(t *testing.T) {
	m := newOrgTokenMock(t, func(hits int, w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json"))
	})

	p := NewOrgTokenProvider(m.srv.URL, "org-1", "cid", "csecret", nil)

	if _, err := p.Token(context.Background()); err == nil {
		t.Fatal("expected error from non-JSON body, got nil")
	}
	if m.hits != 1 {
		t.Fatalf("token endpoint hits = %d, want 1", m.hits)
	}
}
