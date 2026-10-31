package controller

import (
	"context"
	"encoding/json"
	"errors"
	models "github.com/RonaldPhilipsen/omada/controller/models"
	"net/http"
	"testing"
)

func TestAccessPoints(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/accessPoints" {
			if got := r.URL.Query().Get("deviceType"); got != "accessPoint" {
				t.Errorf("deviceType = %q, want accessPoint", got)
			}
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data": []map[string]any{
					{"mac": "AA:BB:CC:DD:EE:02", "name": "Office AP", "type": "ap", "status": 14, "statusCategory": 1, "wirelessLinked": true},
				},
				"currentSize": 1,
				"totalRows":   1,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	aps, err := sc.AccessPoints(context.Background())
	if err != nil {
		t.Fatalf("AccessPoints() error = %v", err)
	}
	if len(aps) != 1 {
		t.Fatalf("AccessPoints() returned %d access points, want 1", len(aps))
	}
	if aps[0].MAC() != "AA:BB:CC:DD:EE:02" || aps[0].Name() != "Office AP" {
		t.Errorf("AccessPoints()[0] = %+v, want Office AP", aps[0])
	}
}

func TestAccessPointEmptyID(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	if _, err := sc.AccessPoint(context.Background(), ""); err == nil {
		t.Fatal("models.AccessPoint() expected error for empty device ID")
	} else {
		var verr *models.ValidationError
		if !errors.As(err, &verr) {
			t.Fatalf("models.AccessPoint() error = %v, want *models.ValidationError", err)
		}
	}
}

func TestAccessPointsByMAC(t *testing.T) {
	var gotMac string
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/openapi/v2/"+testControllerID+"/sites/site-1/accessPoints" {
			gotMac = r.URL.Query().Get("mac")
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data": []map[string]any{
					{"mac": "AA:BB:CC:DD:EE:02", "name": "Office AP", "type": "ap"},
				},
				"currentSize": 1,
				"totalRows":   1,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	aps, err := sc.AccessPointsByMAC(context.Background(), []string{"AA:BB:CC:DD:EE:02", "AA:BB:CC:DD:EE:04"})
	if err != nil {
		t.Fatalf("AccessPointsByMAC() error = %v", err)
	}
	if gotMac != "AA:BB:CC:DD:EE:02,AA:BB:CC:DD:EE:04" {
		t.Errorf("AccessPointsByMAC mac = %q, want comma-joined MAC list", gotMac)
	}
	if len(aps) != 1 {
		t.Fatalf("AccessPointsByMAC() returned %d access points, want 1", len(aps))
	}
	if _, err := sc.AccessPointsByMAC(context.Background(), nil); err == nil {
		t.Fatal("AccessPointsByMAC() expected error for empty MAC list")
	}
}

func TestApPorts(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/openapi/v2/"+testControllerID+"/sites/site-1/accessPoints/AA:BB:CC:DD:EE:02/portConfig" {
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data": []map[string]any{
					{"lanPort": "LAN1", "supportVlan": true, "localVlanEnable": true, "localVlanId": 10, "supportPoe": false, "poeOutEnable": false},
				},
				"currentSize": 1,
				"totalRows":   1,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	ports, err := sc.ApPorts(context.Background(), "AA:BB:CC:DD:EE:02")
	if err != nil {
		t.Fatalf("ApPorts() error = %v", err)
	}
	if len(ports) != 1 {
		t.Fatalf("ApPorts() returned %d ports, want 1", len(ports))
	}
	if ports[0].PortName() != "LAN1" {
		t.Errorf("ApPorts()[0].PortName() = %q, want LAN1", ports[0].PortName())
	}
}

func TestSetApPortEnabled(t *testing.T) {
	var body map[string]any
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch && r.URL.Path == "/openapi/v2/"+testControllerID+"/sites/site-1/accessPoints/AA:BB:CC:DD:EE:02/portConfig" {
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
	if err := sc.SetApPortEnabled(context.Background(), "AA:BB:CC:DD:EE:02", 1, true, false); err != nil {
		t.Fatalf("SetApPortEnabled() error = %v", err)
	}
	if body["index"] != float64(1) {
		t.Errorf("SetApPortEnabled body index = %v, want 1", body["index"])
	}
	if body["enable"] != true {
		t.Errorf("SetApPortEnabled body enable = %v, want true", body["enable"])
	}
	if body["poeEnable"] != false {
		t.Errorf("SetApPortEnabled body poeEnable = %v, want false", body["poeEnable"])
	}
}

func TestApRadioSettingsMiss(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/accessPoints/AA:BB:CC:DD:EE:02" {
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"type": "ap", "mac": "AA:BB:CC:DD:EE:02", "status": 14, "statusCategory": 1,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	_, err := sc.ApRadioSettings(context.Background(), "AA:BB:CC:DD:EE:02", 99)
	if err == nil {
		t.Fatal("ApRadioSettings() expected error for unknown radio")
	}
	var verr *models.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("ApRadioSettings() error = %v, want *models.ValidationError", err)
	}
}

func TestSetApRadioEnabled(t *testing.T) {
	var gotBody map[string]any
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/accessPoints/AA:BB:CC:DD:EE:02":
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"type": "ap", "mac": "AA:BB:CC:DD:EE:02", "status": 14, "statusCategory": 1,
				"radioSetting2g": map[string]any{
					"radioEnable": true, "channelWidth": float64(models.ChannelWidthWidth80), "channel": float64(1),
					"freq": float64(2412), "wirelessMode": float64(-2), "txPower": float64(20), "txPowerLevel": float64(2),
				},
			}})
		case r.Method == http.MethodPatch && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/accessPoints/AA:BB:CC:DD:EE:02/radioConfig":
			if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
				t.Errorf("decode body: %v", err)
				return
			}
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"success": true}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	if err := sc.SetApRadioEnabled(context.Background(), "AA:BB:CC:DD:EE:02", 0, false); err != nil {
		t.Fatalf("SetApRadioEnabled() error = %v", err)
	}
	radioEnable, ok := gotBody["radioEnable"].(bool)
	if !ok || radioEnable {
		t.Errorf("SetApRadioEnabled body radioEnable = %v, want false", gotBody["radioEnable"])
	}
}

func TestSetApRadioChannelMerges(t *testing.T) {
	var gotBody map[string]any
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/accessPoints/AA:BB:CC:DD:EE:02":
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"type": "ap", "mac": "AA:BB:CC:DD:EE:02", "status": 14, "statusCategory": 1,
				"radioSetting2g": map[string]any{
					"radioEnable": true, "channelWidth": float64(models.ChannelWidthWidth80), "channel": float64(1),
					"freq": float64(2412), "wirelessMode": float64(-2), "txPower": float64(20), "txPowerLevel": float64(2),
				},
			}})
		case r.Method == http.MethodPatch && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/accessPoints/AA:BB:CC:DD:EE:02/radioConfig":
			if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
				t.Errorf("decode body: %v", err)
				return
			}
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"success": true}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	if err := sc.SetApRadioChannel(context.Background(), "AA:BB:CC:DD:EE:02", 0, 6, models.ChannelWidthWidth40); err != nil {
		t.Fatalf("SetApRadioChannel() error = %v", err)
	}
	if gotBody == nil {
		t.Fatal("SetApRadioChannel() did not call the radio config endpoint")
	}
	if gotBody["channel"] != float64(6) {
		t.Errorf("SetApRadioChannel body channel = %v, want 6", gotBody["channel"])
	}
	if gotBody["channelWidth"] != float64(int(models.ChannelWidthWidth40)) {
		t.Errorf("SetApRadioChannel body channelWidth = %v, want %v", gotBody["channelWidth"], int(models.ChannelWidthWidth40))
	}
	if gotBody["freq"] != float64(2412) {
		t.Errorf("SetApRadioChannel body freq = %v, want 2412 (original preserved)", gotBody["freq"])
	}
}

func TestApAvailableChannels(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/openapi/v2/"+testControllerID+"/sites/site-1/accessPoints/AA:BB:CC:DD:EE:02/radioConfig/availableChannels" {
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data": []map[string]any{
					{"channel": 1},
					{"channel": 6},
				},
				"currentSize": 2,
				"totalRows":   2,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	channels, err := sc.ApAvailableChannels(context.Background(), "AA:BB:CC:DD:EE:02", 0)
	if err != nil {
		t.Fatalf("ApAvailableChannels() error = %v", err)
	}
	if len(channels) != 2 {
		t.Fatalf("ApAvailableChannels() returned %d channels, want 2", len(channels))
	}
	if channels[0] != 1 || channels[1] != 6 {
		t.Errorf("ApAvailableChannels() = %v, want [1 6]", channels)
	}
}
