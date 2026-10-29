package controller

import "testing"

func TestDeviceBase(t *testing.T) {
	d := NewDevice(loadFixture(t, "device-gateway.json"))
	if d.Type() != "gateway" {
		t.Errorf("type: %q", d.Type())
	}
	if d.ResourcePath() != "gateways/AA:BB:CC:DD:EE:01" {
		t.Errorf("resource path: %q", d.ResourcePath())
	}
	if d.MAC() != "AA:BB:CC:DD:EE:01" {
		t.Errorf("mac: %q", d.MAC())
	}
	if d.Name() != "Main Gateway" {
		t.Errorf("name: %q", d.Name())
	}
	if d.Model() != "EAC2401" || d.ModelDisplayName() != "Omada EAC2401" {
		t.Errorf("model: %q / %q", d.Model(), d.ModelDisplayName())
	}
	if d.Status() != DeviceStatusConnected || d.StatusCategory() != DeviceStatusCategoryConnected {
		t.Errorf("status: %v / %v", d.Status(), d.StatusCategory())
	}
	if d.IPAddress() != "192.168.1.1" {
		t.Errorf("ip: %q", d.IPAddress())
	}
	if d.DisplayUptime() == nil || *d.DisplayUptime() != "1 day 2 hours" {
		t.Errorf("display uptime: %v", d.DisplayUptime())
	}
	if d.CPUUsage() != 12 || d.MemUsage() != 34 {
		t.Errorf("cpu/mem: %d/%d", d.CPUUsage(), d.MemUsage())
	}
	if d.Uptime() != 100000 {
		t.Errorf("uptime: %d", d.Uptime())
	}
	if d.FirmwareVersion() != "1.8.10" {
		t.Errorf("firmware: %q", d.FirmwareVersion())
	}
	if !d.NeedUpgrade() || d.FWDownload() {
		t.Error("needUpgrade/fwDownload flags")
	}
	if d.LedSetting() != LedSettingSiteSettings {
		t.Errorf("led: %v", d.LedSetting())
	}
}

func TestDeviceDisconnectedDefaults(t *testing.T) {
	d := NewDevice(map[string]any{"type": "ap", "mac": "AA:00:00:00:00:00", "statusCategory": 0})
	if d.DisplayUptime() != nil || d.CPUUsage() != 0 || d.MemUsage() != 0 || d.Uptime() != 0 {
		t.Error("disconnected device must zero out live stats")
	}
	if d.FirmwareVersion() != "Unknown" {
		t.Errorf("firmware: %q", d.FirmwareVersion())
	}
	if d.NeedUpgrade() || d.FWDownload() {
		t.Error("disconnected device cannot be upgrading")
	}
	if d.Name() != "AA:00:00:00:00:00" {
		t.Errorf("name falls back to mac: %q", d.Name())
	}
	if d.Model() != "Unknown" || d.ModelDisplayName() != "Unknown Model" {
		t.Error("model fallbacks")
	}
	// 0 is a real value: DISCONNECTED.
	if d.Status() != DeviceStatusUnknown || d.StatusCategory() != DeviceStatusCategoryDisconnected || d.LedSetting() != LedSettingUnknown {
		t.Error("enum fallbacks")
	}
	bare := NewDevice(map[string]any{"type": "ap", "mac": "AA:00:00:00:00:00"})
	if bare.StatusCategory() != DeviceStatusCategoryUnknown {
		t.Error("absent statusCategory must be unknown")
	}
}

func TestDeviceSparseName(t *testing.T) {
	d := NewDevice(loadFixture(t, "device-ap-sparse.json"))
	if d.Name() != "AA:BB:CC:DD:EE:02" {
		t.Errorf("ap name falls back to mac: %q", d.Name())
	}
}

func TestSwitch(t *testing.T) {
	s := NewSwitch(loadFixture(t, "device-switch.json"))
	if s.NumberPorts() != 8 {
		t.Errorf("ports fallback to deviceMisc: %d", s.NumberPorts())
	}
	ports := s.Ports()
	if len(ports) != 2 {
		t.Fatalf("ports: %d", len(ports))
	}
	p1 := ports[0]
	if p1.Port() != 1 || p1.Name() != "Port1" || p1.ProfileID() != "prof-1" {
		t.Errorf("port1: %v %v %v", p1.Port(), p1.Name(), p1.ProfileID())
	}
	if p1.Type() != PortTypeCopper || p1.Operation() != "switching" || p1.IsDisabled() {
		t.Error("port1 config")
	}
	ps := p1.PortStatus()
	if ps.LinkStatus() != LinkStatusLinkUp || ps.LinkSpeed() != LinkSpeedSpeed1G || !ps.PoEActive() {
		t.Errorf("port status: %v %v %v", ps.LinkStatus(), ps.LinkSpeed(), ps.PoEActive())
	}
	if ps.PoEPower() == nil || *ps.PoEPower() != 6.5 {
		t.Errorf("poe power: %v", ps.PoEPower())
	}
	if ps.BytesTX() != 111 || ps.BytesRX() != 222 || ps.STPDiscarding() {
		t.Error("port counters")
	}
	p8 := ports[1]
	if p8.PortStatus().PoEPower() != nil || p8.PortStatus().PoEActive() {
		t.Error("port8 must not be PoE active")
	}

	caps := s.DeviceCapabilities()
	if caps.PoEPorts() != 4 || !caps.SupportsPoE() || caps.SupportsBT() {
		t.Error("capabilities")
	}

	up := s.Uplink()
	if up == nil || up.MAC() != "AA:BB:CC:DD:EE:01" || up.Name() != "Main Gateway" || up.Type() != "gateway" || up.Port() != 1 {
		t.Errorf("uplink: %+v", up)
	}
	down := s.Downlinks()
	if len(down) != 1 || down[0].Type() != "ap" || down[0].Model() != "PS-1" {
		t.Errorf("downlinks: %+v", down)
	}

	noUplink := NewSwitch(map[string]any{"type": "switch", "mac": "x", "uplink": nil})
	if noUplink.Uplink() != nil {
		t.Error("null uplink must be nil")
	}
	if len(noUplink.Downlinks()) != 0 {
		t.Error("missing downlinkList must be empty")
	}
}

func TestAccessPoint(t *testing.T) {
	ap := NewAccessPoint(loadFixture(t, "device-ap-sparse.json"))
	if ap.WirelessLinked() {
		t.Error("not wireless linked")
	}
	if !ap.Supports5G() || !ap.Supports6G() || !ap.Supports11AC() || !ap.SupportsMesh() || ap.Supports5G2() {
		t.Error("band support flags")
	}
	lan := ap.LANPortSettings()
	if len(lan) != 1 || lan[0].PortName() != "LAN1" || !lan[0].LocalVLANEnable() || lan[0].LocalVLANID() != 10 || lan[0].PoEEnable() {
		t.Errorf("lan ports: %+v", lan)
	}
	up := ap.WiredUplink()
	if up == nil || up.MAC() != "AA:BB:CC:DD:EE:01" || up.Port() != 3 {
		t.Errorf("wired uplink: %+v", up)
	}

	bands := ap.RadioBands()
	if len(bands) != 2 || bands[0] != RadioIdFreq24 || bands[1] != RadioIdFreq51 {
		t.Errorf("bands: %v", bands)
	}

	s2g := ap.RadioSettings(RadioIdFreq24)
	if s2g == nil {
		t.Fatal("2.4g settings missing")
	}
	if !s2g.RadioEnabled() || s2g.ChannelIndex() != 0 || !s2g.AutoChannel() || s2g.Frequency() != 0 {
		t.Error("2.4g auto channel")
	}
	if s2g.ChannelWidth() != ChannelWidthWidth80 || s2g.WirelessMode() != -2 {
		t.Errorf("2.4g width/mode: %v %v", s2g.ChannelWidth(), s2g.WirelessMode())
	}
	s5g := ap.RadioSettings(RadioIdFreq51)
	if s5g == nil || s5g.ChannelIndex() != 3 || s5g.AutoChannel() || s5g.Frequency() != 5180 {
		t.Errorf("5g settings: %+v", s5g)
	}
	if s5g.TxPower() != 23 || s5g.TxPowerLevel() != 3 {
		t.Error("5g power")
	}
	if ap.RadioSettings(RadioIdFreq6) != nil {
		t.Error("6g radio must be absent")
	}

	st2g := ap.RadioStatus(RadioIdFreq24)
	if st2g == nil || st2g.ActualChannel() != "1 / 2412MHz" || st2g.WirelessMode() != "b/g/n mixed" {
		t.Errorf("2.4g status: %+v", st2g)
	}
	if st2g.RXUtil() != 10 || st2g.TXUtil() != 20 || st2g.InterUtil() != 1 || st2g.MaxTxRate() != 72 {
		t.Error("2.4g utilization")
	}
	if ap.RadioStatus(RadioIdFreq52) != nil {
		t.Error("5g2 status must be absent")
	}

	bare := NewAccessPoint(map[string]any{"type": "ap", "mac": "x"})
	if bare.Supports5G() || bare.WiredUplink() != nil || len(bare.LANPortSettings()) != 0 {
		t.Error("bare ap must report no features")
	}
	if len(bare.RadioBands()) != 0 {
		t.Error("bare ap has no radios")
	}
}

func TestGateway(t *testing.T) {
	g := NewGateway(loadFixture(t, "device-gateway.json"))
	if g.NumberPorts() != 4 || !g.SupportsPoE() || !g.IsCombinedGateway() || !g.LLDPEnabled() {
		t.Error("gateway basics")
	}
	if g.IP() != "192.168.1.1" || g.EchoServer() == nil || *g.EchoServer() != "1.1.1.1" {
		t.Error("gateway ip/echo")
	}

	stats := g.PortStatuses()
	if len(stats) != 2 {
		t.Fatalf("port stats: %d", len(stats))
	}
	wan := stats[0]
	if wan.PortNumber() != 1 || wan.Name() != "Ethernet1" || wan.DisplayName() != "WAN" {
		t.Errorf("wan port: %+v", wan)
	}
	if wan.Type() != GatewayPortTypeWAN || wan.Mode() != GatewayPortModeWAN {
		t.Error("wan type/mode")
	}
	if wan.LinkStatus() != LinkStatusLinkUp || wan.LinkSpeed() != LinkSpeedSpeed1G || wan.LinkDuplex() != LinkDuplexFull {
		t.Errorf("wan link: %v %v %v", wan.LinkStatus(), wan.LinkSpeed(), wan.LinkDuplex())
	}
	if !wan.WANConnected() || !wan.IPv6WANConnected() || !wan.OnlineDetection() {
		t.Error("wan connectivity")
	}
	if wan.WANIPAddress() == nil || *wan.WANIPAddress() != "203.0.113.5" {
		t.Errorf("wan ip: %v", wan.WANIPAddress())
	}
	if !wan.WANIPv6Enabled() || wan.WANIPv6Address() == nil || *wan.WANIPv6Address() != "2001:db8::1" {
		t.Errorf("wan ipv6: %v %v", wan.WANIPv6Enabled(), wan.WANIPv6Address())
	}
	if wan.WANProtocol() == nil || *wan.WANProtocol() != "dhcp" {
		t.Errorf("wan proto: %v", wan.WANProtocol())
	}
	if wan.PoEActive() {
		t.Error("wan port is not PoE")
	}
	lan := stats[1]
	if !lan.PoEActive() || lan.WANConnected() {
		t.Error("lan port flags")
	}
	if lan.WANIPv6Enabled() || lan.WANIPv6Address() != nil {
		t.Error("lan port has no ipv6 wan config")
	}

	configs := g.PortConfigs()
	if len(configs) != 2 {
		t.Fatalf("port configs: %d", len(configs))
	}
	c1 := configs[0]
	// port 1 is absent from poeSettings, so its mode is NONE.
	if c1.PortNumber() != 1 || c1.PoEMode() != PoEModeNone {
		t.Errorf("config1 poe: %v", c1.PoEMode())
	}
	if c1.Duplex() != LinkDuplexFull || c1.LinkSpeed() != LinkSpeedSpeed1G || c1.MirrorEnable() {
		t.Error("config1 settings")
	}
	if c1.PortStatus().PortNumber() != 1 || !c1.PortStatus().WANConnected() {
		t.Error("config1 embedded status")
	}
	c3 := configs[1]
	if c3.PoEMode() != PoEModeDisabled {
		t.Errorf("config3 poe: %v (expect disabled from poeSettings)", c3.PoEMode())
	}
	if c3.Duplex() != LinkDuplexAuto || c3.LinkSpeed() != LinkSpeedSpeedAuto {
		t.Error("config3 defaults")
	}

	noPoe := NewGateway(map[string]any{"type": "gateway", "mac": "x", "portConfigs": []any{
		map[string]any{"port": 1, "portStat": map[string]any{"port": 1, "name": "p1", "type": 0, "mode": 0, "status": 2, "tx": 0, "rx": 0}},
	}})
	cfg := noPoe.PortConfigs()[0]
	if cfg.PoEMode() != PoEModeNone {
		t.Errorf("non-poe gateway config: %v", cfg.PoEMode())
	}
}

func TestPortDetails(t *testing.T) {
	m := map[string]any{
		"id":                            "port-id-1",
		"port":                          2,
		"name":                          "Port2",
		"profileId":                     "p1",
		"profileName":                   "UPlink",
		"profileOverrideEnable":         true,
		"type":                          1,
		"operation":                     "switching",
		"disable":                       false,
		"maxSpeed":                      5,
		"linkSpeed":                     3,
		"duplex":                        2,
		"supportPoe":                    true,
		"poe":                           2,
		"bandWidthCtrlType":             2,
		"dot1x":                         2,
		"lldpMedEnable":                 true,
		"topoNotifyEnable":              true,
		"spanningTreeEnable":            true,
		"loopbackDetectEnable":          true,
		"portIsolationEnable":           true,
		"loopbackDetectVlanBasedEnable": false,
		"nativeNetworkId":               "net-native",
		"tagIds":                        []any{"net-a", "net-b"},
		"networkTagsSetting":            2,
		"tagNetworkIds":                 []any{"net-a"},
		"untagNetworkIds":               []any{"net-b"},
		"voiceNetworkEnable":            true,
		"voiceNetworkId":                "net-voice",
		"flowControlEnable":             true,
		"eeeEnable":                     false,
	}
	p := NewPortDetails(m)
	if p.PortID() != "port-id-1" || p.MaxSpeed() != LinkSpeedSpeed10G || p.LinkSpeed() != LinkSpeedSpeed1G {
		t.Error("speeds")
	}
	if p.Duplex() != LinkDuplexFull || p.ProfileName() != "UPlink" || !p.HasProfileOverride() {
		t.Error("profile")
	}
	if !p.SupportsPoE() || p.PoEMode() != PoEModeUseDeviceSettings {
		t.Errorf("poe: %v %v", p.SupportsPoE(), p.PoEMode())
	}
	if p.BandwidthLimitMode() != BandwidthControlStormControl || p.Eth8021xControl() != Eth802Dot1XAuto {
		t.Error("control modes")
	}
	if !p.LLDPMedEnabled() || !p.TopologyNotifyEnabled() || !p.SpanningTreeEnabled() || !p.LoopbackDetectEnabled() || !p.PortIsolationEnabled() {
		t.Error("enable flags")
	}
	if p.LoopbackDetectVlanBasedEnabled() || !p.HasLoopbackDetectVlanBased() {
		t.Error("vlan loopback")
	}
	if p.NativeNetworkID() == nil || *p.NativeNetworkID() != "net-native" {
		t.Errorf("native: %v", p.NativeNetworkID())
	}
	if len(p.TagIDs()) != 2 || p.TagIDs()[1] != "net-b" {
		t.Errorf("tags: %v", p.TagIDs())
	}
	if p.NetworkTagsSetting() != NetworkTagsSettingCustom {
		t.Errorf("tags setting: %v", p.NetworkTagsSetting())
	}
	if len(p.TaggedNetworkIDs()) != 1 || len(p.UntaggedNetworkIDs()) != 1 {
		t.Error("network ids")
	}
	if !p.VoiceNetworkEnabled() || !p.HasVoiceNetwork() || p.VoiceNetworkID() == nil || *p.VoiceNetworkID() != "net-voice" {
		t.Error("voice network")
	}
	if !p.FlowControlEnabled() || !p.HasFlowControl() || p.EEEEnabled() || !p.HasEEE() {
		t.Error("flow control/eee presence")
	}
}

func TestPortDetailsNonPoE(t *testing.T) {
	p := NewPortDetails(map[string]any{"port": 1, "portStatus": map[string]any{"tx": 0, "rx": 0}})
	if !p.SupportsPoE() {
		t.Fatal("missing supportPoe must default to true (old controllers)")
	}
	old := NewPortDetails(map[string]any{"port": 1, "supportPoe": false, "poe": 1})
	if old.SupportsPoE() || old.PoEMode() != PoEModeNone {
		t.Errorf("non-poe port must force PoEMode.NONE: %v", old.PoEMode())
	}
}

func TestPortProfile(t *testing.T) {
	m := map[string]any{
		"id":                "profile-1",
		"site":              "site-1",
		"name":              "Trunk",
		"poe":               1,
		"bandWidthCtrlType": 1,
		"dot1x":             0,
		"lldpMedEnable":     true,
		"eeeEnable":         true,
	}
	p := NewPortProfile(m)
	if p.ProfileID() != "profile-1" || p.Site() != "site-1" || p.Name() != "Trunk" {
		t.Error("profile identity")
	}
	if p.PoEMode() != PoEModeEnabled || p.BandwidthLimitMode() != BandwidthControlRateLimit || p.Eth8021xControl() != Eth802Dot1XForceUnauthorized {
		t.Error("profile modes")
	}
	if !p.LLDPMedEnabled() || !p.EEEEnabled() || !p.HasEEE() {
		t.Error("profile flags")
	}
	if p.SpanningTreeEnabled() || p.HasFlowControl() {
		t.Error("absent flags must be off")
	}
}
