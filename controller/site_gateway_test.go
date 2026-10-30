package controller

import (
	"context"
	"encoding/json"
	"errors"
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
					{"mac": "AA:BB:CC:DD:EE:01", "name": "Main Gateway", "type": "gateway", "status": 14, "statusCategory": 1, "ip": "192.168.1.1"},
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
	if gateways[0].MAC() != "AA:BB:CC:DD:EE:01" || gateways[0].Name() != "Main Gateway" {
		t.Errorf("Gateways()[0] = %+v, want Main Gateway", gateways[0])
	}
}

func TestGatewayEmptyID(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	if _, err := sc.Gateway(context.Background(), ""); err == nil {
		t.Fatal("Gateway() expected error for empty device ID")
	} else {
		var verr *ValidationError
		if !errors.As(err, &verr) {
			t.Fatalf("Gateway() error = %v, want *ValidationError", err)
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
	if ports[0].Type() != GatewayPortTypeWAN {
		t.Errorf("GatewayPorts()[0].Type() = %v, want %v", ports[0].Type(), GatewayPortTypeWAN)
	}
	if ports[0].PoE() != PoEModeEnabled {
		t.Errorf("GatewayPorts()[0].PoE() = %v, want %v", ports[0].PoE(), PoEModeEnabled)
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
	if err := sc.SetGatewayPortSettings(context.Background(), "AA:BB:CC:DD:EE:01", 1, PoEModeEnabled, "2G", nil); err != nil {
		t.Fatalf("SetGatewayPortSettings() error = %v", err)
	}
	if patched == nil {
		t.Fatal("SetGatewayPortSettings() did not call the gateway port endpoint")
	}
	if patched["port"] != float64(1) || patched["poe"] != float64(int(PoEModeEnabled)) {
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
	if err := sc.SetGatewayPoE(context.Background(), "AA:BB:CC:DD:EE:01", 1, PoEModeEnabled); err != nil {
		t.Fatalf("SetGatewayPoE() error = %v", err)
	}
	if body["port"] != float64(1) || body["poe"] != float64(int(PoEModeEnabled)) {
		t.Errorf("SetGatewayPoE body = %v, want port 1 poe enabled", body)
	}
}
