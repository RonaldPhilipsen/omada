package site_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/RonaldPhilipsen/omada/controller/internal/testutil"
	models "github.com/RonaldPhilipsen/omada/controller/models"
	"github.com/RonaldPhilipsen/omada/controller/site"
	"net/http"
	"strings"
	"testing"
)

func TestLanNetworks(t *testing.T) {
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/"+testutil.TestControllerID+"/api/v2/sites/site-1/lanNetworks" {
			if got := r.URL.Query().Get("type"); got != "lan" {
				t.Errorf("type = %q, want lan", got)
			}
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data": []map[string]any{
					{"id": "net-1", "name": "Guest", "purpose": "vlan", "vlan": 10},
				},
				"currentSize": 1,
				"totalRows":   1,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := mustSiteClient(t, cl)
	networks, err := sc.LanNetworks(context.Background())
	if err != nil {
		t.Fatalf("LanNetworks() error = %v", err)
	}
	if len(networks) != 1 {
		t.Fatalf("LanNetworks() returned %d networks, want 1", len(networks))
	}
	if networks[0].ID() != "net-1" || networks[0].Name() != "Guest" {
		t.Errorf("LanNetworks()[0] = %+v, want net-1/Guest", networks[0])
	}
}

func TestLanNetworkEmptyID(t *testing.T) {
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	sc := mustSiteClient(t, cl)
	if _, err := sc.LanNetwork(context.Background(), ""); err == nil {
		t.Fatal("models.LanNetwork() expected error for empty network ID")
	} else {
		var verr *models.ValidationError
		if !errors.As(err, &verr) {
			t.Fatalf("models.LanNetwork() error = %v, want *models.ValidationError", err)
		}
	}
}

func TestCreateLanNetworkValidation(t *testing.T) {
	if _, err := site.NewLanNetworkCreate("", 5); err == nil {
		t.Fatal("site.NewLanNetworkCreate() expected error for empty name")
	}
	if _, err := site.NewLanNetworkCreate("Guest", 0); err == nil {
		t.Fatal("site.NewLanNetworkCreate() expected error for VLAN 0")
	}
	if _, err := site.NewLanNetworkCreate("Guest", 1); err == nil {
		t.Fatal("site.NewLanNetworkCreate() expected error for protected VLAN 1")
	}
	if _, err := site.NewLanNetworkCreate("Guest", 4090); err != nil {
		t.Errorf("site.NewLanNetworkCreate() error = %v, want 4090 to be valid", err)
	}
}

func TestCreateLanNetwork(t *testing.T) {
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/"+testutil.TestControllerID+"/api/v2/sites/site-1/lanNetworks" {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode body: %v", err)
				return
			}
			if body["name"] != "Guest" || body["vlan"] != float64(10) {
				t.Errorf("create body = %v, want name Guest vlan 10", body)
			}
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"id": "net-1", "name": "Guest", "vlan": 10,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := mustSiteClient(t, cl)
	n, err := sc.CreateLanNetwork(context.Background(), site.LanNetworkCreate{Name: "Guest", VLAN: 10})
	if err != nil {
		t.Fatalf("CreateLanNetwork() error = %v", err)
	}
	if n.ID() != "net-1" || n.Name() != "Guest" {
		t.Errorf("CreateLanNetwork() = %+v, want net-1/Guest", n)
	}
}

func TestUpdateLanNetworkPartial(t *testing.T) {
	var patched map[string]any
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == "/"+testutil.TestControllerID+"/api/v2/sites/site-1/lanNetworks/net-1" {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode body: %v", err)
				return
			}
			patched = body
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"id": "net-1", "name": "Guest", "vlan": 20,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := mustSiteClient(t, cl)
	vlan := 20
	if _, err := sc.UpdateLanNetwork(context.Background(), site.LanNetworkUpdate{NetworkID: "net-1", VLAN: &vlan}); err != nil {
		t.Fatalf("UpdateLanNetwork() error = %v", err)
	}
	if patched == nil {
		t.Fatal("UpdateLanNetwork() did not call the update endpoint")
	}
	if patched["vlan"] != float64(20) {
		t.Errorf("UpdateLanNetwork body = %v, want vlan 20", patched)
	}
	if _, err := sc.UpdateLanNetwork(context.Background(), site.LanNetworkUpdate{NetworkID: "net-1"}); err == nil {
		t.Fatal("UpdateLanNetwork() expected error for empty update")
	}
}

func TestVlanCreatePreviewAndWrite(t *testing.T) {
	var previewBodies []map[string]any
	var createBodies []map[string]any
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/openapi/v2/"+testutil.TestControllerID+"/sites/site-1/vlan/preview":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode preview body: %v", err)
				return
			}
			previewBodies = append(previewBodies, body)
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"success": true}})
		case r.Method == http.MethodPost && r.URL.Path == "/openapi/v2/"+testutil.TestControllerID+"/sites/site-1/vlans":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode create body: %v", err)
				return
			}
			createBodies = append(createBodies, body)
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"id": "net-1", "name": "Guest", "vlan": 10,
			}})
		case r.Method == http.MethodGet && r.URL.Path == "/openapi/v2/"+testutil.TestControllerID+"/sites/site-1/vlans":
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data": []map[string]any{{"vlan": 10}},
			}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	sc := mustSiteClient(t, cl)
	create, err := models.NewVlanCreate("Guest", 10)
	if err != nil {
		t.Fatalf("models.NewVlanCreate() error = %v", err)
	}
	if _, err := sc.PreviewVlanCreate(context.Background(), create); err != nil {
		t.Fatalf("PreviewVlanCreate() error = %v", err)
	}
	if len(previewBodies) != 1 || previewBodies[0]["vlans"] != "10" {
		t.Errorf("preview bodies = %v, want vlans=10", previewBodies)
	}
	if _, err := sc.VlanCreate(context.Background(), create); err != nil {
		t.Fatalf("models.VlanCreate() error = %v", err)
	}
	if len(createBodies) != 1 {
		t.Fatalf("create calls = %d, want 1", len(createBodies))
	}
}

func TestVlanWriteSuccessReadback(t *testing.T) {
	var writes int
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/openapi/v2/"+testutil.TestControllerID+"/sites/site-1/vlans":
			writes++
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"success": true}})
		case r.Method == http.MethodGet && r.URL.Path == "/openapi/v2/"+testutil.TestControllerID+"/sites/site-1/vlans":
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data": []map[string]any{{"vlan": 10}},
			}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	sc := mustSiteClient(t, cl)
	create, err := models.NewVlanCreate("Guest", 10)
	if err != nil {
		t.Fatalf("models.NewVlanCreate() error = %v", err)
	}
	if _, err := sc.VlanCreate(context.Background(), create); err != nil {
		t.Fatalf("models.VlanCreate() error = %v", err)
	}
	if writes != 1 {
		t.Errorf("models.VlanCreate writes = %d, want 1", writes)
	}
}

func TestVlanWriteReadbackFailure(t *testing.T) {
	var writes int
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/openapi/v2/"+testutil.TestControllerID+"/sites/site-1/vlans":
			writes++
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"success": true}})
		case r.Method == http.MethodGet && r.URL.Path == "/openapi/v2/"+testutil.TestControllerID+"/sites/site-1/vlans":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	sc := mustSiteClient(t, cl)
	create, err := models.NewVlanCreate("Guest", 10)
	if err != nil {
		t.Fatalf("models.NewVlanCreate() error = %v", err)
	}
	_, err = sc.VlanCreate(context.Background(), create)
	if err == nil {
		t.Fatal("models.VlanCreate() expected read-back failure error")
	}
	if !strings.Contains(err.Error(), "write may have succeeded; inspect controller state before retrying") {
		t.Errorf("models.VlanCreate() error = %v, want read-back guidance", err)
	}
	if writes != 1 {
		t.Errorf("models.VlanCreate writes = %d, want 1 (no auto-retry)", writes)
	}
}

func TestVlanDelete(t *testing.T) {
	var deletes int
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/openapi/v2/"+testutil.TestControllerID+"/sites/site-1/vlan/preview":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode preview body: %v", err)
				return
			}
			if body["vlan"] != float64(10) {
				t.Errorf("preview body vlan = %v, want 10", body["vlan"])
			}
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"success": true}})
		case r.Method == http.MethodDelete && r.URL.Path == "/openapi/v2/"+testutil.TestControllerID+"/sites/site-1/vlans/10":
			deletes++
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"success": true}})
		case r.Method == http.MethodGet && r.URL.Path == "/openapi/v2/"+testutil.TestControllerID+"/sites/site-1/vlans":
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data": []map[string]any{{"vlan": 10}},
			}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	sc := mustSiteClient(t, cl)
	if _, err := sc.PreviewVlanDelete(context.Background(), site.VlanDelete{VLAN: 10}); err != nil {
		t.Fatalf("PreviewVlanDelete() error = %v", err)
	}
	if _, err := sc.VlanDelete(context.Background(), site.VlanDelete{VLAN: 10}); err != nil {
		t.Fatalf("VlanDelete() error = %v", err)
	}
	if deletes != 1 {
		t.Errorf("site.VlanDelete calls = %d, want 1", deletes)
	}
}
