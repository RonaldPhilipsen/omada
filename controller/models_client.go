package controller

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
	if v := mapPtrInt(data, "connectType"); v != nil {
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
	if v := mapPtrInt(c.data, "uptime"); v != nil {
		return v
	}
	return mapPtrInt(c.data, "duration")
}

// IsBlocked reports whether the client is blocked. Most API calls return
// "blocked"; the known_clients call returns "block".
func (c Client) IsBlocked() bool {
	if v := mapPtrBool(c.data, "blocked"); v != nil {
		return *v
	}
	return mapBool(c.data, "block")
}

// IsGuest reports whether the client is a guest.
func (c Client) IsGuest() bool { return mapBool(c.data, "guest") }

// LastSeen is the Unix timestamp (seconds) when the client was last
// connected.
func (c Client) LastSeen() float64 {
	return float64(mapInt(c.data, "lastSeen")) / 1000
}

// MAC is the MAC address of the client.
func (c Client) MAC() string { return mapStr(c.data, "mac") }

// Name is the name of the client.
func (c Client) Name() string { return mapStr(c.data, "name") }

// IsWireless reports whether the client is wireless, nil when unreported.
func (c Client) IsWireless() *bool { return mapPtrBool(c.data, "wireless") }

// IsActive reports whether the client is active.
func (c Client) IsActive() bool { return mapBool(c.data, "active") }

// Activity is the realtime downlink rate in bytes/s.
func (c Client) Activity() int { return mapInt(c.data, "activity") }

// AuthStatus is the authentication status.
func (c Client) AuthStatus() AuthenticationStatus {
	return AuthenticationStatus(mapInt(c.data, "authStatus"))
}

// ConnectDevType is the type of device the client is connected to.
func (c Client) ConnectDevType() *string {
	return mapPtrStr(c.data, "connectDevType")
}

// ConnectType is the connection type.
func (c Client) ConnectType() ConnectType {
	return ConnectType(mapInt(c.data, "connectType"))
}

// DeviceType is the client device type (Android, iPhone, ...).
func (c Client) DeviceType() string { return mapStr(c.data, "deviceType") }

// DownPacket is the number of packets sent to the client.
func (c Client) DownPacket() int { return mapInt(c.data, "downPacket") }

// HostName is the client's hostname.
func (c Client) HostName() *string { return mapPtrStr(c.data, "hostName") }

// IP is the client's IP address.
func (c Client) IP() *string { return mapPtrStr(c.data, "ip") }

// TrafficDown is the number of bytes sent to the client.
func (c Client) TrafficDown() int { return mapInt(c.data, "trafficDown") }

// TrafficUp is the number of bytes sent by the client.
func (c Client) TrafficUp() int { return mapInt(c.data, "trafficUp") }

// UpPacket is the number of packets received from the client.
func (c Client) UpPacket() int { return mapInt(c.data, "upPacket") }

// VLAN is the VLAN ID of the client. Some access points do not report it.
func (c Client) VLAN() *int { return mapPtrInt(c.data, "vid") }

// DeviceCategory is the device category.
func (c Client) DeviceCategory() *string {
	return mapPtrStr(c.data, "deviceCategory")
}

// IPSetting is the IP reservation setting of the client.
func (c Client) IPSetting() IpSetting {
	m, _ := mapMap(c.data, "ipSetting")
	return NewIpSetting(m)
}

// OSName is the operating system name.
func (c Client) OSName() *string { return mapPtrStr(c.data, "osName") }

// RateLimit is the rate limit setting of the client.
func (c Client) RateLimit() RateLimit {
	m, _ := mapMap(c.data, "rateLimit")
	return NewRateLimit(m)
}

// Vendor is the vendor of the device.
func (c Client) Vendor() *string { return mapPtrStr(c.data, "vendor") }

// IpSetting is the IP setting of a client.
type IpSetting struct {
	data map[string]any
}

// NewIpSetting builds an IpSetting from a raw API object.
func NewIpSetting(data map[string]any) IpSetting { return IpSetting{data: data} }

// UsedFixedAddr reports whether a fixed IP address is reserved.
func (s IpSetting) UsedFixedAddr() bool { return mapBool(s.data, "useFixedAddr") }

// RateLimit is the rate limit setting of a client.
type RateLimit struct {
	data map[string]any
}

// NewRateLimit builds a RateLimit from a raw API object.
func NewRateLimit(data map[string]any) RateLimit { return RateLimit{data: data} }

// Enabled reports whether rate limiting is enabled.
func (r RateLimit) Enabled() bool { return mapBool(r.data, "enable") }

// DownEnabled reports whether downstream rate limiting is enabled.
func (r RateLimit) DownEnabled() bool { return mapBool(r.data, "downEnable") }

// DownLimit is the downstream rate limit.
func (r RateLimit) DownLimit() int { return mapInt(r.data, "downLimit") }

// DownUnit is the downstream rate limit unit.
func (r RateLimit) DownUnit() int { return mapInt(r.data, "downUnit") }

// UpEnabled reports whether upstream rate limiting is enabled.
func (r RateLimit) UpEnabled() bool { return mapBool(r.data, "upEnable") }

// UpLimit is the upstream rate limit.
func (r RateLimit) UpLimit() int { return mapInt(r.data, "upLimit") }

// UpUnit is the upstream rate limit unit.
func (r RateLimit) UpUnit() int { return mapInt(r.data, "upUnit") }

// WiredDetails are the wired-connection details of a client.
type WiredDetails struct {
	data map[string]any
}

// NewWiredDetails builds a WiredDetails from a raw API object.
func NewWiredDetails(data map[string]any) WiredDetails {
	return WiredDetails{data: data}
}

// Dot1xVlan is the VLAN obtained by 802.1x D-VLAN.
func (w WiredDetails) Dot1xVlan() int { return mapInt(w.data, "dot1xVlan") }

// GatewayMAC is the MAC of the gateway the client is connected to.
func (w WiredDetails) GatewayMAC() *string {
	return mapPtrStr(w.data, "gatewayMac")
}

// GatewayName is the name of the gateway the client is connected to.
func (w WiredDetails) GatewayName() *string {
	return mapPtrStr(w.data, "gatewayName")
}

// NetworkName is the network name.
func (w WiredDetails) NetworkName() string {
	return mapStr(w.data, "networkName")
}

// Port is the switch port the client is connected to.
func (w WiredDetails) Port() int {
	if v := mapPtrInt(w.data, "port"); v != nil {
		return *v
	}
	return -1
}

// SwitchMAC is the MAC of the switch the client is connected to.
func (w WiredDetails) SwitchMAC() *string {
	return mapPtrStr(w.data, "switchMac")
}

// SwitchName is the name of the switch the client is connected to.
func (w WiredDetails) SwitchName() *string {
	return mapPtrStr(w.data, "switchName")
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
func (w WirelessDetails) APMAC() string { return mapStr(w.data, "apMac") }

// APName is the name of the access point.
func (w WirelessDetails) APName() string { return mapStr(w.data, "apName") }

// Channel is the wifi channel number.
func (w WirelessDetails) Channel() int { return mapInt(w.data, "channel") }

// PowerSave reports whether power save mode is enabled.
func (w WirelessDetails) PowerSave() bool { return mapBool(w.data, "powerSave") }

// RadioID is the radio band the client is connected on.
func (w WirelessDetails) RadioID() RadioId {
	if v := mapPtrInt(w.data, "radioId"); v != nil {
		return RadioId(*v)
	}
	return RadioIdUnknown
}

// RSSI is the signal strength in dBm.
func (w WirelessDetails) RSSI() int { return mapInt(w.data, "rssi") }

// RxRate is the uplink negotiation rate in Kbit/s.
func (w WirelessDetails) RxRate() int { return mapInt(w.data, "rxRate") }

// SignalLevel is the signal strength percentage.
func (w WirelessDetails) SignalLevel() int { return mapInt(w.data, "signalLevel") }

// SignalRank is the signal strength on a 0-5 scale.
func (w WirelessDetails) SignalRank() int { return mapInt(w.data, "signalRank") }

// SSID is the SSID name.
func (w WirelessDetails) SSID() string { return mapStr(w.data, "ssid") }

// TxRate is the downlink negotiation rate in Kbit/s.
func (w WirelessDetails) TxRate() int { return mapInt(w.data, "txRate") }

// WifiMode is the wifi mode.
func (w WirelessDetails) WifiMode() WifiMode {
	if v := mapPtrInt(w.data, "wifiMode"); v != nil {
		return WifiMode(*v)
	}
	return WifiModeUnknown
}
