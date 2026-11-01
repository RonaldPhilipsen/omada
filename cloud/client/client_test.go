package client

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

// TestGetGridSsidClientsGET covers GetGridSsidClients: a GET with path and query parameters.
func TestGetGridSsidClientsGET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want %q", r.Method, http.MethodGet)
		}
		if r.URL.Path != "/openapi/v1/omadac-1/sites/site-1/clientList/ssid-1" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/openapi/v1/omadac-1/sites/site-1/clientList/ssid-1")
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
	resp, err := c.GetGridSsidClients(context.Background(), "omadac-1", "site-1", "ssid-1", &GetGridSsidClientsParams{QueryDataVO: OpenApiQueryDataV2VO{Page: 1, PageSize: 20}})
	if err != nil {
		t.Fatalf("GetGridSsidClients: %v", err)
	}
	var env OperationResponseClientGridVOSsidClientVO
	decodeBody(t, resp, &env)
	checkEnvelope(t, env.ErrorCode, env.Msg)
	if env.Result == nil {
		t.Fatal("result = nil, want non-nil")
	}
	checkTotalRows(t, env.Result.TotalRows, 7)
}

// TestMspClientStatByDevicePOST covers MspClientStatByDevice: a POST with a JSON request body.
func TestMspClientStatByDevicePOST(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want %q", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/openapi/v1/msp/msp-1/clients/stat/devices" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/openapi/v1/msp/msp-1/clients/stat/devices")
		}
		var body struct {
			Devices []struct {
				Mac string `json:"mac"`
			} `json:"devices"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding request body: %v", err)
		} else if len(body.Devices) != 1 || body.Devices[0].Mac != "aa:bb:cc:dd:ee:ff" {
			t.Errorf("devices = %+v, want one device with mac aa:bb:cc:dd:ee:ff", body.Devices)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"errorCode":0,"msg":"OK","result":[{"clientNum":3}]}`)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	resp, err := c.MspClientStatByDevice(context.Background(), "msp-1", MspDevicesQueryOpenApiVO{Devices: []MspDeviceItem{{CustomerId: "cust-1", Mac: "aa:bb:cc:dd:ee:ff", SiteId: "site-1"}}})
	if err != nil {
		t.Fatalf("MspClientStatByDevice: %v", err)
	}
	var env OperationResponseListMspDeviceClientNumOpenApiVO
	decodeBody(t, resp, &env)
	checkEnvelope(t, env.ErrorCode, env.Msg)
	if env.Result == nil {
		t.Fatal("result = nil, want non-nil")
	}
	if len(*env.Result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(*env.Result))
	}
	checkInt32(t, "result[0].clientNum", (*env.Result)[0].ClientNum, 3)
}
