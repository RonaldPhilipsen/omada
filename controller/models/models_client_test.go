package models

import "testing"

func TestClientBaseWired(t *testing.T) {
	c := NewClient(loadFixture(t, "client-wired.json"))
	if c.MAC() != "DE:AD:BE:EF:00:01" || c.Name() != "Workstation 1" {
		t.Error("identity")
	}
	if c.IsGuest() || c.IsBlocked() {
		t.Error("guest/block flags")
	}
	if c.LastSeen() != 1735000000 {
		t.Errorf("lastSeen must be seconds: %v", c.LastSeen())
	}
	if c.ConnectionTime() == nil || *c.ConnectionTime() != 3600 {
		t.Errorf("connection time from uptime: %v", c.ConnectionTime())
	}
	if c.IsWireless() == nil || *c.IsWireless() {
		t.Error("wireless flag")
	}
	if !c.IsActive() || c.Activity() != 1500 {
		t.Error("activity")
	}
	if c.AuthStatus() != AuthenticationStatusAuthorized || c.ConnectType() != ConnectTypeWired {
		t.Error("auth/connect type")
	}
	if c.ConnectDevType() == nil || *c.ConnectDevType() != "gateway" {
		t.Errorf("connect dev: %v", c.ConnectDevType())
	}
	if c.DeviceType() != "Desktop" || c.DownPacket() != 1000 {
		t.Error("device type/packets")
	}
	if c.HostName() == nil || *c.HostName() != "ws1" {
		t.Errorf("hostname: %v", c.HostName())
	}
	if c.IP() == nil || *c.IP() != "192.168.1.50" {
		t.Errorf("ip: %v", c.IP())
	}
	if c.TrafficDown() != 5000000 || c.TrafficUp() != 2000000 || c.UpPacket() != 800 {
		t.Error("traffic")
	}
	if c.VLAN() == nil || *c.VLAN() != 10 {
		t.Errorf("vid: %v", c.VLAN())
	}
	if c.DeviceCategory() == nil || *c.DeviceCategory() != "Computer" {
		t.Errorf("category: %v", c.DeviceCategory())
	}
	if c.OSName() == nil || *c.OSName() != "Windows 11" {
		t.Errorf("os: %v", c.OSName())
	}
	if c.Vendor() == nil || *c.Vendor() != "Dell" {
		t.Errorf("vendor: %v", c.Vendor())
	}
	if !c.IPSetting().UsedFixedAddr() {
		t.Error("fixed addr")
	}
	rl := c.RateLimit()
	if !rl.Enabled() || !rl.DownEnabled() || rl.DownLimit() != 100 || rl.DownUnit() != 1 || rl.UpEnabled() {
		t.Error("rate limit")
	}

	w := c.Wired
	if w == nil {
		t.Fatal("wired details must be populated from connectType=2")
	}
	if w.Dot1xVlan() != 20 || w.Port() != 4 || w.NetworkName() != "Office LAN" {
		t.Error("wired basics")
	}
	if w.SwitchMAC() == nil || *w.SwitchMAC() != "AA:BB:CC:DD:EE:03" {
		t.Errorf("switch mac: %v", w.SwitchMAC())
	}
	if w.SwitchName() == nil || *w.SwitchName() != "Rack Switch" {
		t.Errorf("switch name: %v", w.SwitchName())
	}
	if w.GatewayMAC() == nil || *w.GatewayMAC() != "AA:BB:CC:DD:EE:01" {
		t.Errorf("gateway mac: %v", w.GatewayMAC())
	}
	if w.GatewayName() == nil || *w.GatewayName() != "Main Gateway" {
		t.Errorf("gateway name: %v", w.GatewayName())
	}
	if c.Wireless != nil {
		t.Error("wired client has no wireless details")
	}
}

func TestClientBaseWireless(t *testing.T) {
	c := NewClient(loadFixture(t, "client-wireless.json"))
	if !c.IsGuest() {
		t.Error("guest wireless client must be a guest")
	}
	if w := c.IsWireless(); w == nil || !*w {
		t.Error("client must be wireless")
	}
	if c.ConnectionTime() == nil || *c.ConnectionTime() != 1200 {
		t.Errorf("connection time from duration: %v", c.ConnectionTime())
	}
	if c.AuthStatus() != AuthenticationStatusAuthFree || c.ConnectType() != ConnectTypeGuestWireless {
		t.Error("auth/connect type")
	}
	w := c.Wireless
	if w == nil {
		t.Fatal("wireless details must be populated from connectType in {0,1}")
	}
	if w.APMAC() != "AA:BB:CC:DD:EE:02" || w.APName() != "Lobby AP" || w.SSID() != "Omada-Guest" {
		t.Error("ap identity")
	}
	if w.Channel() != 36 || w.RadioID() != RadioIdFreq51 || w.RSSI() != -55 {
		t.Error("radio basics")
	}
	if w.RxRate() != 405000 || w.TxRate() != 800000 {
		t.Error("rates")
	}
	if w.SignalLevel() != 80 || w.SignalRank() != 4 || w.PowerSave() {
		t.Error("signal")
	}
	if w.WifiMode() != WifiModeAC {
		t.Errorf("wifi mode: %v", w.WifiMode())
	}
	if c.Wired != nil {
		t.Error("wireless client has no wired details")
	}
}

func TestClientSparse(t *testing.T) {
	c := NewClient(loadFixture(t, "client-sparse.json"))
	if c.ConnectionTime() != nil {
		t.Errorf("no uptime/duration: %v", c.ConnectionTime())
	}
	// known_clients endpoint uses 'block' instead of 'blocked'.
	if !c.IsBlocked() {
		t.Error("block key must be honoured")
	}
	if c.IsWireless() != nil {
		t.Error("wireless absent must be nil")
	}
	if c.IsActive() || c.Activity() != 0 {
		t.Error("absent activity must be zero")
	}
	if c.VLAN() != nil || c.HostName() != nil || c.IP() != nil {
		t.Error("absent optionals must be nil")
	}
	if c.Wired != nil || c.Wireless != nil {
		t.Error("sparse client has no details")
	}
	if c.LastSeen() != 1734000000 {
		t.Errorf("lastSeen: %v", c.LastSeen())
	}
}
