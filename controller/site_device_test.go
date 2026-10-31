package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestDevices(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/devices" {
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data": []map[string]any{
					{"mac": "AA:BB:CC:DD:EE:01", "name": "Main Gateway", "type": "gateway", "status": 14, "statusCategory": 1, "ip": "192.168.1.1"},
					{"mac": "AA:BB:CC:DD:EE:03", "name": "Rack Switch", "type": "switch", "status": 14, "statusCategory": 1, "ip": "192.168.1.2"},
				},
				"currentSize": 2,
				"totalRows":   2,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	devices, err := sc.Devices(context.Background())
	if err != nil {
		t.Fatalf("Devices() error = %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("Devices() returned %d devices, want 2", len(devices))
	}
	if devices[0].MAC() != "AA:BB:CC:DD:EE:01" || devices[0].Name() != "Main Gateway" {
		t.Errorf("Devices()[0] = %+v, want Main Gateway", devices[0])
	}
	if devices[1].MAC() != "AA:BB:CC:DD:EE:03" || devices[1].Name() != "Rack Switch" {
		t.Errorf("Devices()[1] = %+v, want Rack Switch", devices[1])
	}
}

func TestDeviceEmptyID(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	if _, err := sc.Device(context.Background(), ""); err == nil {
		t.Fatal("Device() expected error for empty device ID")
	} else {
		var verr *ValidationError
		if !errors.As(err, &verr) {
			t.Fatalf("Device() error = %v, want *ValidationError", err)
		}
	}
}

func TestDeviceByMACMiss(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/openapi/v2/"+testControllerID+"/sites/site-1/devices" && r.URL.Query().Get("mac") == "AA:BB:CC:DD:EE:99" {
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	_, err := sc.DeviceByMAC(context.Background(), "AA:BB:CC:DD:EE:99")
	if err == nil {
		t.Fatal("DeviceByMAC() expected error for unknown MAC")
	}
	if !errors.Is(err, ErrInvalidDevice) {
		t.Fatalf("DeviceByMAC() error = %v, want ErrInvalidDevice", err)
	}
}

func TestSiteTopology(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/topology" {
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"topologyArray": []map[string]any{
					{"type": "gateway", "mac": "AA:BB:CC:DD:EE:01", "name": "Main Gateway", "statusCategory": 1},
				},
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	topo, err := sc.Topology(context.Background())
	if err != nil {
		t.Fatalf("Topology() error = %v", err)
	}
	roots := topo.Roots()
	if len(roots) != 1 {
		t.Fatalf("Topology() roots = %d, want 1", len(roots))
	}
	if roots[0].MAC() != "AA:BB:CC:DD:EE:01" || roots[0].Name() != "Main Gateway" {
		t.Errorf("Topology()[0] = %+v, want Main Gateway", roots[0])
	}
}

func TestTopologyNodeMiss(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/topology/net-1" {
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	if _, err := sc.TopologyNode(context.Background(), "net-1"); err == nil {
		t.Fatal("TopologyNode() expected error for missing node")
	} else {
		var verr *ValidationError
		if !errors.As(err, &verr) {
			t.Fatalf("TopologyNode() error = %v, want *ValidationError", err)
		}
	}
}

func TestSetDeviceLED(t *testing.T) {
	var body map[string]any
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/devices/AA:BB:CC:DD:EE:01/led" {
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
	if err := sc.SetDeviceLED(context.Background(), "AA:BB:CC:DD:EE:01", LedSettingOn, 30*time.Second); err != nil {
		t.Fatalf("SetDeviceLED() error = %v", err)
	}
	if body["state"] != float64(int(LedSettingOn)) {
		t.Errorf("SetDeviceLED body state = %v, want %v", body["state"], int(LedSettingOn))
	}
	if body["duration"] != float64(30) {
		t.Errorf("SetDeviceLED body duration = %v, want 30", body["duration"])
	}
}

func TestRebootRestoreReset(t *testing.T) {
	var paths []string
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/devices/AA:BB:CC:DD:EE:01/cmd/reboot":
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"rebootTime": 300}})
		case r.Method == http.MethodPost && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/devices/AA:BB:CC:DD:EE:01/cmd/restore":
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"success": true}})
		case r.Method == http.MethodPost && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/devices/AA:BB:CC:DD:EE:01/cmd/factoryReset":
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"success": true}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	if err := sc.RebootDevice(context.Background(), "AA:BB:CC:DD:EE:01"); err != nil {
		t.Fatalf("RebootDevice() error = %v", err)
	}
	if err := sc.RestoreDevice(context.Background(), "AA:BB:CC:DD:EE:01"); err != nil {
		t.Fatalf("RestoreDevice() error = %v", err)
	}
	if err := sc.FactoryResetDevice(context.Background(), "AA:BB:CC:DD:EE:01"); err != nil {
		t.Fatalf("FactoryResetDevice() error = %v", err)
	}
	found := map[string]bool{}
	for _, p := range paths {
		found[p] = true
	}
	for _, want := range []string{
		"POST /" + testControllerID + "/api/v2/sites/site-1/devices/AA:BB:CC:DD:EE:01/cmd/reboot",
		"POST /" + testControllerID + "/api/v2/sites/site-1/devices/AA:BB:CC:DD:EE:01/cmd/restore",
		"POST /" + testControllerID + "/api/v2/sites/site-1/devices/AA:BB:CC:DD:EE:01/cmd/factoryReset",
	} {
		if !found[want] {
			t.Errorf("missing device command call %q, got %v", want, paths)
		}
	}
}

func TestUpgradeFirmware(t *testing.T) {
	var bodies []map[string]any
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/cmd/upgrade" {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode body: %v", err)
				return
			}
			bodies = append(bodies, body)
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"success": true}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	if err := sc.UpgradeDeviceFirmware(context.Background(), []string{"AA:BB:CC:DD:EE:01"}); err != nil {
		t.Fatalf("UpgradeDeviceFirmware() error = %v", err)
	}
	if len(bodies) != 1 {
		t.Fatalf("upgrade calls = %d, want 1", len(bodies))
	}
	ids, ok := bodies[0]["deviceIds"].([]any)
	if !ok || len(ids) != 1 || ids[0] != "AA:BB:CC:DD:EE:01" {
		t.Errorf("upgrade deviceIds = %v, want [AA:BB:CC:DD:EE:01]", bodies[0]["deviceIds"])
	}
	if err := sc.UpgradeDeviceFirmware(context.Background(), nil); err == nil {
		t.Fatal("UpgradeDeviceFirmware() expected error for empty device IDs")
	}
}

func TestGetFirmwareUpdates(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/upgrade" {
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data": []map[string]any{
					{"deviceId": "AA:BB:CC:DD:EE:01", "curFwVer": "6.2.0.0", "lastFwVer": "6.2.1.0", "fwReleaseLog": "notes"},
				},
				"currentSize": 1,
				"totalRows":   1,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	updates, err := sc.GetFirmwareUpdates(context.Background())
	if err != nil {
		t.Fatalf("GetFirmwareUpdates() error = %v", err)
	}
	if len(updates) != 1 {
		t.Fatalf("GetFirmwareUpdates() returned %d updates, want 1", len(updates))
	}
	if updates[0].CurrentVersion() != "6.2.0.0" {
		t.Errorf("GetFirmwareUpdates()[0].CurrentVersion() = %q, want 6.2.0.0", updates[0].CurrentVersion())
	}
	if updates[0].LatestVersion() != "6.2.1.0" {
		t.Errorf("GetFirmwareUpdates()[0].LatestVersion() = %q, want 6.2.1.0", updates[0].LatestVersion())
	}
}

func TestUpdateFirmwareChannel(t *testing.T) {
	var body map[string]any
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/upgrade" {
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
	if err := sc.UpdateFirmwareChannel(context.Background(), UpdateSectionHardware, "beta"); err != nil {
		t.Fatalf("UpdateFirmwareChannel() error = %v", err)
	}
	if body["section"] != "hardware" {
		t.Errorf("UpdateFirmwareChannel body section = %v, want hardware", body["section"])
	}
	if body["channel"] != "beta" {
		t.Errorf("UpdateFirmwareChannel body channel = %v, want beta", body["channel"])
	}
}

func TestGetFirmwareDetails(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/devices/AA:BB:CC:DD:EE:03/firmware" {
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"curFwVer":     "6.2.0.0",
				"lastFwVer":    "6.2.1.0",
				"fwReleaseLog": "Security fixes",
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	update, err := sc.GetFirmwareDetails(context.Background(), "AA:BB:CC:DD:EE:03")
	if err != nil {
		t.Fatalf("GetFirmwareDetails() error = %v", err)
	}
	if update.CurrentVersion() != "6.2.0.0" {
		t.Errorf("GetFirmwareDetails() CurrentVersion = %q, want 6.2.0.0", update.CurrentVersion())
	}
	if update.LatestVersion() != "6.2.1.0" {
		t.Errorf("GetFirmwareDetails() LatestVersion = %q, want 6.2.1.0", update.LatestVersion())
	}
	if update.ReleaseNotes() != "Security fixes" {
		t.Errorf("GetFirmwareDetails() ReleaseNotes = %q, want Security fixes", update.ReleaseNotes())
	}
}

func TestGetFirmwareDetailsEmptyID(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	if _, err := sc.GetFirmwareDetails(context.Background(), ""); err == nil {
		t.Fatal("GetFirmwareDetails() expected error for empty device ID")
	} else {
		var verr *ValidationError
		if !errors.As(err, &verr) {
			t.Fatalf("GetFirmwareDetails() error = %v, want *ValidationError", err)
		}
	}
}

func TestGetFirmwareDetailsMiss(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/devices/AA:BB:CC:DD:EE:03/firmware" {
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	if _, err := sc.GetFirmwareDetails(context.Background(), "AA:BB:CC:DD:EE:03"); err == nil {
		t.Fatal("GetFirmwareDetails() expected error for missing device")
	} else {
		if !errors.Is(err, ErrInvalidDevice) {
			t.Fatalf("GetFirmwareDetails() error = %v, want wrapped %v", err, ErrInvalidDevice)
		}
	}
}
