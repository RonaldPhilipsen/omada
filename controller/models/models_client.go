package models

// This file ports the client model classes from upstream
// tplink_omada_client clients.py. The upstream hierarchy
// (OmadaNetworkClient / OmadaConnectedClient / OmadaClientDetails with
// wired/wireless specialisations) is flattened into one Client, whose
// Wired and Wireless details are populated based on connectType.

// Client is a device connected to the Omada network.
type Client struct {
	data map[string]any
	// Wired holds switch/gateway port details. Populated for wired clients.
	Wired *WiredDetails
	// Wireless holds AP radio details. Populated for wireless clients.
	Wireless *WirelessDetails
}

// NewClient builds a Client from a raw API object.
func NewClient(data map[string]any) Client {
	c := Client{data: data}
	if v := MapPtrInt(data, "connectType"); v != nil {
		switch ConnectType(*v) {
		case ConnectTypeWired:
			c.Wired = &WiredDetails{data: data}
		case ConnectTypeWireless, ConnectTypeGuestWireless:
			c.Wireless = &WirelessDetails{data: data}
		}
	}
	return c
}

// ConnectionTime is the client connection time in seconds.
func (c Client) ConnectionTime() *int {
	if v := MapPtrInt(c.data, "uptime"); v != nil {
		return v
	}
	return MapPtrInt(c.data, "duration")
}

// IsBlocked reports whether the client is blocked. Most API calls return
// "blocked"; the known_clients call returns "block".
func (c Client) IsBlocked() bool {
	if v := MapPtrBool(c.data, "blocked"); v != nil {
		return *v
	}
	return MapBool(c.data, "block")
}

// IsGuest reports whether the client is a guest.
func (c Client) IsGuest() bool { return MapBool(c.data, "guest") }

// LastSeen is the Unix timestamp (seconds) when the client was last
// connected.
func (c Client) LastSeen() float64 {
	return float64(MapInt(c.data, "lastSeen")) / 1000
}

// MAC is the MAC address of the client.
func (c Client) MAC() string { return MapStr(c.data, "mac") }

// Name is the name of the client.
func (c Client) Name() string { return MapStr(c.data, "name") }

// IsWireless reports whether the client is wireless, nil when unreported.
func (c Client) IsWireless() *bool { return MapPtrBool(c.data, "wireless") }

// IsActive reports whether the client is active.
func (c Client) IsActive() bool { return MapBool(c.data, "active") }

// Activity is the realtime downlink rate in bytes/s.
func (c Client) Activity() int { return MapInt(c.data, "activity") }

// AuthStatus is the authentication status.
func (c Client) AuthStatus() AuthenticationStatus {
	return AuthenticationStatus(MapInt(c.data, "authStatus"))
}

// ConnectDevType is the type of device the client is connected to.
func (c Client) ConnectDevType() *string {
	return MapPtrStr(c.data, "connectDevType")
}

// ConnectType is the connection type.
func (c Client) ConnectType() ConnectType {
	return ConnectType(MapInt(c.data, "connectType"))
}

// DeviceType is the client device type (Android, iPhone, ...).
func (c Client) DeviceType() string { return MapStr(c.data, "deviceType") }

// DownPacket is the number of packets sent to the client.
func (c Client) DownPacket() int { return MapInt(c.data, "downPacket") }

// HostName is the client's hostname.
func (c Client) HostName() *string { return MapPtrStr(c.data, "hostName") }

// IP is the client's IP address.
func (c Client) IP() *string { return MapPtrStr(c.data, "ip") }

// TrafficDown is the number of bytes sent to the client.
func (c Client) TrafficDown() int { return MapInt(c.data, "trafficDown") }

// TrafficUp is the number of bytes sent by the client.
func (c Client) TrafficUp() int { return MapInt(c.data, "trafficUp") }

// UpPacket is the number of packets received from the client.
func (c Client) UpPacket() int { return MapInt(c.data, "upPacket") }

// VLAN is the VLAN ID of the client. Some access points do not report it.
func (c Client) VLAN() *int { return MapPtrInt(c.data, "vid") }

// DeviceCategory is the device category.
func (c Client) DeviceCategory() *string {
	return MapPtrStr(c.data, "deviceCategory")
}

// IPSetting is the IP reservation setting of the client.
func (c Client) IPSetting() IpSetting {
	m, _ := MapMap(c.data, "ipSetting")
	return NewIpSetting(m)
}

// OSName is the operating system name.
func (c Client) OSName() *string { return MapPtrStr(c.data, "osName") }

// RateLimit is the rate limit setting of the client.
func (c Client) RateLimit() RateLimit {
	m, _ := MapMap(c.data, "rateLimit")
	return NewRateLimit(m)
}

// Vendor is the vendor of the device.
func (c Client) Vendor() *string { return MapPtrStr(c.data, "vendor") }

// IpSetting is the IP setting of a client.
type IpSetting struct {
	data map[string]any
}

// NewIpSetting builds an IpSetting from a raw API object.
func NewIpSetting(data map[string]any) IpSetting { return IpSetting{data: data} }

// Raw is the underlying API object.
func (s IpSetting) Raw() map[string]any { return s.data }

// UsedFixedAddr reports whether a fixed IP address is reserved.
func (s IpSetting) UsedFixedAddr() bool { return MapBool(s.data, "useFixedAddr") }

// RateLimit is the rate limit setting of a client.
type RateLimit struct {
	data map[string]any
}

// NewRateLimit builds a RateLimit from a raw API object.
func NewRateLimit(data map[string]any) RateLimit { return RateLimit{data: data} }

// Enabled reports whether rate limiting is enabled.
func (r RateLimit) Enabled() bool { return MapBool(r.data, "enable") }

// DownEnabled reports whether downstream rate limiting is enabled.
func (r RateLimit) DownEnabled() bool { return MapBool(r.data, "downEnable") }

// DownLimit is the downstream rate limit.
func (r RateLimit) DownLimit() int { return MapInt(r.data, "downLimit") }

// DownUnit is the downstream rate limit unit.
func (r RateLimit) DownUnit() int { return MapInt(r.data, "downUnit") }

// UpEnabled reports whether upstream rate limiting is enabled.
func (r RateLimit) UpEnabled() bool { return MapBool(r.data, "upEnable") }

// UpLimit is the upstream rate limit.
func (r RateLimit) UpLimit() int { return MapInt(r.data, "upLimit") }

// UpUnit is the upstream rate limit unit.
func (r RateLimit) UpUnit() int { return MapInt(r.data, "upUnit") }

// WiredDetails are the wired-connection details of a client.
type WiredDetails struct {
	data map[string]any
}

// NewWiredDetails builds a WiredDetails from a raw API object.
func NewWiredDetails(data map[string]any) WiredDetails {
	return WiredDetails{data: data}
}

// Dot1xVlan is the VLAN obtained by 802.1x D-VLAN.
func (w WiredDetails) Dot1xVlan() int { return MapInt(w.data, "dot1xVlan") }

// GatewayMAC is the MAC of the gateway the client is connected to.
func (w WiredDetails) GatewayMAC() *string {
	return MapPtrStr(w.data, "gatewayMac")
}

// GatewayName is the name of the gateway the client is connected to.
func (w WiredDetails) GatewayName() *string {
	return MapPtrStr(w.data, "gatewayName")
}

// NetworkName is the network name.
func (w WiredDetails) NetworkName() string {
	return MapStr(w.data, "networkName")
}

// Port is the switch port the client is connected to.
func (w WiredDetails) Port() int {
	if v := MapPtrInt(w.data, "port"); v != nil {
		return *v
	}
	return -1
}

// SwitchMAC is the MAC of the switch the client is connected to.
func (w WiredDetails) SwitchMAC() *string {
	return MapPtrStr(w.data, "switchMac")
}

// SwitchName is the name of the switch the client is connected to.
func (w WiredDetails) SwitchName() *string {
	return MapPtrStr(w.data, "switchName")
}

// WirelessDetails are the wireless-connection details of a client.
type WirelessDetails struct {
	data map[string]any
}

// NewWirelessDetails builds a WirelessDetails from a raw API object.
func NewWirelessDetails(data map[string]any) WirelessDetails {
	return WirelessDetails{data: data}
}

// APMAC is the MAC address of the access point.
func (w WirelessDetails) APMAC() string { return MapStr(w.data, "apMac") }

// APName is the name of the access point.
func (w WirelessDetails) APName() string { return MapStr(w.data, "apName") }

// Channel is the wifi channel number.
func (w WirelessDetails) Channel() int { return MapInt(w.data, "channel") }

// PowerSave reports whether power save mode is enabled.
func (w WirelessDetails) PowerSave() bool { return MapBool(w.data, "powerSave") }

// RadioID is the radio band the client is connected on.
func (w WirelessDetails) RadioID() RadioId {
	if v := MapPtrInt(w.data, "radioId"); v != nil {
		return RadioId(*v)
	}
	return RadioIdUnknown
}

// RSSI is the signal strength in dBm.
func (w WirelessDetails) RSSI() int { return MapInt(w.data, "rssi") }

// RxRate is the uplink negotiation rate in Kbit/s.
func (w WirelessDetails) RxRate() int { return MapInt(w.data, "rxRate") }

// SignalLevel is the signal strength percentage.
func (w WirelessDetails) SignalLevel() int { return MapInt(w.data, "signalLevel") }

// SignalRank is the signal strength on a 0-5 scale.
func (w WirelessDetails) SignalRank() int { return MapInt(w.data, "signalRank") }

// SSID is the SSID name.
func (w WirelessDetails) SSID() string { return MapStr(w.data, "ssid") }

// TxRate is the downlink negotiation rate in Kbit/s.
func (w WirelessDetails) TxRate() int { return MapInt(w.data, "txRate") }

// WifiMode is the wifi mode.
func (w WirelessDetails) WifiMode() WifiMode {
	if v := MapPtrInt(w.data, "wifiMode"); v != nil {
		return WifiMode(*v)
	}
	return WifiModeUnknown
}
