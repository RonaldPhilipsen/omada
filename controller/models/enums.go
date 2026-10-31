package models

// This file ports the IntEnum tables from upstream tplink_omada_client
// definitions.py verbatim. Unknown input maps to the enum's Unknown (or
// equivalent) member, mirroring the Python _missing_ behavior.

type DeviceStatus int32

const (
	DeviceStatusUnknown                       DeviceStatus = -1
	DeviceStatusDisconnected                  DeviceStatus = 0
	DeviceStatusDisconnectedMigrating         DeviceStatus = 1
	DeviceStatusProvisioning                  DeviceStatus = 10
	DeviceStatusConfiguring                   DeviceStatus = 11
	DeviceStatusUpgrading                     DeviceStatus = 12
	DeviceStatusRebooting                     DeviceStatus = 13
	DeviceStatusConnected                     DeviceStatus = 14
	DeviceStatusConnectedWireless             DeviceStatus = 15
	DeviceStatusConnectedMigrating            DeviceStatus = 16
	DeviceStatusConnectedWirelessMigrating    DeviceStatus = 17
	DeviceStatusPending                       DeviceStatus = 20
	DeviceStatusPendingWireless               DeviceStatus = 21
	DeviceStatusAdopting                      DeviceStatus = 22
	DeviceStatusAdoptingWireless              DeviceStatus = 23
	DeviceStatusAdoptFailed                   DeviceStatus = 24
	DeviceStatusAdoptFailedWireless           DeviceStatus = 25
	DeviceStatusManagedExternally             DeviceStatus = 26
	DeviceStatusManagedExternallyWireless     DeviceStatus = 27
	DeviceStatusHeartbeatMissed               DeviceStatus = 30
	DeviceStatusHeartbeatMissedWireless       DeviceStatus = 31
	DeviceStatusHeartbeatMissedMigrating      DeviceStatus = 32
	DeviceStatusHeartbeatMissedWirelessMigrat DeviceStatus = 33
	DeviceStatusIsolated                      DeviceStatus = 40
	DeviceStatusIsolatedMigrating             DeviceStatus = 41
)

var deviceStatusNames = map[DeviceStatus]string{
	DeviceStatusUnknown:                       "unknown",
	DeviceStatusDisconnected:                  "disconnected",
	DeviceStatusDisconnectedMigrating:         "disconnected_migrating",
	DeviceStatusProvisioning:                  "provisioning",
	DeviceStatusConfiguring:                   "configuring",
	DeviceStatusUpgrading:                     "upgrading",
	DeviceStatusRebooting:                     "rebooting",
	DeviceStatusConnected:                     "connected",
	DeviceStatusConnectedWireless:             "connected_wireless",
	DeviceStatusConnectedMigrating:            "connected_migrating",
	DeviceStatusConnectedWirelessMigrating:    "connected_wireless_migrating",
	DeviceStatusPending:                       "pending",
	DeviceStatusPendingWireless:               "pending_wireless",
	DeviceStatusAdopting:                      "adopting",
	DeviceStatusAdoptingWireless:              "adopting_wireless",
	DeviceStatusAdoptFailed:                   "adopt_failed",
	DeviceStatusAdoptFailedWireless:           "adopt_failed_wireless",
	DeviceStatusManagedExternally:             "managed_externally",
	DeviceStatusManagedExternallyWireless:     "managed_externally_wireless",
	DeviceStatusHeartbeatMissed:               "heartbeat_missed",
	DeviceStatusHeartbeatMissedWireless:       "heartbeat_missed_wireless",
	DeviceStatusHeartbeatMissedMigrating:      "heartbeat_missed_migrating",
	DeviceStatusHeartbeatMissedWirelessMigrat: "heartbeat_missed_wireless_migrating",
	DeviceStatusIsolated:                      "isolated",
	DeviceStatusIsolatedMigrating:             "isolated_migrating",
}

func (s DeviceStatus) String() string {
	if n, ok := deviceStatusNames[s]; ok {
		return n
	}
	return "unknown"
}

type DeviceStatusCategory int32

const (
	DeviceStatusCategoryUnknown         DeviceStatusCategory = -1
	DeviceStatusCategoryDisconnected    DeviceStatusCategory = 0
	DeviceStatusCategoryConnected       DeviceStatusCategory = 1
	DeviceStatusCategoryPending         DeviceStatusCategory = 2
	DeviceStatusCategoryHeartbeatMissed DeviceStatusCategory = 3
	DeviceStatusCategoryIsolated        DeviceStatusCategory = 4
)

var deviceStatusCategoryNames = map[DeviceStatusCategory]string{
	DeviceStatusCategoryUnknown:         "unknown",
	DeviceStatusCategoryDisconnected:    "disconnected",
	DeviceStatusCategoryConnected:       "connected",
	DeviceStatusCategoryPending:         "pending",
	DeviceStatusCategoryHeartbeatMissed: "heartbeat_missed",
	DeviceStatusCategoryIsolated:        "isolated",
}

func (s DeviceStatusCategory) String() string {
	if n, ok := deviceStatusCategoryNames[s]; ok {
		return n
	}
	return "unknown"
}

type PortType int32

const (
	PortTypeUnknown PortType = -1
	PortTypeCopper  PortType = 1
	PortTypeCombo   PortType = 2
	PortTypeSFP     PortType = 3
)

var portTypeNames = map[PortType]string{
	PortTypeUnknown: "unknown",
	PortTypeCopper:  "copper",
	PortTypeCombo:   "combo",
	PortTypeSFP:     "sfp",
}

func (s PortType) String() string {
	if n, ok := portTypeNames[s]; ok {
		return n
	}
	return "unknown"
}

type GatewayPortType int32

const (
	GatewayPortTypeUnknown GatewayPortType = -1
	GatewayPortTypeWAN     GatewayPortType = 0
	GatewayPortTypeWANLAN  GatewayPortType = 1
	GatewayPortTypeLAN     GatewayPortType = 2
	GatewayPortTypeSFPWAN  GatewayPortType = 3
)

var gatewayPortTypeNames = map[GatewayPortType]string{
	GatewayPortTypeUnknown: "unknown",
	GatewayPortTypeWAN:     "wan",
	GatewayPortTypeWANLAN:  "wan_lan",
	GatewayPortTypeLAN:     "lan",
	GatewayPortTypeSFPWAN:  "sfp_wan",
}

func (s GatewayPortType) String() string {
	if n, ok := gatewayPortTypeNames[s]; ok {
		return n
	}
	return "unknown"
}

type GatewayPortMode int32

const (
	GatewayPortModeDisabled GatewayPortMode = -1
	GatewayPortModeWAN      GatewayPortMode = 0
	GatewayPortModeLAN      GatewayPortMode = 1
)

var gatewayPortModeNames = map[GatewayPortMode]string{
	GatewayPortModeDisabled: "disabled",
	GatewayPortModeWAN:      "wan",
	GatewayPortModeLAN:      "lan",
}

func (s GatewayPortMode) String() string {
	if n, ok := gatewayPortModeNames[s]; ok {
		return n
	}
	return "disabled"
}

type LinkStatus int32

const (
	LinkStatusUnknown  LinkStatus = -1
	LinkStatusLinkDown LinkStatus = 0
	LinkStatusLinkUp   LinkStatus = 1
)

var linkStatusNames = map[LinkStatus]string{
	LinkStatusUnknown:  "unknown",
	LinkStatusLinkDown: "link_down",
	LinkStatusLinkUp:   "link_up",
}

func (s LinkStatus) String() string {
	if n, ok := linkStatusNames[s]; ok {
		return n
	}
	return "unknown"
}

type LinkSpeed int32

const (
	LinkSpeedUnknown   LinkSpeed = -1
	LinkSpeedSpeedAuto LinkSpeed = 0
	LinkSpeedSpeed10   LinkSpeed = 1
	LinkSpeedSpeed100  LinkSpeed = 2
	LinkSpeedSpeed1G   LinkSpeed = 3
	LinkSpeedSpeed2_5G LinkSpeed = 4
	LinkSpeedSpeed10G  LinkSpeed = 5
)

var linkSpeedNames = map[LinkSpeed]string{
	LinkSpeedUnknown:   "unknown",
	LinkSpeedSpeedAuto: "speed_auto",
	LinkSpeedSpeed10:   "speed_10_mbps",
	LinkSpeedSpeed100:  "speed_100_mbps",
	LinkSpeedSpeed1G:   "speed_1_gbps",
	LinkSpeedSpeed2_5G: "speed_2_5_gbps",
	LinkSpeedSpeed10G:  "speed_10_gbps",
}

func (s LinkSpeed) String() string {
	if n, ok := linkSpeedNames[s]; ok {
		return n
	}
	return "unknown"
}

type LinkDuplex int32

const (
	LinkDuplexUnknown LinkDuplex = -1
	LinkDuplexAuto    LinkDuplex = 0
	LinkDuplexHalf    LinkDuplex = 1
	LinkDuplexFull    LinkDuplex = 2
)

var linkDuplexNames = map[LinkDuplex]string{
	LinkDuplexUnknown: "unknown",
	LinkDuplexAuto:    "auto",
	LinkDuplexHalf:    "half",
	LinkDuplexFull:    "full",
}

func (s LinkDuplex) String() string {
	if n, ok := linkDuplexNames[s]; ok {
		return n
	}
	return "unknown"
}

type Eth802Dot1X int32

const (
	Eth802Dot1XUnknown           Eth802Dot1X = -1
	Eth802Dot1XForceUnauthorized Eth802Dot1X = 0
	Eth802Dot1XForceAuthorized   Eth802Dot1X = 1
	Eth802Dot1XAuto              Eth802Dot1X = 2
)

var eth802Dot1XNames = map[Eth802Dot1X]string{
	Eth802Dot1XUnknown:           "unknown",
	Eth802Dot1XForceUnauthorized: "force_unauthorized",
	Eth802Dot1XForceAuthorized:   "force_authorized",
	Eth802Dot1XAuto:              "auto",
}

func (s Eth802Dot1X) String() string {
	if n, ok := eth802Dot1XNames[s]; ok {
		return n
	}
	return "unknown"
}

type NetworkTagsSetting int32

const (
	NetworkTagsSettingUnknown  NetworkTagsSetting = -1
	NetworkTagsSettingAllowAll NetworkTagsSetting = 0
	NetworkTagsSettingBlockAll NetworkTagsSetting = 1
	NetworkTagsSettingCustom   NetworkTagsSetting = 2
)

var networkTagsSettingNames = map[NetworkTagsSetting]string{
	NetworkTagsSettingUnknown:  "unknown",
	NetworkTagsSettingAllowAll: "allow_all",
	NetworkTagsSettingBlockAll: "block_all",
	NetworkTagsSettingCustom:   "custom",
}

func (s NetworkTagsSetting) String() string {
	if n, ok := networkTagsSettingNames[s]; ok {
		return n
	}
	return "unknown"
}

type BandwidthControl int32

const (
	BandwidthControlUnknown      BandwidthControl = -1
	BandwidthControlOff          BandwidthControl = 0
	BandwidthControlRateLimit    BandwidthControl = 1
	BandwidthControlStormControl BandwidthControl = 2
)

var bandwidthControlNames = map[BandwidthControl]string{
	BandwidthControlUnknown:      "unknown",
	BandwidthControlOff:          "off",
	BandwidthControlRateLimit:    "rate_limit",
	BandwidthControlStormControl: "storm_control",
}

func (s BandwidthControl) String() string {
	if n, ok := bandwidthControlNames[s]; ok {
		return n
	}
	return "unknown"
}

type PoEMode int32

const (
	PoEModeNone              PoEMode = -1
	PoEModeDisabled          PoEMode = 0
	PoEModeEnabled           PoEMode = 1
	PoEModeUseDeviceSettings PoEMode = 2
)

var poEModeNames = map[PoEMode]string{
	PoEModeNone:              "none",
	PoEModeDisabled:          "disabled",
	PoEModeEnabled:           "enabled",
	PoEModeUseDeviceSettings: "use_device_settings",
}

func (s PoEMode) String() string {
	if n, ok := poEModeNames[s]; ok {
		return n
	}
	return "none"
}

type AuthenticationStatus int32

const (
	AuthenticationStatusUnknown    AuthenticationStatus = -1
	AuthenticationStatusConnected  AuthenticationStatus = 0
	AuthenticationStatusPending    AuthenticationStatus = 1
	AuthenticationStatusAuthorized AuthenticationStatus = 2
	AuthenticationStatusAuthFree   AuthenticationStatus = 3
)

var authenticationStatusNames = map[AuthenticationStatus]string{
	AuthenticationStatusUnknown:    "unknown",
	AuthenticationStatusConnected:  "connected",
	AuthenticationStatusPending:    "pending",
	AuthenticationStatusAuthorized: "authorized",
	AuthenticationStatusAuthFree:   "auth_free",
}

func (s AuthenticationStatus) String() string {
	if n, ok := authenticationStatusNames[s]; ok {
		return n
	}
	return "unknown"
}

type ConnectType int32

const (
	ConnectTypeUnknown       ConnectType = -1
	ConnectTypeGuestWireless ConnectType = 0
	ConnectTypeWireless      ConnectType = 1
	ConnectTypeWired         ConnectType = 2
)

var connectTypeNames = map[ConnectType]string{
	ConnectTypeUnknown:       "unknown",
	ConnectTypeGuestWireless: "guest_wireless",
	ConnectTypeWireless:      "wireless",
	ConnectTypeWired:         "wired",
}

func (s ConnectType) String() string {
	if n, ok := connectTypeNames[s]; ok {
		return n
	}
	return "unknown"
}

type RadioId int32

const (
	RadioIdUnknown RadioId = -1
	RadioIdFreq24  RadioId = 0
	RadioIdFreq51  RadioId = 1
	RadioIdFreq52  RadioId = 2
	RadioIdFreq6   RadioId = 3
)

var radioIdNames = map[RadioId]string{
	RadioIdUnknown: "unknown",
	RadioIdFreq24:  "freq_2_4",
	RadioIdFreq51:  "freq_5_1",
	RadioIdFreq52:  "freq_5_2",
	RadioIdFreq6:   "freq_6",
}

var radioSettingsKeys = map[RadioId]string{
	RadioIdFreq24: "radioSetting2g",
	RadioIdFreq51: "radioSetting5g",
	RadioIdFreq52: "radioSetting5g2",
	RadioIdFreq6:  "radioSetting6g",
}

var radioStatusKeys = map[RadioId]string{
	RadioIdFreq24: "wp2g",
	RadioIdFreq51: "wp5g",
	RadioIdFreq52: "wp5g2",
	RadioIdFreq6:  "wp6g",
}

func (r RadioId) String() string {
	if n, ok := radioIdNames[r]; ok {
		return n
	}
	return "unknown"
}

// SettingsKey is the field name the controller uses for this radio's settings.
func (r RadioId) SettingsKey() string { return radioSettingsKeys[r] }

// StatusKey is the field name the controller uses for this radio's live status.
func (r RadioId) StatusKey() string { return radioStatusKeys[r] }

type WifiMode int32

const (
	WifiModeUnknown WifiMode = -1
	WifiModeA       WifiMode = 0
	WifiModeB       WifiMode = 1
	WifiModeG       WifiMode = 2
	WifiModeNA      WifiMode = 3
	WifiModeNG      WifiMode = 4
	WifiModeAC      WifiMode = 5
	WifiModeAXA     WifiMode = 6
	WifiModeAXG     WifiMode = 7
)

var wifiModeNames = map[WifiMode]string{
	WifiModeUnknown: "unknown",
	WifiModeA:       "a",
	WifiModeB:       "b",
	WifiModeG:       "g",
	WifiModeNA:      "na",
	WifiModeNG:      "ng",
	WifiModeAC:      "ac",
	WifiModeAXA:     "axa",
	WifiModeAXG:     "axg",
}

func (s WifiMode) String() string {
	if n, ok := wifiModeNames[s]; ok {
		return n
	}
	return "unknown"
}

type LedSetting int32

const (
	LedSettingUnknown      LedSetting = -1
	LedSettingOff          LedSetting = 0
	LedSettingOn           LedSetting = 1
	LedSettingSiteSettings LedSetting = 2
)

var ledSettingNames = map[LedSetting]string{
	LedSettingUnknown:      "unknown",
	LedSettingOff:          "off",
	LedSettingOn:           "on",
	LedSettingSiteSettings: "site_settings",
}

func (s LedSetting) String() string {
	if n, ok := ledSettingNames[s]; ok {
		return n
	}
	return "unknown"
}

type ChannelWidth int32

const (
	ChannelWidthUnknown       ChannelWidth = -1
	ChannelWidthWidth20       ChannelWidth = 2
	ChannelWidthWidth40       ChannelWidth = 3
	ChannelWidthAuto4020      ChannelWidth = 4
	ChannelWidthWidth80       ChannelWidth = 5
	ChannelWidthAuto804020    ChannelWidth = 6
	ChannelWidthWidth160      ChannelWidth = 7
	ChannelWidthAuto160804020 ChannelWidth = 8
	ChannelWidthWidth240      ChannelWidth = 9
	ChannelWidthWidth320      ChannelWidth = 10
)

var channelWidthNames = map[ChannelWidth]string{
	ChannelWidthUnknown:       "unknown",
	ChannelWidthWidth20:       "width_20",
	ChannelWidthWidth40:       "width_40",
	ChannelWidthAuto4020:      "auto_40_20",
	ChannelWidthWidth80:       "width_80",
	ChannelWidthAuto804020:    "auto_80_40_20",
	ChannelWidthWidth160:      "width_160",
	ChannelWidthAuto160804020: "auto_160_80_40_20",
	ChannelWidthWidth240:      "width_240",
	ChannelWidthWidth320:      "width_320",
}

func (s ChannelWidth) String() string {
	if n, ok := channelWidthNames[s]; ok {
		return n
	}
	return "unknown"
}
