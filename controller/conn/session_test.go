package conn

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSessionCookieSentAfterLogin(t *testing.T) {
	var gotCookie string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		result := map[string]any{}
		switch r.URL.Path {
		case "/api/info":
			result = map[string]any{"controllerVer": "6.3.0.111", "omadacId": "id"}
		case "/id/api/v2/login":
			http.SetCookie(w, &http.Cookie{Name: "TPOMADA_SESSIONID", Value: "session", Path: "/"})
			result = map[string]any{"token": "csrf"}
		case "/id/api/v2/users/current":
			if c, err := r.Cookie("TPOMADA_SESSIONID"); err == nil {
				gotCookie = c.Value
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"errorCode": 0, "result": result})
	}))
	defer srv.Close()

	c, err := New(srv.URL, "user", "pass", srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Login(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := c.RequestEndpoint(context.Background(), "GET", "v2", "users/current", "", nil, nil); err != nil {
		t.Fatal(err)
	}
	if gotCookie != "session" {
		t.Errorf("session cookie = %q, want %q", gotCookie, "session")
	}
}
