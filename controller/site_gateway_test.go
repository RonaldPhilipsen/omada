package controller

import (
	"context"
	"encoding/json"
	"errors"
	models "github.com/RonaldPhilipsen/omada/controller/models"
	"net/http"
	"testing"
)

func TestGateways(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/gateways" {
			if got := r.URL.Query().Get("deviceType"); got != "gateway" {
				t.Errorf("deviceType = %q, want gateway", got)
			}
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data": []map[string]any{
					{"mac": "AA:BB:CC:DD:EE:01", "name": "Main models.Gateway", "type": "gateway", "status": 14, "statusCategory": 1, "ip": "192.168.1.1"},
				},
				"currentSize": 1,
				"totalRows":   1,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	gateways, err := sc.Gateways(context.Background())
	if err != nil {
		t.Fatalf("Gateways() error = %v", err)
	}
	if len(gateways) != 1 {
		t.Fatalf("Gateways() returned %d gateways, want 1", len(gateways))
	}
	if gateways[0].MAC() != "AA:BB:CC:DD:EE:01" || gateways[0].Name() != "Main models.Gateway" {
		t.Errorf("Gateways()[0] = %+v, want Main models.Gateway", gateways[0])
	}
}

func TestGatewayEmptyID(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	if _, err := sc.Gateway(context.Background(), ""); err == nil {
		t.Fatal("models.Gateway() expected error for empty device ID")
	} else {
		var verr *models.ValidationError
		if !errors.As(err, &verr) {
			t.Fatalf("models.Gateway() error = %v, want *models.ValidationError", err)
		}
	}
}

func TestGatewayPorts(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/gateways/AA:BB:CC:DD:EE:01/portConfig" {
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data": []map[string]any{
					{"port": 1, "type": 0, "poe": 1, "speed": "1G"},
				},
				"currentSize": 1,
				"totalRows":   1,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	ports, err := sc.GatewayPorts(context.Background(), "AA:BB:CC:DD:EE:01")
	if err != nil {
		t.Fatalf("GatewayPorts() error = %v", err)
	}
	if len(ports) != 1 {
		t.Fatalf("GatewayPorts() returned %d ports, want 1", len(ports))
	}
	if ports[0].Index() != 1 {
		t.Errorf("GatewayPorts()[0].Index() = %d, want 1", ports[0].Index())
	}
	if ports[0].Type() != models.GatewayPortTypeWAN {
		t.Errorf("GatewayPorts()[0].Type() = %v, want %v", ports[0].Type(), models.GatewayPortTypeWAN)
	}
	if ports[0].PoE() != models.PoEModeEnabled {
		t.Errorf("GatewayPorts()[0].PoE() = %v, want %v", ports[0].PoE(), models.PoEModeEnabled)
	}
	if ports[0].Speed() != "1G" {
		t.Errorf("GatewayPorts()[0].Speed() = %q, want 1G", ports[0].Speed())
	}
}

func TestSetGatewayPortSettingsMerge(t *testing.T) {
	var patched map[string]any
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/gateways/AA:BB:CC:DD:EE:01/portConfig":
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data": []map[string]any{
					{"port": 1, "type": 0, "poe": 1, "speed": "1G"},
				},
				"currentSize": 1,
				"totalRows":   1,
			}})
		case r.Method == http.MethodPatch && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/gateways/AA:BB:CC:DD:EE:01/portConfig":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode body: %v", err)
				return
			}
			patched = body
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"success": true}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	if err := sc.SetGatewayPortSettings(context.Background(), "AA:BB:CC:DD:EE:01", 1, models.PoEModeEnabled, "2G", nil); err != nil {
		t.Fatalf("SetGatewayPortSettings() error = %v", err)
	}
	if patched == nil {
		t.Fatal("SetGatewayPortSettings() did not call the gateway port endpoint")
	}
	if patched["port"] != float64(1) || patched["poe"] != float64(int(models.PoEModeEnabled)) {
		t.Errorf("SetGatewayPortSettings body = %v, want merged gateway port config", patched)
	}
}

func TestSetGatewayPoE(t *testing.T) {
	var body map[string]any
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/gateways/AA:BB:CC:DD:EE:01/portConfig" {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode body: %v", err)
				return
			}
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"success": true}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	if err := sc.SetGatewayPoE(context.Background(), "AA:BB:CC:DD:EE:01", 1, models.PoEModeEnabled); err != nil {
		t.Fatalf("SetGatewayPoE() error = %v", err)
	}
	if body["port"] != float64(1) || body["poe"] != float64(int(models.PoEModeEnabled)) {
		t.Errorf("SetGatewayPoE body = %v, want port 1 poe enabled", body)
	}
}

func TestGetGatewayPort(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/gateways/AA:BB:CC:DD:EE:01" {
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"id": "AA:BB:CC:DD:EE:01",
				"portConfigs": []map[string]any{
					{"port": 1, "name": "WAN", "type": 1},
					{"port": 2, "name": "LAN", "type": 2},
				},
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	cfg, err := sc.GetGatewayPort(context.Background(), "AA:BB:CC:DD:EE:01", 1)
	if err != nil {
		t.Fatalf("GetGatewayPort() error = %v", err)
	}
	if cfg.PortNumber() != 1 {
		t.Errorf("GetGatewayPort() = %+v, want port 1", cfg)
	}
}

func TestGetGatewayPortEmptyID(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	if _, err := sc.GetGatewayPort(context.Background(), "", 1); err == nil {
		t.Fatal("GetGatewayPort() expected error for empty device ID")
	} else {
		var verr *models.ValidationError
		if !errors.As(err, &verr) {
			t.Fatalf("GetGatewayPort() error = %v, want *models.ValidationError", err)
		}
	}
}

func TestGetGatewayPortMiss(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/gateways/AA:BB:CC:DD:EE:01" {
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"id": "AA:BB:CC:DD:EE:01",
				"portConfigs": []map[string]any{
					{"port": 1, "name": "WAN", "type": 1},
				},
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	if _, err := sc.GetGatewayPort(context.Background(), "AA:BB:CC:DD:EE:01", 9); err == nil {
		t.Fatal("GetGatewayPort() expected error for missing port")
	} else {
		var verr *models.ValidationError
		if !errors.As(err, &verr) {
			t.Fatalf("GetGatewayPort() error = %v, want *models.ValidationError", err)
		}
	}
}

func TestSetGatewayWanPortConnectState(t *testing.T) {
	var body map[string]any
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/gateways/AA:BB:CC:DD:EE:01/internetState" {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode body: %v", err)
				return
			}
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"port":          1,
				"internetState": 1,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	status, err := sc.SetGatewayWanPortConnectState(context.Background(), "AA:BB:CC:DD:EE:01", 1, true)
	if err != nil {
		t.Fatalf("SetGatewayWanPortConnectState() error = %v", err)
	}
	if body["portId"] != float64(1) || body["operation"] != float64(1) {
		t.Errorf("SetGatewayWanPortConnectState body = %v, want portId 1 operation 1", body)
	}
	if !status.WANConnected() {
		t.Errorf("SetGatewayWanPortConnectState() WANConnected = false, want true")
	}

	cl2 := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/gateways/AA:BB:CC:DD:EE:01/internetState" {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode body: %v", err)
				return
			}
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"port":          1,
				"internetState": 0,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc2 := &SiteClient{c: cl2.c, site: "site-1"}
	status2, err := sc2.SetGatewayWanPortConnectState(context.Background(), "AA:BB:CC:DD:EE:01", 1, false)
	if err != nil {
		t.Fatalf("SetGatewayWanPortConnectState() error = %v", err)
	}
	if body["operation"] != float64(0) {
		t.Errorf("SetGatewayWanPortConnectState body operation = %v, want 0", body["operation"])
	}
	if status2.WANConnected() {
		t.Errorf("SetGatewayWanPortConnectState() WANConnected = true, want false")
	}
}

func TestSetGatewayWanPortConnectStateEmptyID(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	if _, err := sc.SetGatewayWanPortConnectState(context.Background(), "", 1, true); err == nil {
		t.Fatal("SetGatewayWanPortConnectState() expected error for empty device ID")
	} else {
		var verr *models.ValidationError
		if !errors.As(err, &verr) {
			t.Fatalf("SetGatewayWanPortConnectState() error = %v, want *models.ValidationError", err)
		}
	}
}
