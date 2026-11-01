// Package testutil provides shared mock-controller test helpers for the
// controller package tests.
package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RonaldPhilipsen/omada/controller"
)

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

// NewTestClient starts a mock Omada controller serving /api/info and login,
// then creates a logged-in Controller. All other requests are delegated to
// handler.
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
