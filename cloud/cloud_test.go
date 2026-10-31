package cloud

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockNorthbound is a combined token+API mock: /v1/token serves token JSON,
// every other path is delegated to api.
type mockNorthbound struct {
	srv       *httptest.Server
	apiHits   int
	tokenHits int
}

func newMockNorthbound(t *testing.T, api func(w http.ResponseWriter, r *http.Request)) *mockNorthbound {
	t.Helper()
	m := &mockNorthbound{}
	m.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/token" {
			m.tokenHits++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(fmt.Sprintf(`{"access_token":"tok-%d","expires_in":3600}`, m.tokenHits)))
			return
		}
		m.apiHits++
		api(w, r)
	}))
	t.Cleanup(m.srv.Close)
	return m
}

func TestNewRequiresCredentials(t *testing.T) {
	cases := map[string]Options{
		"empty username": {BaseURL: "https://example.com", Password: "p"},
		"empty password": {BaseURL: "https://example.com", Username: "u"},
		"both empty":     {},
	}
	for name, opts := range cases {
		if _, err := New(opts); err == nil {
			t.Fatalf("%s: New succeeded, want error", name)
		}
	}
}

func TestRequestCarriesBearerToken(t *testing.T) {
	m := newMockNorthbound(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tok-1" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer tok-1")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errorCode":0}`))
	})

	c, err := New(Options{BaseURL: m.srv.URL, Username: "u", Password: "p"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	resp, err := c.Site().GetMspClusterNodesInfo(context.Background(), "msp-1")
	if err != nil {
		t.Fatalf("GetMspClusterNodesInfo: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if m.apiHits != 1 {
		t.Fatalf("api hits = %d, want 1", m.apiHits)
	}
	if m.tokenHits != 1 {
		t.Fatalf("token hits = %d, want 1", m.tokenHits)
	}
}

func Test401TriggersRefreshAndRetry(t *testing.T) {
	var first bool
	m := newMockNorthbound(t, func(w http.ResponseWriter, r *http.Request) {
		if !first {
			first = true
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"errorCode":-44116,"message":"token expired"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errorCode":0}`))
	})

	c, err := New(Options{BaseURL: m.srv.URL, Username: "u", Password: "p"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	resp, err := c.Site().GetMspClusterNodesInfo(context.Background(), "msp-1")
	if err != nil {
		t.Fatalf("GetMspClusterNodesInfo: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 after refresh and retry", resp.StatusCode)
	}
	if m.apiHits != 2 {
		t.Fatalf("api hits = %d, want 2", m.apiHits)
	}
	if m.tokenHits != 2 {
		t.Fatalf("token hits = %d, want 2", m.tokenHits)
	}
}

func Test401TwiceFails(t *testing.T) {
	m := newMockNorthbound(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"errorCode":-44116,"message":"token expired"}`))
	})

	c, err := New(Options{BaseURL: m.srv.URL, Username: "u", Password: "p"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	resp, err := c.Site().GetMspClusterNodesInfo(context.Background(), "msp-1")
	if err != nil {
		t.Fatalf("GetMspClusterNodesInfo: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	if m.apiHits != 2 {
		t.Fatalf("api hits = %d, want 2 (no retry loop)", m.apiHits)
	}
	if m.tokenHits != 3 {
		t.Fatalf("token hits = %d, want 3", m.tokenHits)
	}
}
