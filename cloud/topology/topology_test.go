package topology

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

// TestGetGridAvailableSsidsGET covers GetGridAvailableSsids: a GET with path and query parameters.
func TestGetGridAvailableSsidsGET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want %q", r.Method, http.MethodGet)
		}
		if r.URL.Path != "/openapi/v2/omadac-1/sites/site-1/topology/available-ssids" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/openapi/v2/omadac-1/sites/site-1/topology/available-ssids")
		}
		if got := r.URL.Query().Get("page"); got != "1" {
			t.Errorf("page = %q, want %q", got, "1")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"errorCode":0,"msg":"OK","result":{"ssids":[]}}`)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	resp, err := c.GetGridAvailableSsids(context.Background(), "omadac-1", "site-1", &GetGridAvailableSsidsParams{Page: 1, PageSize: 20, SearchKey: "x"})
	if err != nil {
		t.Fatalf("GetGridAvailableSsids: %v", err)
	}
	var env OperationResponseTopologyAvailableNetworkAndSSID
	decodeBody(t, resp, &env)
	checkEnvelope(t, env.ErrorCode, env.Msg)
	if env.Result == nil {
		t.Fatal("result = nil, want non-nil")
	}
	if env.Result.Ssids == nil {
		t.Fatal("result.ssids = nil, want []")
	}
	if len(*env.Result.Ssids) != 0 {
		t.Errorf("len(result.ssids) = %d, want 0", len(*env.Result.Ssids))
	}
}

// TestGetTopologyClientsByDevicesPOST covers GetTopologyClientsByDevices: a POST with a JSON request body.
func TestGetTopologyClientsByDevicesPOST(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want %q", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/openapi/v2/omadac-1/sites/site-1/topology/devices/all-clients" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/openapi/v2/omadac-1/sites/site-1/topology/devices/all-clients")
		}
		var body struct {
			DeviceMacs []struct {
				Mac string `json:"mac"`
			} `json:"deviceMacs"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding request body: %v", err)
		} else if len(body.DeviceMacs) != 1 || body.DeviceMacs[0].Mac != "aa:bb:cc:dd:ee:ff" {
			t.Errorf("deviceMacs = %+v, want one mac aa:bb:cc:dd:ee:ff", body.DeviceMacs)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"errorCode":0,"msg":"OK","result":[{"devMac":"aa:bb:cc:dd:ee:ff"}]}`)
	}))
	t.Cleanup(srv.Close)

	mac := "aa:bb:cc:dd:ee:ff"
	c, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	resp, err := c.GetTopologyClientsByDevices(context.Background(), "omadac-1", "site-1", GetTopologyClientsByDevicesJSONRequestBody{DeviceMacs: &[]ClientsQueryMacAndFilterType{{Mac: &mac}}})
	if err != nil {
		t.Fatalf("GetTopologyClientsByDevices: %v", err)
	}
	var env OperationResponseListTopologyDeviceClients
	decodeBody(t, resp, &env)
	checkEnvelope(t, env.ErrorCode, env.Msg)
	if env.Result == nil {
		t.Fatal("result = nil, want non-nil")
	}
	if len(*env.Result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(*env.Result))
	}
	checkString(t, "result[0].devMac", (*env.Result)[0].DevMac, "aa:bb:cc:dd:ee:ff")
}
