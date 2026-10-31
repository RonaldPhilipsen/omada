package cloud

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTokenFetchAndCache(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token":"tok-1","expires_in":3600}`))
	}))
	defer srv.Close()

	tp := NewTokenProvider(srv.URL, "user", "pass", nil)

	tok1, err := tp.Token(context.Background())
	if err != nil {
		t.Fatalf("first Token() error: %v", err)
	}
	if tok1 != "tok-1" {
		t.Fatalf("got token %q, want %q", tok1, "tok-1")
	}
	tok2, err := tp.Token(context.Background())
	if err != nil {
		t.Fatalf("second Token() error: %v", err)
	}
	if tok2 != tok1 {
		t.Fatalf("second Token() = %q, want cached %q", tok2, tok1)
	}
	if hits != 1 {
		t.Fatalf("token endpoint hits = %d, want 1", hits)
	}
}

func TestTokenRefreshOnExpiry(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"access_token":"tok-%d","expires_in":1}`, hits)))
	}))
	defer srv.Close()

	tp := NewTokenProvider(srv.URL, "user", "pass", nil)

	tok1, err := tp.Token(context.Background())
	if err != nil {
		t.Fatalf("first Token() error: %v", err)
	}
	tok2, err := tp.Token(context.Background())
	if err != nil {
		t.Fatalf("second Token() error: %v", err)
	}
	if tok1 == tok2 {
		t.Fatalf("expected refresh after expiry, got same token %q", tok1)
	}
	if hits != 2 {
		t.Fatalf("token endpoint hits = %d, want 2", hits)
	}
}

func TestTokenEndpointFailure(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer srv.Close()

	tp := NewTokenProvider(srv.URL, "user", "pass", nil)

	if _, err := tp.Token(context.Background()); err == nil {
		t.Fatal("expected error from 500 response, got nil")
	}
	if _, err := tp.Token(context.Background()); err == nil {
		t.Fatal("expected error on retry, got nil")
	}
	if hits != 2 {
		t.Fatalf("token endpoint hits = %d, want 2 (no poisoned cache)", hits)
	}
}

func TestTokenEndpointBadJSON(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	tp := NewTokenProvider(srv.URL, "user", "pass", nil)

	if _, err := tp.Token(context.Background()); err == nil {
		t.Fatal("expected error from non-JSON body, got nil")
	}
	if hits != 1 {
		t.Fatalf("token endpoint hits = %d, want 1", hits)
	}
}
