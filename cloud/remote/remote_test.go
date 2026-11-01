package remote

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// decodeBody decodes a JSON response body into dst.
func decodeBody(t *testing.T, resp *http.Response, dst any) {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		t.Fatalf("decoding response body: %v", err)
	}
}

// checkEnvelope asserts the standard {errorCode, msg} success envelope.
func checkEnvelope(t *testing.T, code *int32, msg *string) {
	t.Helper()
	if code == nil {
		t.Fatal("errorCode = nil, want 0")
	}
	if *code != 0 {
		t.Errorf("errorCode = %d, want 0", *code)
	}
	if msg == nil {
		t.Fatal("msg = nil, want OK")
	}
	if *msg != "OK" {
		t.Errorf("msg = %q, want %q", *msg, "OK")
	}
}

// checkTotalRows asserts the totalRows field of a grid result.
func checkTotalRows(t *testing.T, got *int64, want int64) {
	t.Helper()
	if got == nil {
		t.Fatal("totalRows = nil, want non-nil")
	}
	if *got != want {
		t.Errorf("totalRows = %d, want %d", *got, want)
	}
}

// checkString asserts a non-nil string field.
func checkString(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Errorf("%s = %q, want %q", name, *got, want)
	}
}

// checkInt32 asserts a non-nil int32 field.
func checkInt32(t *testing.T, name string, got *int32, want int32) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %d", name, want)
	}
	if *got != want {
		t.Errorf("%s = %d, want %d", name, *got, want)
	}
}

// TestGetTunnelGET covers GetTunnel: a GET with path and query parameters.
func TestGetTunnelGET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want %q", r.Method, http.MethodGet)
		}
		if r.URL.Path != "/openapi/v1/omadac-1/sites/site-1/remoteAccess/tunnel" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/openapi/v1/omadac-1/sites/site-1/remoteAccess/tunnel")
		}
		if got := r.URL.Query().Get("page"); got != "1" {
			t.Errorf("page = %q, want %q", got, "1")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"errorCode":0,"msg":"OK","result":{"totalRows":7}}`)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	resp, err := c.GetTunnel(context.Background(), "omadac-1", "site-1", &GetTunnelParams{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("GetTunnel: %v", err)
	}
	var env OperationResponseNatTraversalTunnelGridVONatTraversalTunnelVO
	decodeBody(t, resp, &env)
	checkEnvelope(t, env.ErrorCode, env.Msg)
	if env.Result == nil {
		t.Fatal("result = nil, want non-nil")
	}
	checkTotalRows(t, env.Result.TotalRows, 7)
}

// TestAddTunnelPOST covers AddTunnel: a POST with a JSON request body.
func TestAddTunnelPOST(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want %q", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/openapi/v1/omadac-1/sites/site-1/remoteAccess/tunnel" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/openapi/v1/omadac-1/sites/site-1/remoteAccess/tunnel")
		}
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding request body: %v", err)
		} else if body.Name != "tun-1" {
			t.Errorf("name = %q, want %q", body.Name, "tun-1")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"errorCode":0,"msg":"OK","result":null}`)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	resp, err := c.AddTunnel(context.Background(), "omadac-1", "site-1", NatTraversalTunnelOpenApiVO{AppType: "HTTP", LocalAddress: "10.0.0.1", LocalPort: 8080, Name: "tun-1"})
	if err != nil {
		t.Fatalf("AddTunnel: %v", err)
	}
	var env OperationResponseWithoutResult
	decodeBody(t, resp, &env)
	checkEnvelope(t, env.ErrorCode, env.Msg)
}
