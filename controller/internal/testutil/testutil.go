// Package testutil provides shared mock-controller test helpers for the
// controller and site package tests.
package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RonaldPhilipsen/omada/controller"
)

// statusCapture records the status code, headers, and body a handler writes.
type statusCapture struct {
	http.ResponseWriter
	header http.Header
	body   []byte
	status int
}

func (s *statusCapture) Header() http.Header {
	if s.header == nil {
		s.header = http.Header{}
	}
	return s.header
}

func (s *statusCapture) WriteHeader(code int) {
	if s.status == 0 {
		s.status = code
	}
}

func (s *statusCapture) Write(p []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	s.body = append(s.body, p...)
	return len(p), nil
}

const (
	// TestControllerID is the omadacId served by the mock controller.
	TestControllerID = "abc"
	// TestControllerVer is the controller version served by the mock controller.
	TestControllerVer = "6.2.10.18"
)

var loginCount int

// LoginCount returns the total number of logins performed by mock controllers
// created via NewTestClient.
func LoginCount() int { return loginCount }

// WriteJSON writes v as a JSON response body.
func WriteJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Errorf("encode response: %v", err)
	}
}

// NewTestClient starts a mock Omada controller serving /api/info, login, and
// the site-1 lookup, then creates a logged-in Controller. All other requests
// are delegated to handler. The site-1 lookup is first offered to handler; if
// the handler answers 404, the standard site-1 info is served so site-scoped
// handles can be created without every test mocking the lookup.
func NewTestClient(t *testing.T, handler http.HandlerFunc) *controller.Controller {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/info":
			WriteJSON(t, w, map[string]any{"errorCode": 0, "controllerVer": TestControllerVer, "omadacId": TestControllerID})
		case r.URL.Path == "/"+TestControllerID+"/api/v2/login":
			if r.Method == http.MethodPost {
				loginCount++
			}
			WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"token": "csrf-token"}})
		case r.URL.Path == "/"+TestControllerID+"/api/v2/sites/site-1":
			captured := &statusCapture{ResponseWriter: w}
			handler(captured, r)
			if captured.status == http.StatusNotFound {
				WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"id": "site-1", "name": "Office", "controllerVer": TestControllerVer}})
				return
			}
			for k, vv := range captured.header {
				for _, v := range vv {
					w.Header().Add(k, v)
				}
			}
			w.WriteHeader(captured.status)
			w.Write(captured.body)
		default:
			handler(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	cl, err := controller.New(srv.URL, "user", "pass")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return cl
}
