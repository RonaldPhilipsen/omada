package controller

import (
	"errors"
	"testing"
)

func TestLanNetwork(t *testing.T) {
	n := NewLanNetwork(loadFixture(t, "network-lan.json"))
	if n.ID() != "net-lan-1" || n.Name() != "Office LAN" || n.Purpose() != "lan" {
		t.Error("identity")
	}
	if n.VLANID() == nil || *n.VLANID() != 10 {
		t.Errorf("vlan: %v", n.VLANID())
	}
	if n.GatewaySubnet() == nil || *n.GatewaySubnet() != "192.168.10.1/24" {
		t.Errorf("subnet: %v", n.GatewaySubnet())
	}
	if n.DHCPEnabled() == nil || !*n.DHCPEnabled() {
		t.Errorf("dhcp: %v", n.DHCPEnabled())
	}

	sparse := NewLanNetwork(map[string]any{"id": "x", "name": "Plain", "purpose": "lan"})
	if sparse.VLANID() != nil || sparse.GatewaySubnet() != nil || sparse.DHCPEnabled() != nil {
		t.Error("sparse network must have nil optionals")
	}
}

func TestDhcpReservation(t *testing.T) {
	r := NewDhcpReservation(loadFixture(t, "dhcp-reservation.json"))
	if r.ID() != "res-1" || r.MAC() != "DE:AD:BE:EF:99:99" || r.IP() != "192.168.10.50" {
		t.Error("identity")
	}
	if r.Description() == nil || *r.Description() != "Printer" {
		t.Errorf("description: %v", r.Description())
	}
	if r.Status() == nil || !*r.Status() {
		t.Errorf("status: %v", r.Status())
	}
	if r.NetID() == nil || *r.NetID() != "net-lan-1" {
		t.Errorf("net id: %v", r.NetID())
	}
	if r.NetName() == nil || *r.NetName() != "Office LAN" {
		t.Errorf("net name: %v", r.NetName())
	}
	if r.ClientName() == nil || *r.ClientName() != "Print Server" {
		t.Errorf("client name: %v", r.ClientName())
	}
	if r.ExportToIpMacBinding() == nil || !*r.ExportToIpMacBinding() {
		t.Errorf("export: %v", r.ExportToIpMacBinding())
	}
}

func TestVpnPolicy(t *testing.T) {
	p := NewVpnPolicy(VpnSiteToSite, loadFixture(t, "vpn-policy.json"))
	if p.Category() != VpnSiteToSite || p.PolicyID() != "vpn-1" || p.Name() != "HQ IPSec" {
		t.Error("identity")
	}
	if !p.Enabled() {
		t.Error("must be enabled")
	}
	if p.VpnType() != VpnTypeIPSec || p.VpnTypeName() != "IPsec" {
		t.Errorf("type: %v %v", p.VpnType(), p.VpnTypeName())
	}
	if p.UniqueID() != "site_to_site_vpn-1" {
		t.Errorf("unique id: %q", p.UniqueID())
	}

	unknown := NewVpnPolicy(VpnServer, map[string]any{"id": "v2", "vpnType": 77})
	if unknown.Name() != "v2" {
		t.Errorf("name falls back to id: %q", unknown.Name())
	}
	if unknown.Enabled() {
		t.Error("missing status must be disabled")
	}
	if unknown.VpnType() != VpnTypeUnknown {
		t.Errorf("unknown type: %v", unknown.VpnType())
	}
}

func TestVpnCategories(t *testing.T) {
	if VpnServer.ListEndpoint() != "client-to-site-vpn-servers" {
		t.Error("server endpoint")
	}
	if VpnClient.ListEndpoint() != "client-to-site-vpn-clients" {
		t.Error("client endpoint")
	}
	if VpnSiteToSite.ListEndpoint() != "site-to-site-vpns" {
		t.Error("site-to-site endpoint")
	}
	if VpnSiteToSite.String() != "site_to_site" {
		t.Error("string form")
	}
}

func TestVpnTypeNameFallback(t *testing.T) {
	if VpnTypeL2TP.TypeName() != "l2tp" || VpnTypeSSL.TypeName() != "ssl" {
		t.Error("type name fallbacks")
	}
}

func TestVlanCreate(t *testing.T) {
	v, err := NewVlanCreate("  Staff  ", 20)
	if err != nil {
		t.Fatal(err)
	}
	payload := v.Payload()
	if payload["name"] != "Staff" || payload["vlans"] != "20" || payload["purpose"] != "vlan" || payload["application"] != 0 {
		t.Errorf("payload: %v", payload)
	}
	if payload["dhcpL2RelayEnable"] != false || payload["igmpSnoopEnable"] != false || payload["mldSnoopEnable"] != false {
		t.Error("default disables")
	}
	guard, ok := payload["dhcpGuard"].(map[string]any)
	if !ok || guard["enable"] != false || guard["dhcpSvr1"] != nil {
		t.Errorf("dhcp guard: %v", payload["dhcpGuard"])
	}
	guard6, ok := payload["dhcpv6Guard"].(map[string]any)
	if !ok || guard6["enable"] != false || guard6["dhcpv6Svr2"] != nil {
		t.Errorf("dhcpv6 guard: %v", payload["dhcpv6Guard"])
	}
}

func TestVlanCreateValidation(t *testing.T) {
	cases := map[string]struct {
		name string
		id   int
	}{
		"empty name":      {"", 5},
		"whitespace name": {"   ", 5},
		"long name":       {string(make([]byte, 129)), 5},
		"control char":    {"bad\nname", 5},
		"protected 1":     {"Default", 5},
		"protected 2":     {"management", 5},
		"protected 3":     {"Management(Default)", 5},
		"id zero":         {"ok", 0},
		"id one":          {"ok", 1},
		"id high":         {"ok", 4091},
	}
	for label, c := range cases {
		if _, err := NewVlanCreate(c.name, c.id); err == nil {
			t.Errorf("%s: expected validation error", label)
		} else {
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Errorf("%s: expected ValidationError, got %v", label, err)
			}
		}
	}
	if _, err := NewVlanCreate("edge", 4090); err != nil {
		t.Errorf("4090 must be valid: %v", err)
	}
}

func validVLANNetwork(raw map[string]any) LanNetwork {
	base := map[string]any{
		"id":          "net-vlan-20",
		"name":        "Staff",
		"purpose":     "vlan",
		"primary":     false,
		"application": 0,
		"vlan":        20,
	}
	for k, v := range raw {
		base[k] = v
	}
	return NewLanNetwork(base)
}

func TestVlanUpdate(t *testing.T) {
	name := "Staff Renamed"
	u, err := NewVlanUpdate(&name, nil)
	if err != nil {
		t.Fatal(err)
	}
	network := validVLANNetwork(map[string]any{
		"igmpSnoopEnable": true,
		"dhcpGuard":       map[string]any{"enable": true, "dhcpSvr1": "10.0.0.1", "dhcpSvr2": nil},
	})
	payload, err := u.Payload(network)
	if err != nil {
		t.Fatal(err)
	}
	if payload["name"] != "Staff Renamed" {
		t.Errorf("name: %v", payload["name"])
	}
	if payload["vlan"] != 20 {
		t.Errorf("vlan preserved: %v", payload["vlan"])
	}
	if payload["igmpSnoopEnable"] != true {
		t.Error("writable settings preserved")
	}
	if _, present := payload["id"]; present {
		t.Error("computed fields must be stripped")
	}
	if _, present := payload["primary"]; present {
		t.Error("computed fields must be stripped")
	}

	id := 21
	u2, err := NewVlanUpdate(nil, &id)
	if err != nil {
		t.Fatal(err)
	}
	payload2, err := u2.Payload(network)
	if err != nil {
		t.Fatal(err)
	}
	if payload2["vlan"] != 21 || payload2["name"] != "Staff" {
		t.Errorf("id-only update: %v", payload2)
	}

	if _, err := NewVlanUpdate(nil, nil); err == nil {
		t.Error("update without changes must fail")
	}
}

func TestValidateVlanTarget(t *testing.T) {
	if err := ValidateVlanTarget(validVLANNetwork(nil)); err != nil {
		t.Fatalf("valid target: %v", err)
	}
	if err := ValidateVlanTarget(validVLANNetwork(map[string]any{"primary": true})); err == nil {
		t.Error("primary network must be rejected")
	}
	if err := ValidateVlanTarget(validVLANNetwork(map[string]any{"purpose": "lan"})); err == nil {
		t.Error("non-vlan purpose must be rejected")
	}
	if err := ValidateVlanTarget(validVLANNetwork(map[string]any{"application": 1})); err == nil {
		t.Error("routed network must be rejected")
	}
	if err := ValidateVlanTarget(validVLANNetwork(map[string]any{"stray": true})); err == nil {
		t.Error("unknown fields must be rejected")
	}
	if err := ValidateVlanTarget(validVLANNetwork(map[string]any{"id": "bad id"})); err == nil {
		t.Error("unsafe id must be rejected")
	}
	if err := ValidateVlanTarget(validVLANNetwork(map[string]any{"igmpSnoopEnable": "yes"})); err == nil {
		t.Error("non-bool flag must be rejected")
	}
	if err := ValidateVlanTarget(validVLANNetwork(map[string]any{"dhcpGuard": map[string]any{"enable": "yes"}})); err == nil {
		t.Error("non-bool guard enable must be rejected")
	}
	if err := ValidateVlanTarget(validVLANNetwork(map[string]any{"dhcpGuard": map[string]any{"enable": false, "bogus": 1}})); err == nil {
		t.Error("unknown guard field must be rejected")
	}
	if err := ValidateVlanTarget(validVLANNetwork(map[string]any{"dhcpGuard": map[string]any{"enable": false, "dhcpSvr1": 5}})); err == nil {
		t.Error("non-string guard server must be rejected")
	}
}

func TestValidateVlanConflicts(t *testing.T) {
	staff := validVLANNetwork(nil)
	other := validVLANNetwork(map[string]any{"id": "net-other", "name": "Other", "vlan": 30})
	inventory := []LanNetwork{staff, other}

	if err := ValidateVlanConflicts(map[string]any{"vlan": 20, "name": "Fresh"}, inventory, nil); err == nil {
		t.Error("id collision must be rejected")
	}
	if err := ValidateVlanConflicts(map[string]any{"vlan": 40, "name": "STAFF"}, inventory, nil); err == nil {
		t.Error("case-insensitive name collision must be rejected")
	}
	exclude := "net-vlan-20"
	if err := ValidateVlanConflicts(map[string]any{"vlan": 20, "name": "Fresh"}, inventory, &exclude); err != nil {
		t.Errorf("excluded network must be skipped: %v", err)
	}
	if err := ValidateVlanConflicts(map[string]any{"vlans": "25", "name": "Fresh"}, inventory, nil); err != nil {
		t.Errorf("range-string identifier must be accepted: %v", err)
	}
	if err := ValidateVlanConflicts(map[string]any{"name": "Fresh"}, inventory, nil); err == nil {
		t.Error("payload without identifier must fail")
	}
	badInventory := []LanNetwork{NewLanNetwork(map[string]any{"name": "no id"})}
	if err := ValidateVlanConflicts(map[string]any{"vlan": 20, "name": "Fresh"}, badInventory, nil); err == nil {
		t.Error("inventory without ids must fail")
	}
}

func TestResolveVlanTarget(t *testing.T) {
	staff := validVLANNetwork(nil)
	other := validVLANNetwork(map[string]any{"id": "net-other", "name": "Other", "vlan": 30})
	inventory := []LanNetwork{staff, other}

	byID, err := ResolveVlanTarget("net-vlan-20", inventory, true)
	if err != nil || byID.ID() != "net-vlan-20" {
		t.Errorf("by id: %v %v", byID.ID(), err)
	}
	byName, err := ResolveVlanTarget("Other", inventory, false)
	if err != nil || byName.ID() != "net-other" {
		t.Errorf("by name: %v %v", byName.ID(), err)
	}
	if _, err := ResolveVlanTarget("Missing", inventory, false); err == nil {
		t.Error("missing target must fail")
	}
	dup := append([]LanNetwork{}, inventory...)
	dup = append(dup, validVLANNetwork(map[string]any{"id": "net-other", "name": "Other2", "vlan": 31}))
	if _, err := ResolveVlanTarget("net-other", dup, true); err == nil {
		t.Error("ambiguous identity must fail")
	}
}
