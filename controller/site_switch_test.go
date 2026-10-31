package controller

import (
	"context"
	"encoding/json"
	"errors"
	models "github.com/RonaldPhilipsen/omada/controller/models"
	"net/http"
	"testing"
)

func TestSwitches(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/switches" {
			if got := r.URL.Query().Get("deviceType"); got != "switch" {
				t.Errorf("deviceType = %q, want switch", got)
			}
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data": []map[string]any{
					{"mac": "AA:BB:CC:DD:EE:03", "name": "Rack models.Switch", "type": "switch", "status": 14, "statusCategory": 1},
				},
				"currentSize": 1,
				"totalRows":   1,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	switches, err := sc.Switches(context.Background())
	if err != nil {
		t.Fatalf("Switches() error = %v", err)
	}
	if len(switches) != 1 {
		t.Fatalf("Switches() returned %d switches, want 1", len(switches))
	}
	if switches[0].MAC() != "AA:BB:CC:DD:EE:03" || switches[0].Name() != "Rack models.Switch" {
		t.Errorf("Switches()[0] = %+v, want Rack models.Switch", switches[0])
	}
}

func TestSwitchEmptyID(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	if _, err := sc.Switch(context.Background(), ""); err == nil {
		t.Fatal("models.Switch() expected error for empty device ID")
	} else {
		var verr *models.ValidationError
		if !errors.As(err, &verr) {
			t.Fatalf("models.Switch() error = %v, want *models.ValidationError", err)
		}
	}
}

func TestSwitchesByMAC(t *testing.T) {
	var gotMac string
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/openapi/v2/"+testControllerID+"/sites/site-1/switches" {
			gotMac = r.URL.Query().Get("mac")
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data": []map[string]any{
					{"mac": "AA:BB:CC:DD:EE:03", "name": "Rack models.Switch", "type": "switch"},
				},
				"currentSize": 1,
				"totalRows":   1,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	switches, err := sc.SwitchesByMAC(context.Background(), []string{"AA:BB:CC:DD:EE:03", "AA:BB:CC:DD:EE:05"})
	if err != nil {
		t.Fatalf("SwitchesByMAC() error = %v", err)
	}
	if gotMac != "AA:BB:CC:DD:EE:03,AA:BB:CC:DD:EE:05" {
		t.Errorf("SwitchesByMAC mac = %q, want comma-joined MAC list", gotMac)
	}
	if len(switches) != 1 {
		t.Fatalf("SwitchesByMAC() returned %d switches, want 1", len(switches))
	}
	if _, err := sc.SwitchesByMAC(context.Background(), nil); err == nil {
		t.Fatal("SwitchesByMAC() expected error for empty MAC list")
	}
}

func TestSwitchPorts(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/switches/AA:BB:CC:DD:EE:03/portConfig" {
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data": []map[string]any{
					{"port": 1, "name": "Port 1", "type": 1, "pvid": 10},
				},
				"currentSize": 1,
				"totalRows":   1,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	ports, err := sc.SwitchPorts(context.Background(), "AA:BB:CC:DD:EE:03")
	if err != nil {
		t.Fatalf("SwitchPorts() error = %v", err)
	}
	if len(ports) != 1 {
		t.Fatalf("SwitchPorts() returned %d ports, want 1", len(ports))
	}
	if ports[0].Index() != 1 || ports[0].Name() != "Port 1" {
		t.Errorf("SwitchPorts()[0] = %+v, want port 1/Port 1", ports[0])
	}
}

func TestSetSwitchPortSettingsMerge(t *testing.T) {
	var patched map[string]any
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1":
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"id": "site-1", "name": "Office", "controllerVer": "6.2.10.18",
			}})
		case r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/switches/AA:BB:CC:DD:EE:03/portConfig":
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data": []map[string]any{
					{"port": 1, "name": "Port 1", "type": 1, "pvid": 10},
				},
				"currentSize": 1,
				"totalRows":   1,
			}})
		case r.Method == http.MethodPatch && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/switches/AA:BB:CC:DD:EE:03/portConfig":
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
	if err := sc.SetSwitchPortSettings(context.Background(), "AA:BB:CC:DD:EE:03", 1, models.PortTypeCopper, 20, []int{10}, nil, nil); err != nil {
		t.Fatalf("SetSwitchPortSettings() error = %v", err)
	}
	if patched == nil {
		t.Fatal("SetSwitchPortSettings() did not call the port config endpoint")
	}
	if patched["port"] != float64(1) || patched["pvid"] != float64(20) {
		t.Errorf("SetSwitchPortSettings body = %v, want merged port config", patched)
	}
}

func TestPortProfileCRUD(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/portProfiles":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode body: %v", err)
				return
			}
			if body["name"] == "" {
				t.Errorf("create body name = %v, want non-empty", body["name"])
			}
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"id": "profile-1", "name": "Profile A", "site": "site-1",
			}})
		case r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/portProfiles/profile-1":
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"id": "profile-1", "name": "Profile A", "site": "site-1",
			}})
		case r.Method == http.MethodPatch && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/portProfiles/profile-1":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode body: %v", err)
				return
			}
			if body["name"] == "" {
				t.Errorf("update body name = %v, want non-empty", body["name"])
			}
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"id": "profile-1", "name": "Profile B", "site": "site-1",
			}})
		case r.Method == http.MethodDelete && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/portProfiles/profile-1":
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"success": true}})
		case r.Method == http.MethodPost && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/portProfiles/profile-1/cmd/apply":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode body: %v", err)
				return
			}
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"success": true}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	if _, err := sc.CreatePortProfile(context.Background(), PortProfileOverrides{Name: "Profile A", Ports: []int{1, 2}}); err != nil {
		t.Fatalf("CreatePortProfile() error = %v", err)
	}
	if _, err := sc.PortProfile(context.Background(), "profile-1"); err != nil {
		t.Fatalf("models.PortProfile() error = %v", err)
	}
	if _, err := sc.UpdatePortProfile(context.Background(), "profile-1", PortProfileOverrides{Name: "Profile B"}); err != nil {
		t.Fatalf("UpdatePortProfile() error = %v", err)
	}
	if err := sc.DeletePortProfile(context.Background(), "profile-1"); err != nil {
		t.Fatalf("DeletePortProfile() error = %v", err)
	}
	if err := sc.ApplyPortProfile(context.Background(), "profile-1", []int{1, 2}); err != nil {
		t.Fatalf("ApplyPortProfile() error = %v", err)
	}
}

func TestPortConfigVersionGate(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1":
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"id": "site-1", "name": "Office", "controllerVer": "5.15.5.5",
			}})
		case r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/switches/AA:BB:CC:DD:EE:03/portConfig":
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data":        []map[string]any{{"port": 1, "name": "Port 1", "type": 1, "pvid": 10}},
				"currentSize": 1, "totalRows": 1,
			}})
		case r.Method == http.MethodPatch && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/switches/AA:BB:CC:DD:EE:03/portConfig":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode body: %v", err)
				return
			}
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"success": true}})
		case r.Method == http.MethodPatch && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/switch/AA:BB:CC:DD:EE:03/portConfig":
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"success": true}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	version, err := sc.SiteVersion(context.Background())
	if err != nil {
		t.Fatalf("SiteVersion() error = %v", err)
	}
	if version.Compare(models.ParseVersionMust("5.15")) <= 0 {
		t.Fatalf("SiteVersion() = %v, want the 5.15 fixture version", version)
	}
	if err := sc.SetSwitchPortSettings(context.Background(), "AA:BB:CC:DD:EE:03", 1, models.PortTypeCopper, 20, []int{10}, nil, nil); err != nil {
		t.Fatalf("SetSwitchPortSettings() error = %v", err)
	}
}

func TestGetSwitchPort(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/switches/AA:BB:CC:DD:EE:03/ports/3" {
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"id":                    "3",
				"port":                  3,
				"poe":                   1,
				"profileOverrideEnable": true,
				"profileName":           "Default",
				"dot1x":                 0,
				"lldpMedEnable":         true,
				"spanningTreeEnable":    false,
				"loopbackDetectEnable":  true,
				"portIsolationEnable":   false,
				"eeeEnable":             false,
				"flowControlEnable":     false,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	port, err := sc.GetSwitchPort(context.Background(), "AA:BB:CC:DD:EE:03", 3)
	if err != nil {
		t.Fatalf("GetSwitchPort() error = %v", err)
	}
	if port.PortID() != "3" {
		t.Errorf("GetSwitchPort() PortID = %q, want 3", port.PortID())
	}
	if port.PoEMode() != models.PoEModeEnabled {
		t.Errorf("GetSwitchPort() models.PoEMode = %v, want enabled", port.PoEMode())
	}
	if !port.HasProfileOverride() {
		t.Errorf("GetSwitchPort() HasProfileOverride = false, want true")
	}
}

func TestGetSwitchPortEmptyID(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	if _, err := sc.GetSwitchPort(context.Background(), "", 3); err == nil {
		t.Fatal("GetSwitchPort() expected error for empty device ID")
	} else {
		var verr *models.ValidationError
		if !errors.As(err, &verr) {
			t.Fatalf("GetSwitchPort() error = %v, want *models.ValidationError", err)
		}
	}
}

func TestGetSwitchPortMiss(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/switches/AA:BB:CC:DD:EE:03/ports/3" {
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	if _, err := sc.GetSwitchPort(context.Background(), "AA:BB:CC:DD:EE:03", 3); err == nil {
		t.Fatal("GetSwitchPort() expected error for missing port")
	} else {
		if !errors.Is(err, models.ErrInvalidDevice) {
			t.Fatalf("GetSwitchPort() error = %v, want wrapped %v", err, models.ErrInvalidDevice)
		}
	}
}
