package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestSiteVersion(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1" {
			writeJSON(t, w, map[string]any{"errorCode": 0, "controllerVer": "6.2.0.0", "omadacId": testControllerID})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	version, err := sc.SiteVersion(context.Background())
	if err != nil {
		t.Fatalf("SiteVersion() error = %v", err)
	}
	if version.Compare(ParseVersionMust("6.2.0.0")) != 0 {
		t.Errorf("SiteVersion() = %v, want 6.2.0.0", version)
	}
}

func TestClientsAllPages(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/clients" {
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data":        []map[string]any{{"mac": "AA:BB:CC:DD:EE:01", "name": "Laptop", "wireless": true, "connectType": 2}},
				"currentSize": 1,
				"totalRows":   1,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	clients, err := sc.Clients(context.Background())
	if err != nil {
		t.Fatalf("Clients() error = %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("Clients() returned %d clients, want 1", len(clients))
	}
	if clients[0].MAC() != "AA:BB:CC:DD:EE:01" {
		t.Errorf("Clients()[0].MAC() = %q, want %q", clients[0].MAC(), "AA:BB:CC:DD:EE:01")
	}
}

func TestClientByMAC(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/clients/AA:BB:CC:DD:EE:01":
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"mac": "AA:BB:CC:DD:EE:01", "name": "Laptop", "wireless": true}})
		case r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/clients/unknown":
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	client, err := sc.Client(context.Background(), "AA:BB:CC:DD:EE:01")
	if err != nil {
		t.Fatalf("Client() error = %v", err)
	}
	if client.MAC() != "AA:BB:CC:DD:EE:01" {
		t.Errorf("Client().MAC() = %q, want %q", client.MAC(), "AA:BB:CC:DD:EE:01")
	}
	if _, err := sc.Client(context.Background(), "unknown"); err == nil {
		t.Fatal("Client() expected error for unknown client")
	}
	if _, err := sc.Client(context.Background(), ""); err == nil {
		t.Fatal("Client() expected error for empty MAC")
	}
}

func TestBlockUnblockClient(t *testing.T) {
	var blocked, unblocked []string
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/cmd/clients/AA:BB:CC:DD:EE:01/block":
			blocked = append(blocked, r.URL.Path)
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"success": true}})
		case r.Method == http.MethodPost && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/cmd/clients/AA:BB:CC:DD:EE:01/unblock":
			unblocked = append(unblocked, r.URL.Path)
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"success": true}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	if err := sc.BlockClient(context.Background(), "AA:BB:CC:DD:EE:01"); err != nil {
		t.Fatalf("BlockClient() error = %v", err)
	}
	if err := sc.UnblockClient(context.Background(), "AA:BB:CC:DD:EE:01"); err != nil {
		t.Fatalf("UnblockClient() error = %v", err)
	}
	if len(blocked) != 1 || len(unblocked) != 1 {
		t.Errorf("blocked = %v, unblocked = %v", blocked, unblocked)
	}
}

func TestReconnectClient(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/cmd/clients/AA:BB:CC:DD:EE:01/reconnect" {
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"success": true}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	if err := sc.ReconnectClient(context.Background(), "AA:BB:CC:DD:EE:01"); err != nil {
		t.Fatalf("ReconnectClient() error = %v", err)
	}
}

func TestUpdateClientSettings(t *testing.T) {
	var patchBodies []map[string]any
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/clients/AA:BB:CC:DD:EE:01" {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode patch body: %v", err)
				return
			}
			patchBodies = append(patchBodies, body)
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"mac": "AA:BB:CC:DD:EE:01", "name": "New"}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}

	if err := sc.UpdateClientSettings(context.Background(), "AA:BB:CC:DD:EE:01", "New Name", nil, nil); err != nil {
		t.Fatalf("UpdateClientSettings() error = %v", err)
	}
	if len(patchBodies) != 1 {
		t.Fatalf("patch calls = %d, want 1", len(patchBodies))
	}
	if patchBodies[0]["clientName"] != "New Name" {
		t.Errorf("patch body clientName = %v, want New Name", patchBodies[0]["clientName"])
	}

	ipSetting := &IpSetting{data: map[string]any{"useFixedAddr": true, "netId": "net-1", "ip": "192.168.10.50"}}
	if err := sc.UpdateClientSettings(context.Background(), "AA:BB:CC:DD:EE:01", "", ipSetting, nil); err != nil {
		t.Fatalf("UpdateClientSettings() error = %v", err)
	}
	if len(patchBodies) != 2 {
		t.Fatalf("patch calls = %d, want 2", len(patchBodies))
	}
	ipSettingPayload, ok := patchBodies[1]["ipSetting"].(map[string]any)
	if !ok {
		t.Fatalf("patch body ipSetting missing: %v", patchBodies[1])
	}
	if ipSettingPayload["useFixedAddr"] != true {
		t.Errorf("ipSetting useFixedAddr = %v, want true", ipSettingPayload["useFixedAddr"])
	}
	if ipSettingPayload["netId"] != "net-1" {
		t.Errorf("ipSetting netId = %v, want net-1", ipSettingPayload["netId"])
	}

	if err := sc.UpdateClientSettings(context.Background(), "AA:BB:CC:DD:EE:01", "", nil, nil); err == nil {
		t.Fatal("UpdateClientSettings() expected error when all settings are nil")
	}
}

func TestConnectedClientsVersionGate(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1":
			writeJSON(t, w, map[string]any{"errorCode": 0, "controllerVer": "6.2.0.0", "omadacId": testControllerID})
		case r.Method == http.MethodPost && r.URL.Path == "/openapi/v2/"+testControllerID+"/sites/site-1/clients":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode openapi body: %v", err)
				return
			}
			if filters, ok := body["filters"].(map[string]any); !ok || filters["active"] != true {
				t.Errorf("openapi filters = %v, want active=true", body["filters"])
			}
			if scope, ok := body["scope"].(float64); !ok || scope != 1 {
				t.Errorf("openapi scope = %v, want 1", body["scope"])
			}
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data":        []map[string]any{{"mac": "AA:BB:CC:DD:EE:01", "name": "Laptop", "active": true}},
				"currentSize": 1,
				"totalRows":   1,
			}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	clients, err := sc.ConnectedClients(context.Background())
	if err != nil {
		t.Fatalf("ConnectedClients() error = %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("ConnectedClients() returned %d clients, want 1", len(clients))
	}
}

func TestConnectedClientsLegacyFilter(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1":
			writeJSON(t, w, map[string]any{"errorCode": 0, "controllerVer": "6.1.0.5", "omadacId": testControllerID})
		case r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/clients":
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data": []map[string]any{
					{"mac": "AA:BB:CC:DD:EE:01", "name": "Laptop", "active": true},
					{"mac": "AA:BB:CC:DD:EE:02", "name": "Phone", "active": false},
				},
				"currentSize": 2,
				"totalRows":   2,
			}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	clients, err := sc.ConnectedClients(context.Background())
	if err != nil {
		t.Fatalf("ConnectedClients() error = %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("ConnectedClients() returned %d clients, want 1 (only the active one)", len(clients))
	}
	if clients[0].MAC() != "AA:BB:CC:DD:EE:01" {
		t.Errorf("ConnectedClients()[0].MAC() = %q, want %q", clients[0].MAC(), "AA:BB:CC:DD:EE:01")
	}
}

func TestKnownClients(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1/insight/clients" {
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data":        []map[string]any{{"mac": "AA:BB:CC:DD:EE:09", "name": "Guest"}},
				"currentSize": 1,
				"totalRows":   1,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	clients, err := sc.KnownClients(context.Background())
	if err != nil {
		t.Fatalf("KnownClients() error = %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("KnownClients() returned %d clients, want 1", len(clients))
	}
	if clients[0].MAC() != "AA:BB:CC:DD:EE:09" {
		t.Errorf("KnownClients()[0].MAC() = %q, want %q", clients[0].MAC(), "AA:BB:CC:DD:EE:09")
	}
}

func TestNetworks(t *testing.T) {
	var bodies []map[string]any
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/openapi/v2/"+testControllerID+"/sites/site-1/lan-networks" {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode body: %v", err)
				return
			}
			bodies = append(bodies, body)
			if body["type"] != "lan" {
				t.Errorf("body type = %v, want lan", body["type"])
			}
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data":        []map[string]any{{"id": "net-1", "name": "Guest", "purpose": "vlan", "vlan": 10, "gatewaySubnet": "192.168.10.1/24"}},
				"currentSize": 1,
				"totalRows":   1,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	networks, err := sc.Networks(context.Background())
	if err != nil {
		t.Fatalf("Networks() error = %v", err)
	}
	if len(networks) != 1 {
		t.Fatalf("Networks() returned %d networks, want 1", len(networks))
	}
	if networks[0].ID() != "net-1" || networks[0].Name() != "Guest" {
		t.Errorf("Networks()[0] = %+v, want net-1/Guest", networks[0])
	}
	if networks[0].VLANID() == nil || *networks[0].VLANID() != 10 {
		t.Errorf("Networks()[0].VLANID() = %v, want 10", networks[0].VLANID())
	}
	if len(bodies) != 1 {
		t.Errorf("calls = %d, want 1", len(bodies))
	}
}

func TestVLANs(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/openapi/v2/"+testControllerID+"/sites/site-1/lan-networks" {
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data": []map[string]any{
					{"id": "net-1", "name": "Guest", "purpose": "vlan", "vlan": 10},
					{"id": "net-2", "name": "LAN", "purpose": "lan"},
				},
				"currentSize": 2,
				"totalRows":   2,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	vlans, err := sc.VLANs(context.Background())
	if err != nil {
		t.Fatalf("VLANs() error = %v", err)
	}
	if len(vlans) != 1 || vlans[0].ID() != "net-1" {
		t.Errorf("VLANs() = %+v, want net-1 only", vlans)
	}
}

func TestCreateVLAN(t *testing.T) {
	var created map[string]any
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/openapi/v2/"+testControllerID+"/sites/site-1/lan-networks" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
			return
		}
		if body["purpose"] == "vlan" {
			created = body
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"id": "net-1", "name": "Guest", "purpose": "vlan", "vlan": 10,
			}})
			return
		}
		writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
			"data":        []map[string]any{{"id": "net-1", "name": "LAN", "purpose": "lan", "vlan": 1}},
			"currentSize": 1,
			"totalRows":   1,
		}})
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}

	n, err := sc.CreateVLAN(context.Background(), "Guest", 10)
	if err != nil {
		t.Fatalf("CreateVLAN() error = %v", err)
	}
	if n.ID() != "net-1" || n.Name() != "Guest" || n.Purpose() != "vlan" {
		t.Errorf("CreateVLAN() = %+v, want net-1/Guest/vlan", n)
	}
	if n.VLANID() == nil || *n.VLANID() != 10 {
		t.Errorf("CreateVLAN().VLANID() = %v, want 10", n.VLANID())
	}
	if created == nil {
		t.Fatal("CreateVLAN() did not call the create endpoint")
	}
	if created["application"] != float64(0) || created["purpose"] != "vlan" || created["vlans"] != "10" {
		t.Errorf("create payload = %v, want VLAN contract payload", created)
	}

	if _, err := sc.CreateVLAN(context.Background(), "", 10); err == nil {
		t.Fatal("CreateVLAN() expected error for empty name")
	}
	if _, err := sc.CreateVLAN(context.Background(), "Guest", 1); err == nil {
		t.Fatal("CreateVLAN() expected error for protected VLAN ID")
	}
}

func TestCreateVLANConflict(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/openapi/v2/"+testControllerID+"/sites/site-1/lan-networks" {
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data":        []map[string]any{{"id": "net-1", "name": "Guest", "purpose": "vlan", "vlan": 10}},
				"currentSize": 1,
				"totalRows":   1,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}
	if _, err := sc.CreateVLAN(context.Background(), "Other", 10); err == nil {
		t.Fatal("CreateVLAN() expected conflict error for existing VLAN ID")
	}
}

func TestUpdateVLAN(t *testing.T) {
	var patched map[string]any
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch && r.URL.Path == "/openapi/v2/"+testControllerID+"/sites/site-1/lan-networks/net-1" {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode body: %v", err)
				return
			}
			patched = body
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"id": "net-1", "name": "Renamed", "purpose": "vlan", "vlan": 20,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := &SiteClient{c: cl.c, site: "site-1"}

	name := "Renamed"
	vlanID := 20
	update, err := NewVlanUpdate(&name, &vlanID)
	if err != nil {
		t.Fatalf("NewVlanUpdate() error = %v", err)
	}
	network := LanNetwork{data: map[string]any{
		"id": "net-1", "name": "Guest", "purpose": "vlan", "vlan": 10,
		"primary":           false,
		"application":       0,
		"dhcpL2RelayEnable": false,
		"dhcpGuard":         map[string]any{"enable": false, "dhcpSvr1": nil, "dhcpSvr2": nil},
		"dhcpv6Guard":       map[string]any{"enable": false, "dhcpv6Svr1": nil, "dhcpv6Svr2": nil},
	}}
	n, err := sc.UpdateVLAN(context.Background(), network, update)
	if err != nil {
		t.Fatalf("UpdateVLAN() error = %v", err)
	}
	if n.ID() != "net-1" || n.Name() != "Renamed" || n.Purpose() != "vlan" {
		t.Errorf("UpdateVLAN() = %+v, want net-1/Renamed/vlan", n)
	}
	if n.VLANID() == nil || *n.VLANID() != 20 {
		t.Errorf("UpdateVLAN().VLANID() = %v, want 20", n.VLANID())
	}
	if patched == nil {
		t.Fatal("UpdateVLAN() did not call the update endpoint")
	}
	if patched["name"] != "Renamed" || patched["vlan"] != float64(20) {
		t.Errorf("patch payload = %v, want name Renamed and vlan 20", patched)
	}

	if _, err := sc.UpdateVLAN(context.Background(), LanNetwork{}, update); err == nil {
		t.Fatal("UpdateVLAN() expected error for empty network ID")
	}
	name2 := "X"
	upd, err := NewVlanUpdate(&name2, nil)
	if err != nil {
		t.Fatalf("NewVlanUpdate() error = %v", err)
	}
	lan := LanNetwork{data: map[string]any{"id": "net-2", "name": "LAN", "purpose": "lan", "vlan": 1}}
	if _, err := sc.UpdateVLAN(context.Background(), lan, upd); err == nil {
		t.Fatal("UpdateVLAN() expected error for non-VLAN network")
	}
}
