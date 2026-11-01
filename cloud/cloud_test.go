package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RonaldPhilipsen/omada/cloud/site"
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

// bodySpy records the body each non-token request is sent with, then
// forwards the request carrying a fresh copy of the captured bytes. It
// makes what the auth transport transmits on each attempt observable
// without relying on server-side body-read behavior.
type bodySpy struct {
	base   http.RoundTripper
	bodies *[]string
}

func (s *bodySpy) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Path != "/v1/token" {
		b, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		*s.bodies = append(*s.bodies, string(b))
		req = req.Clone(req.Context())
		req.Body = io.NopCloser(bytes.NewReader(b))
		req.ContentLength = int64(len(b))
	}
	return s.base.RoundTrip(req)
}

func Test401RetryResendsRequestBody(t *testing.T) {
	var sentBodies []string
	spy := &bodySpy{base: http.DefaultTransport, bodies: &sentBodies}
	var first bool
	m := newMockNorthbound(t, func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading request body: %v", err)
			return
		}
		var vo site.CustomerOpenApiVO
		if err := json.Unmarshal(b, &vo); err != nil {
			t.Errorf("decoding request body: %v", err)
			return
		}
		if vo.CustomerName == nil || *vo.CustomerName != "acme" {
			t.Errorf("customerName = %v, want %q", vo.CustomerName, "acme")
		}
		if !first {
			first = true
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"errorCode":-44116,"message":"token expired"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errorCode":0,"data":{"customerId":"cust-1"}}`))
	})

	c, err := New(Options{BaseURL: m.srv.URL, Username: "u", Password: "p", HTTPClient: &http.Client{Transport: spy}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	name := "acme"
	resp, err := c.Site().CreateCustomer(context.Background(), "msp-1", site.CustomerOpenApiVO{CustomerName: &name})
	if err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 after refresh and retry", resp.StatusCode)
	}
	var decoded struct {
		ErrorCode int32 `json:"errorCode"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if decoded.ErrorCode != 0 {
		t.Errorf("errorCode = %d, want 0", decoded.ErrorCode)
	}
	if len(sentBodies) != 2 {
		t.Fatalf("request bodies sent = %d, want 2", len(sentBodies))
	}
	if want := `{"customerName":"acme"}`; sentBodies[1] != want {
		t.Errorf("retried request body = %s, want %s", sentBodies[1], want)
	}
	if m.apiHits != 2 {
		t.Fatalf("api hits = %d, want 2", m.apiHits)
	}
	if m.tokenHits != 2 {
		t.Fatalf("token hits = %d, want 2", m.tokenHits)
	}
}
