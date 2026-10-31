package site_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/RonaldPhilipsen/omada/controller/internal/testutil"
	models "github.com/RonaldPhilipsen/omada/controller/models"
	"net/http"
	"testing"
)

func TestVpnPoliciesAllPages(t *testing.T) {
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/openapi/v2/"+testutil.TestControllerID+"/sites/site-1/vpn/client-to-site-vpn-servers" {
			var data []map[string]any
			if r.URL.Query().Get("page") == "2" {
				for i := 0; i < 30; i++ {
					id := fmt.Sprintf("vpn-server-%d", 100+i)
					data = append(data, map[string]any{"id": id, "name": "Server " + id, "status": true})
				}
			} else {
				for i := 0; i < 100; i++ {
					id := fmt.Sprintf("vpn-server-%d", i)
					data = append(data, map[string]any{"id": id, "name": "Server " + id, "status": true})
				}
			}
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data":        data,
				"currentSize": 100,
				"totalRows":   130,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := mustSiteClient(t, cl)
	policies, err := sc.VpnPolicies(context.Background())
	if err != nil {
		t.Fatalf("VpnPolicies() error = %v", err)
	}
	if len(policies) != 130 {
		t.Fatalf("VpnPolicies() returned %d policies, want 130", len(policies))
	}
}

func TestVpnPoliciesSkipsUnavailableCategory(t *testing.T) {
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/openapi/v2/"+testutil.TestControllerID+"/sites/site-1/vpn/client-to-site-vpn-servers" {
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data":        []map[string]any{{"id": "vpn-server-1", "name": "HQ IPSec", "status": true}, {"id": "vpn-server-2", "name": "Branch", "status": true}},
				"currentSize": 2,
				"totalRows":   2,
			}})
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/openapi/v2/"+testutil.TestControllerID+"/sites/site-1/vpn/client-to-site-vpn-clients" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/openapi/v2/"+testutil.TestControllerID+"/sites/site-1/vpn/site-to-site-vpns" {
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data":        []map[string]any{{"id": "vpn-site-1", "name": "Branch", "status": true}},
				"currentSize": 1,
				"totalRows":   1,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := mustSiteClient(t, cl)
	policies, err := sc.VpnPolicies(context.Background())
	if err != nil {
		t.Fatalf("VpnPolicies() error = %v", err)
	}
	if len(policies) != 3 {
		t.Fatalf("VpnPolicies() returned %d policies, want 3", len(policies))
	}
}

func TestVpnPolicyMiss(t *testing.T) {
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/openapi/v2/"+testutil.TestControllerID+"/sites/site-1/client-to-site-vpn-servers" {
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data":        []map[string]any{{"id": "vpn-server-1", "name": "HQ IPSec", "status": true}},
				"currentSize": 1,
				"totalRows":   1,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := mustSiteClient(t, cl)
	if _, err := sc.VpnPolicy(context.Background(), "vpn-server-missing"); err == nil {
		t.Fatal("models.VpnPolicy() expected error for unknown policy ID")
	} else {
		var verr *models.ValidationError
		if !errors.As(err, &verr) {
			t.Fatalf("models.VpnPolicy() error = %v, want *models.ValidationError", err)
		}
	}
}

func TestSetVpnPolicyEnabled(t *testing.T) {
	var patched map[string]any
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPatch && r.URL.Path == "/openapi/v1/"+testutil.TestControllerID+"/sites/site-1/vpn/vpn-server-1/status":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode body: %v", err)
				return
			}
			patched = body
			if patched["status"] != true {
				t.Errorf("patch body status = %v, want true", patched["status"])
			}
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"id": "vpn-server-1", "name": "HQ IPSec", "status": true,
			}})
		case r.Method == http.MethodGet && r.URL.Path == "/openapi/v2/"+testutil.TestControllerID+"/sites/site-1/client-to-site-vpn-servers":
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data":        []map[string]any{{"id": "vpn-server-1", "name": "HQ IPSec", "status": true}},
				"currentSize": 1,
				"totalRows":   1,
			}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	sc := mustSiteClient(t, cl)
	policy, err := sc.SetVpnPolicyEnabled(context.Background(), "vpn-server-1", true)
	if err != nil {
		t.Fatalf("SetVpnPolicyEnabled() error = %v", err)
	}
	if policy.PolicyID() != "vpn-server-1" {
		t.Errorf("SetVpnPolicyEnabled() policy ID = %q, want vpn-server-1", policy.PolicyID())
	}
	if !policy.Enabled() {
		t.Error("SetVpnPolicyEnabled() policy not enabled")
	}
	if _, err := sc.SetVpnPolicyEnabled(context.Background(), "", true); err == nil {
		t.Fatal("SetVpnPolicyEnabled() expected error for empty policy ID")
	} else {
		var verr *models.ValidationError
		if !errors.As(err, &verr) {
			t.Fatalf("SetVpnPolicyEnabled() error = %v, want *models.ValidationError", err)
		}
	}
}
