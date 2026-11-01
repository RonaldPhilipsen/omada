package site_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RonaldPhilipsen/omada/cloud"
	"github.com/RonaldPhilipsen/omada/cloud/site"
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

// TestGetCustomerListGET covers GetCustomerList: a GET with path and query parameters.
func TestGetCustomerListGET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want %q", r.Method, http.MethodGet)
		}
		if r.URL.Path != "/openapi/v1/msp/msp-1/customers" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/openapi/v1/msp/msp-1/customers")
		}
		if got := r.URL.Query().Get("page"); got != "1" {
			t.Errorf("page = %q, want %q", got, "1")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"errorCode":0,"msg":"OK","result":{"totalRows":7}}`)
	}))
	t.Cleanup(srv.Close)

	c, err := site.NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	resp, err := c.GetCustomerList(context.Background(), "msp-1", &site.GetCustomerListParams{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("GetCustomerList: %v", err)
	}
	var env site.OperationResponseGridVOCustomerInfoOpenApiVO
	decodeBody(t, resp, &env)
	checkEnvelope(t, env.ErrorCode, env.Msg)
	if env.Result == nil {
		t.Fatal("result = nil, want non-nil")
	}
	checkTotalRows(t, env.Result.TotalRows, 7)
}

// TestMspBatchControllerPOST covers MspBatchController: a POST with a JSON request body.
func TestMspBatchControllerPOST(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want %q", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/openapi/v1/msp/msp-1/batch" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/openapi/v1/msp/msp-1/batch")
		}
		var body struct {
			Interrupt *bool `json:"interrupt"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding request body: %v", err)
		} else if body.Interrupt == nil || !*body.Interrupt {
			t.Errorf("interrupt = %v, want true", body.Interrupt)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"errorCode":0,"msg":"OK","result":{"response":[]}}`)
	}))
	t.Cleanup(srv.Close)

	interrupt := true
	c, err := site.NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	resp, err := c.MspBatchController(context.Background(), "msp-1", site.BatchRequestEntity{Interrupt: &interrupt})
	if err != nil {
		t.Fatalf("MspBatchController: %v", err)
	}
	var env site.OperationResponseBatchResponseEntity
	decodeBody(t, resp, &env)
	checkEnvelope(t, env.ErrorCode, env.Msg)
	if env.Result == nil {
		t.Fatal("result = nil, want non-nil")
	}
	if env.Result.Response == nil {
		t.Fatal("result.response = nil, want []")
	}
	if len(*env.Result.Response) != 0 {
		t.Errorf("len(result.response) = %d, want 0", len(*env.Result.Response))
	}
}

// TestSiteEnvelopeError builds an APIError from a non-zero envelope errorCode.
func TestSiteEnvelopeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"errorCode":-7131,"msg":"Controller ID not exist.","result":null}`)
	}))
	t.Cleanup(srv.Close)

	c, err := site.NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	resp, err := c.GetCustomerList(context.Background(), "msp-1", &site.GetCustomerListParams{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("GetCustomerList: %v", err)
	}
	var env site.OperationResponseGridVOCustomerInfoOpenApiVO
	decodeBody(t, resp, &env)
	if env.ErrorCode == nil {
		t.Fatal("errorCode = nil, want -7131")
	}
	if env.Msg == nil {
		t.Fatal("msg = nil, want a message")
	}
	apiErr := cloud.NewAPIError(int(*env.ErrorCode), *env.Msg)
	if !strings.Contains(apiErr.Error(), "-7131") {
		t.Errorf("APIError.Error() = %q, want it to contain %q", apiErr.Error(), "-7131")
	}
}

// TestSiteUnexpectedStatus asserts that a non-envelope 500 response is returned
// as-is: the generated client does not inspect status codes, so err is nil and
// the body is left for the caller to interpret.
func TestSiteUnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "<html>boom</html>")
	}))
	t.Cleanup(srv.Close)

	c, err := site.NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	resp, err := c.GetCustomerList(context.Background(), "msp-1", &site.GetCustomerListParams{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("GetCustomerList returned an error for a 500 response: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if string(body) != "<html>boom</html>" {
		t.Errorf("body = %q, want %q", string(body), "<html>boom</html>")
	}
}
