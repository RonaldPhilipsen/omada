package insight

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

// TestGetMspApHealthDetailGET covers GetMspApHealthDetail: a GET with path and query parameters.
func TestGetMspApHealthDetailGET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want %q", r.Method, http.MethodGet)
		}
		if r.URL.Path != "/openapi/v1/msp/msp-1/customers/cust-1/sites/site-1/eaps/ap-1/health/detail" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/openapi/v1/msp/msp-1/customers/cust-1/sites/site-1/eaps/ap-1/health/detail")
		}
		if got := r.URL.Query().Get("start"); got != "1000" {
			t.Errorf("start = %q, want %q", got, "1000")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"errorCode":0,"msg":"OK","result":{"score":88}}`)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	resp, err := c.GetMspApHealthDetail(context.Background(), "msp-1", "cust-1", "site-1", "ap-1", &GetMspApHealthDetailParams{Start: 1000, End: 2000})
	if err != nil {
		t.Fatalf("GetMspApHealthDetail: %v", err)
	}
	var env OperationResponseApHealthDetailVO
	decodeBody(t, resp, &env)
	checkEnvelope(t, env.ErrorCode, env.Msg)
	if env.Result == nil {
		t.Fatal("result = nil, want non-nil")
	}
	checkInt32(t, "result.score", env.Result.Score, 88)
}

// TestExportRogueApsGlobalPOST covers ExportRogueApsGlobal: a POST with a JSON request body.
func TestExportRogueApsGlobalPOST(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want %q", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/openapi/v1/omadac-1/files/neighbors" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/openapi/v1/omadac-1/files/neighbors")
		}
		var body struct {
			Format *int32 `json:"format"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding request body: %v", err)
		} else if body.Format == nil || *body.Format != 1 {
			t.Errorf("format = %v, want 1", body.Format)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"errorCode":0,"msg":"OK","result":null}`)
	}))
	t.Cleanup(srv.Close)

	format := int32(1)
	c, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	resp, err := c.ExportRogueApsGlobal(context.Background(), "omadac-1", RequiredParametersForExportingRogueAPScanResults{Format: &format})
	if err != nil {
		t.Fatalf("ExportRogueApsGlobal: %v", err)
	}
	var env OperationResponseWithoutResult
	decodeBody(t, resp, &env)
	checkEnvelope(t, env.ErrorCode, env.Msg)
}
