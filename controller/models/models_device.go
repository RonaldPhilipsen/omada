package models

// This file ports the device model classes from upstream
// tplink_omada_client devices.py. The upstream hierarchy
// (OmadaDevice / OmadaListDevice / OmadaDetailedDevice + per-type
// subclasses) is flattened: one Device carries every accessor, and the
// per-type structs (Switch, AccessPoint, Gateway) add their own on top.

// Device is a managed Omada device (gateway, switch, access point).
type Device struct {
	data map[string]any
}

// NewDevice builds a Device from a raw API object.
func NewDevice(data map[string]any) Device { return Device{data: data} }

// Type returns "ap", "gateway", or "switch".
func (d Device) Type() string { return MapStr(d.data, "type") }

// ResourcePath is the API path for querying detailed data about this device.
func (d Device) ResourcePath() string {
	types := map[string]string{"ap": "eaps", "gateway": "gateways", "switch": "switches"}
	return types[d.Type()] + "/" + d.MAC()
}

// MAC is the device's MAC address.
func (d Device) MAC() string { return MapStr(d.data, "mac") }

// Name is the device name, falling back to the MAC.
func (d Device) Name() string {
	if n := MapStr(d.data, "name"); n != "" {
		return n
	}
	return d.MAC()
}

// Model is the device model, such as EAP225.
func (d Device) Model() string {
	if m := MapStr(d.data, "model"); m != "" {
		return m
	}
	return "Unknown"
}

// ModelDisplayName is the model description for front-end display.
func (d Device) ModelDisplayName() string {
	if m := MapStr(d.data, "showModel"); m != "" {
		return m
	}
	return "Unknown Model"
}

// Status is the device status.
func (d Device) Status() DeviceStatus {
	if s := MapPtrInt(d.data, "status"); s != nil {
		return DeviceStatus(*s)
	}
	return DeviceStatusUnknown
}

// StatusCategory is the high-level device status.
func (d Device) StatusCategory() DeviceStatusCategory {
	if s := MapPtrInt(d.data, "statusCategory"); s != nil {
		return DeviceStatusCategory(*s)
	}
	return DeviceStatusCategoryUnknown
}

// IPAddress is the device IP address.
func (d Device) IPAddress() string { return MapStr(d.data, "ip") }

func (d Device) connected() bool {
	return d.data != nil && d.StatusCategory() == DeviceStatusCategoryConnected
}

// DisplayUptime is the uptime as a display string, nil when not connected.
func (d Device) DisplayUptime() *string {
	if !d.connected() {
		return nil
	}
	return MapPtrStr(d.data, "uptime")
}

// CPUUsage is the current CPU usage percentage.
func (d Device) CPUUsage() int {
	if !d.connected() {
		return 0
	}
	return MapInt(d.data, "cpuUtil")
}

// MemUsage is the current memory usage percentage.
func (d Device) MemUsage() int {
	if !d.connected() {
		return 0
	}
	return MapInt(d.data, "memUtil")
}

// Uptime is the uptime in seconds.
func (d Device) Uptime() int {
	if !d.connected() {
		return 0
	}
	return MapInt(d.data, "uptimeLong")
}

// FirmwareVersion is the running firmware version.
func (d Device) FirmwareVersion() string {
	if !d.connected() {
		return "Unknown"
	}
	return MapStr(d.data, "firmwareVersion")
}

// NeedUpgrade reports whether a firmware upgrade is available.
func (d Device) NeedUpgrade() bool {
	return d.connected() && MapBool(d.data, "needUpgrade")
}

// FWDownload reports whether a firmware upgrade is being downloaded.
func (d Device) FWDownload() bool {
	return d.connected() && MapBool(d.data, "fwDownload")
}

// LedSetting is the onboard LED setting.
func (d Device) LedSetting() LedSetting {
	if s := MapPtrInt(d.data, "ledSetting"); s != nil {
		return LedSetting(*s)
	}
	return LedSettingUnknown
}

// Link is an up/downlink connection from a switch or AP port.
type Link struct {
	data map[string]any
}

// NewLink builds a Link from a raw API object.
func NewLink(data map[string]any) Link { return Link{data: data} }

// MAC is the MAC of the linked device.
func (l Link) MAC() string {
	if m := MapStr(l.data, "mac"); m != "" {
		return m
	}
	return MapStr(l.data, "uplinkMac")
}

// Name is the name of the linked device.
func (l Link) Name() string { return MapStr(l.data, "name") }

// Type is the type of the linked device.
func (l Link) Type() string { return MapStr(l.data, "type") }

// Port is the linked port number.
func (l Link) Port() int { return MapInt(l.data, "port") }

// Downlink is a downlink device attached to a switch port.
type Downlink struct {
	data map[string]any
}

// NewDownlink builds a Downlink from a raw API object.
func NewDownlink(data map[string]any) Downlink { return Downlink{data: data} }

// MAC is the MAC of the linked device.
func (d Downlink) MAC() string {
	if m := MapStr(d.data, "mac"); m != "" {
		return m
	}
	return MapStr(d.data, "uplinkMac")
}

// Name is the name of the linked device.
func (d Downlink) Name() string { return MapStr(d.data, "name") }

// Type is always "ap".
func (d Downlink) Type() string { return "ap" }

// Model is the model name of the linked device.
func (d Downlink) Model() string { return MapStr(d.data, "model") }

// Port is the linked port number.
func (d Downlink) Port() int { return MapInt(d.data, "port") }

// SwitchPortStatus is the live status of a switch port.
type SwitchPortStatus struct {
	data map[string]any
}

// NewSwitchPortStatus builds a SwitchPortStatus from a raw API object.
func NewSwitchPortStatus(data map[string]any) SwitchPortStatus {
	return SwitchPortStatus{data: data}
}

// LinkStatus is the port link status.
func (s SwitchPortStatus) LinkStatus() LinkStatus {
	return LinkStatus(MapInt(s.data, "linkStatus"))
}

// LinkSpeed is the port link speed.
func (s SwitchPortStatus) LinkSpeed() LinkSpeed {
	return LinkSpeed(MapInt(s.data, "linkSpeed"))
}

// PoEActive reports whether the port is powering a PoE device.
func (s SwitchPortStatus) PoEActive() bool { return MapBool(s.data, "poe") }

// PoEPower is the power (W) supplied over PoE, nil when unreported.
func (s SwitchPortStatus) PoEPower() *float64 { return MapPtrFloat(s.data, "poePower") }

// BytesTX is the number of bytes transmitted by the port.
func (s SwitchPortStatus) BytesTX() int { return MapInt(s.data, "tx") }

// BytesRX is the number of bytes received by the port.
func (s SwitchPortStatus) BytesRX() int { return MapInt(s.data, "rx") }

// STPDiscarding reports the port's STP blocking state.
func (s SwitchPortStatus) STPDiscarding() bool { return MapBool(s.data, "stpDiscarding") }

// SwitchPort is a port on a switch.
type SwitchPort struct {
	data map[string]any
}

// NewSwitchPort builds a SwitchPort from a raw API object.
func NewSwitchPort(data map[string]any) SwitchPort { return SwitchPort{data: data} }

// Port is the port number.
func (p SwitchPort) Port() int { return MapInt(p.data, "port") }

// Name is the port name.
func (p SwitchPort) Name() string { return MapStr(p.data, "name") }

// ProfileID is the ID of the port's config profile.
func (p SwitchPort) ProfileID() string { return MapStr(p.data, "profileId") }

// Type is the port type.
func (p SwitchPort) Type() PortType { return PortType(MapInt(p.data, "type")) }

// Operation is the port operation: switching, mirroring, or aggregating.
func (p SwitchPort) Operation() string { return MapStr(p.data, "operation") }

// IsDisabled reports whether the port is disabled.
func (p SwitchPort) IsDisabled() bool { return MapBool(p.data, "disable") }

// PortStatus is the live status of the port.
func (p SwitchPort) PortStatus() SwitchPortStatus {
	m, _ := MapMap(p.data, "portStatus")
	return NewSwitchPortStatus(m)
}

// SwitchDeviceCaps is the capabilities of a switch.
type SwitchDeviceCaps struct {
	data map[string]any
}

// NewSwitchDeviceCaps builds a SwitchDeviceCaps from a raw API object.
func NewSwitchDeviceCaps(data map[string]any) SwitchDeviceCaps {
	return SwitchDeviceCaps{data: data}
}

// PoEPorts is the number of PoE ports supported.
func (c SwitchDeviceCaps) PoEPorts() int { return MapInt(c.data, "poePortNum") }

// SupportsPoE reports whether PoE is supported.
func (c SwitchDeviceCaps) SupportsPoE() bool { return MapBool(c.data, "poeSupport") }

// SupportsBT reports whether BT is supported.
func (c SwitchDeviceCaps) SupportsBT() bool { return MapBool(c.data, "supportBt") }

// Switch is the details of a managed switch.
type Switch struct {
	Device
}

// NewSwitch builds a Switch from a raw API object.
func NewSwitch(data map[string]any) Switch {
	return Switch{Device: NewDevice(data)}
}

// NumberPorts is the number of ports on the switch.
func (s Switch) NumberPorts() int {
	if n := MapPtrInt(s.data, "portNum"); n != nil {
		return *n
	}
	misc, _ := MapMap(s.data, "deviceMisc")
	return MapInt(misc, "portNum")
}

// Ports are the ports on the switch.
func (s Switch) Ports() []SwitchPort {
	out := make([]SwitchPort, 0, len(MapList(s.data, "ports")))
	for _, p := range MapList(s.data, "ports") {
		m, ok := p.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, NewSwitchPort(m))
	}
	return out
}

// Uplink is the switch's uplink device, nil when absent.
func (s Switch) Uplink() *Link {
	v, present := s.data["uplink"]
	if !present || v == nil {
		return nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	up := NewLink(m)
	return &up
}

// Downlinks are the devices attached downstream of the switch.
func (s Switch) Downlinks() []Downlink {
	items := MapList(s.data, "downlinkList")
	out := make([]Downlink, 0, len(items))
	for _, d := range items {
		m, ok := d.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, NewDownlink(m))
	}
	return out
}

// DeviceCapabilities are the switch capabilities.
func (s Switch) DeviceCapabilities() SwitchDeviceCaps {
	m, _ := MapMap(s.data, "devCap")
	return NewSwitchDeviceCaps(m)
}

// PortDetails is the full configuration of a switch port.
type PortDetails struct {
	data map[string]any
}

// NewPortDetails builds a PortDetails from a raw API object.
func NewPortDetails(data map[string]any) PortDetails { return PortDetails{data: data} }

// PortID is the ID of the port.
func (p PortDetails) PortID() string { return MapStr(p.data, "id") }

// MaxSpeed is the maximum speed of the port.
func (p PortDetails) MaxSpeed() LinkSpeed {
	return LinkSpeed(MapInt(p.data, "maxSpeed"))
}

// LinkSpeed is the current link speed of the port.
func (p PortDetails) LinkSpeed() LinkSpeed {
	return LinkSpeed(MapInt(p.data, "linkSpeed"))
}

// Duplex is the link duplex state of the port.
func (p PortDetails) Duplex() LinkDuplex { return LinkDuplex(MapInt(p.data, "duplex")) }

// ProfileName is the name of the port's config profile.
func (p PortDetails) ProfileName() string { return MapStr(p.data, "profileName") }

// HasProfileOverride reports whether the port's config profile is overridden.
func (p PortDetails) HasProfileOverride() bool {
	return MapBool(p.data, "profileOverrideEnable")
}

// SupportsPoE reports whether the port supports PoE. Older controllers do
// not report the field, so it defaults to true (upstream behavior).
func (p PortDetails) SupportsPoE() bool {
	if v := MapPtrBool(p.data, "supportPoe"); v != nil {
		return *v
	}
	return true
}

// PoEMode is the PoE mode of the port. Omada may claim "enabled" on
// non-PoE ports (port profiles), so the mode is forced to NONE there.
func (p PortDetails) PoEMode() PoEMode {
	if !p.SupportsPoE() {
		return PoEModeNone
	}
	if v := MapPtrInt(p.data, "poe"); v != nil {
		return PoEMode(*v)
	}
	return PoEModeNone
}

// BandwidthLimitMode is the type of bandwidth control applied.
func (p PortDetails) BandwidthLimitMode() BandwidthControl {
	return BandwidthControl(MapInt(p.data, "bandWidthCtrlType"))
}

// Eth8021xControl is the 802.1x auth mode.
func (p PortDetails) Eth8021xControl() Eth802Dot1X {
	if v := MapPtrInt(p.data, "dot1x"); v != nil {
		return Eth802Dot1X(*v)
	}
	return Eth802Dot1XUnknown
}

// LLDPMedEnabled reports whether LLDP-MED is enabled.
func (p PortDetails) LLDPMedEnabled() bool { return MapBool(p.data, "lldpMedEnable") }

// TopologyNotifyEnabled reports whether topology notify is enabled.
func (p PortDetails) TopologyNotifyEnabled() bool {
	return MapBool(p.data, "topoNotifyEnable")
}

// SpanningTreeEnabled reports whether spanning tree is enabled.
func (p PortDetails) SpanningTreeEnabled() bool {
	return MapBool(p.data, "spanningTreeEnable")
}

// LoopbackDetectEnabled reports whether loopback detection is enabled.
func (p PortDetails) LoopbackDetectEnabled() bool {
	return MapBool(p.data, "loopbackDetectEnable")
}

// PortIsolationEnabled reports whether port isolation is enabled.
func (p PortDetails) PortIsolationEnabled() bool {
	return MapBool(p.data, "portIsolationEnable")
}

// LoopbackDetectVlanBasedEnabled reports whether VLAN-based loopback
// detection is enabled.
func (p PortDetails) LoopbackDetectVlanBasedEnabled() bool {
	return MapBool(p.data, "loopbackDetectVlanBasedEnable")
}

// HasLoopbackDetectVlanBased reports whether VLAN-based loopback detection
// settings are available for this port.
func (p PortDetails) HasLoopbackDetectVlanBased() bool {
	_, ok := p.data["loopbackDetectVlanBasedEnable"]
	return ok
}

// NativeNetworkID is the ID of the port's native network.
func (p PortDetails) NativeNetworkID() *string {
	return MapPtrStr(p.data, "nativeNetworkId")
}

// TagIDs are the IDs of the port's tagged networks.
func (p PortDetails) TagIDs() []string { return MapStrList(p.data, "tagIds") }

// NetworkTagsSetting is the network tags setting for the port.
func (p PortDetails) NetworkTagsSetting() NetworkTagsSetting {
	if v := MapPtrInt(p.data, "networkTagsSetting"); v != nil {
		return NetworkTagsSetting(*v)
	}
	return NetworkTagsSettingUnknown
}

// TaggedNetworkIDs are the IDs of the port's tagged networks.
func (p PortDetails) TaggedNetworkIDs() []string {
	return MapStrList(p.data, "tagNetworkIds")
}

// UntaggedNetworkIDs are the IDs of the port's untagged networks.
func (p PortDetails) UntaggedNetworkIDs() []string {
	return MapStrList(p.data, "untagNetworkIds")
}

// VoiceNetworkEnabled reports whether voice network is enabled for the port.
func (p PortDetails) VoiceNetworkEnabled() bool {
	return MapBool(p.data, "voiceNetworkEnable")
}

// HasVoiceNetwork reports whether voice network settings are available for
// this port.
func (p PortDetails) HasVoiceNetwork() bool {
	_, ok := p.data["voiceNetworkEnable"]
	return ok
}

// VoiceNetworkID is the ID of the port's voice network.
func (p PortDetails) VoiceNetworkID() *string {
	return MapPtrStr(p.data, "voiceNetworkId")
}

// FlowControlEnabled reports whether flow control is enabled.
func (p PortDetails) FlowControlEnabled() bool {
	return MapBool(p.data, "flowControlEnable")
}

// HasFlowControl reports whether flow control settings are available for
// this port.
func (p PortDetails) HasFlowControl() bool {
	_, ok := p.data["flowControlEnable"]
	return ok
}

// EEEEnabled reports whether energy-efficient Ethernet is enabled.
func (p PortDetails) EEEEnabled() bool { return MapBool(p.data, "eeeEnable") }

// HasEEE reports whether EEE settings are available for this port.
func (p PortDetails) HasEEE() bool {
	_, ok := p.data["eeeEnable"]
	return ok
}

// PortProfile is a switch port configuration profile.
type PortProfile struct {
	data map[string]any
}

// NewPortProfile builds a PortProfile from a raw API object.
func NewPortProfile(data map[string]any) PortProfile { return PortProfile{data: data} }

// ProfileID is the ID of the profile.
func (p PortProfile) ProfileID() string { return MapStr(p.data, "id") }

// Site is the site the profile is valid for.
func (p PortProfile) Site() string { return MapStr(p.data, "site") }

// Name is the name of the profile.
func (p PortProfile) Name() string { return MapStr(p.data, "name") }

// PoEMode is the PoE mode.
func (p PortProfile) PoEMode() PoEMode {
	if v := MapPtrInt(p.data, "poe"); v != nil {
		return PoEMode(*v)
	}
	return PoEModeNone
}

// BandwidthLimitMode is the type of bandwidth control applied.
func (p PortProfile) BandwidthLimitMode() BandwidthControl {
	return BandwidthControl(MapInt(p.data, "bandWidthCtrlType"))
}

// Eth8021xControl is the 802.1x auth mode.
func (p PortProfile) Eth8021xControl() Eth802Dot1X {
	if v := MapPtrInt(p.data, "dot1x"); v != nil {
		return Eth802Dot1X(*v)
	}
	return Eth802Dot1XUnknown
}

// LLDPMedEnabled reports whether LLDP-MED is enabled.
func (p PortProfile) LLDPMedEnabled() bool { return MapBool(p.data, "lldpMedEnable") }

// TopologyNotifyEnabled reports whether topology notify is enabled.
func (p PortProfile) TopologyNotifyEnabled() bool {
	return MapBool(p.data, "topoNotifyEnable")
}

// SpanningTreeEnabled reports whether spanning tree is enabled.
func (p PortProfile) SpanningTreeEnabled() bool {
	return MapBool(p.data, "spanningTreeEnable")
}

// LoopbackDetectEnabled reports whether loopback detection is enabled.
func (p PortProfile) LoopbackDetectEnabled() bool {
	return MapBool(p.data, "loopbackDetectEnable")
}

// PortIsolationEnabled reports whether port isolation is enabled.
func (p PortProfile) PortIsolationEnabled() bool {
	return MapBool(p.data, "portIsolationEnable")
}

// LoopbackDetectVlanBasedEnabled reports whether VLAN-based loopback
// detection is enabled.
func (p PortProfile) LoopbackDetectVlanBasedEnabled() bool {
	return MapBool(p.data, "loopbackDetectVlanBasedEnable")
}

// HasLoopbackDetectVlanBased reports whether VLAN-based loopback detection
// settings are available for this profile.
func (p PortProfile) HasLoopbackDetectVlanBased() bool {
	_, ok := p.data["loopbackDetectVlanBasedEnable"]
	return ok
}

// FlowControlEnabled reports whether flow control is enabled.
func (p PortProfile) FlowControlEnabled() bool {
	return MapBool(p.data, "flowControlEnable")
}

// HasFlowControl reports whether flow control settings are available for
// this profile.
func (p PortProfile) HasFlowControl() bool {
	_, ok := p.data["flowControlEnable"]
	return ok
}

// EEEEnabled reports whether energy-efficient Ethernet is enabled.
func (p PortProfile) EEEEnabled() bool { return MapBool(p.data, "eeeEnable") }

// HasEEE reports whether EEE settings are available for this profile.
func (p PortProfile) HasEEE() bool {
	_, ok := p.data["eeeEnable"]
	return ok
}

// APLANPortSettings is a LAN port on an access point.
type APLANPortSettings struct {
	data map[string]any
}

// NewAPLANPortSettings builds an APLANPortSettings from a raw API object.
func NewAPLANPortSettings(data map[string]any) APLANPortSettings {
	return APLANPortSettings{data: data}
}

// PortName is the port name (not editable).
func (p APLANPortSettings) PortName() string {
	if n := MapStr(p.data, "lanPort"); n != "" {
		return n
	}
	return "LAN Port"
}

// SupportsVLAN reports whether the port supports VLAN tagging.
func (p APLANPortSettings) SupportsVLAN() bool { return MapBool(p.data, "supportVlan") }

// LocalVLANEnable reports whether VLAN tagging is enabled for the port.
func (p APLANPortSettings) LocalVLANEnable() bool {
	return MapBool(p.data, "localVlanEnable")
}

// LocalVLANID is the VLAN ID of the port.
func (p APLANPortSettings) LocalVLANID() int {
	if v := MapPtrInt(p.data, "localVlanId"); v != nil {
		return *v
	}
	return -1
}

// SupportsPoE reports whether the port supports PoE output.
func (p APLANPortSettings) SupportsPoE() bool { return MapBool(p.data, "supportPoe") }

// PoEEnable is the PoE state of the port.
func (p APLANPortSettings) PoEEnable() bool {
	return MapBool(p.data, "poeOutEnable")
}

// APRadioSettings is the configured state of one radio on an access point.
type APRadioSettings struct {
	data map[string]any
}

// NewAPRadioSettings builds an APRadioSettings from a raw API object.
func NewAPRadioSettings(data map[string]any) APRadioSettings {
	return APRadioSettings{data: data}
}

// Raw is the underlying API object.
func (s APRadioSettings) Raw() map[string]any { return s.data }

// RadioEnabled reports whether the radio is switched on.
func (s APRadioSettings) RadioEnabled() bool {
	return MapBool(s.data, "radioEnable")
}

// ChannelWidth is the configured channel width.
func (s APRadioSettings) ChannelWidth() ChannelWidth {
	if v := MapPtrInt(s.data, "channelWidth"); v != nil {
		return ChannelWidth(*v)
	}
	return ChannelWidthUnknown
}

// ChannelIndex is the configured channel as a 1-based index into the
// radio's channel list. 0 means the AP picks the channel automatically.
func (s APRadioSettings) ChannelIndex() int { return MapInt(s.data, "channel") }

// AutoChannel reports whether the AP chooses the channel itself.
func (s APRadioSettings) AutoChannel() bool { return s.ChannelIndex() == 0 }

// Frequency is the centre frequency of the configured channel in MHz.
func (s APRadioSettings) Frequency() int { return MapInt(s.data, "freq") }

// WirelessMode is the 802.11 mode of the radio. -2 means automatic.
func (s APRadioSettings) WirelessMode() int {
	if v := MapPtrInt(s.data, "wirelessMode"); v != nil {
		return *v
	}
	return -2
}

// TxPower is the transmit power in dBm.
func (s APRadioSettings) TxPower() int { return MapInt(s.data, "txPower") }

// TxPowerLevel is the transmit power preset (0=Low, 1=Medium, 2=High,
// 3=Custom, 4=Auto).
func (s APRadioSettings) TxPowerLevel() int { return MapInt(s.data, "txPowerLevel") }

// APRadioStatus is the live state of one radio on an access point.
type APRadioStatus struct {
	data map[string]any
}

// NewAPRadioStatus builds an APRadioStatus from a raw API object.
func NewAPRadioStatus(data map[string]any) APRadioStatus {
	return APRadioStatus{data: data}
}

// ActualChannel is the channel currently in use, e.g. "36 / 5180MHz".
func (s APRadioStatus) ActualChannel() string {
	return MapStr(s.data, "actualChannel")
}

// BandWidth is the channel width currently in use, e.g. "40MHz".
func (s APRadioStatus) BandWidth() string { return MapStr(s.data, "bandWidth") }

// TxPower is the transmit power currently in use in dBm.
func (s APRadioStatus) TxPower() int { return MapInt(s.data, "txPower") }

// MaxTxRate is the maximum PHY rate at the current width in Mbps.
func (s APRadioStatus) MaxTxRate() int { return MapInt(s.data, "maxTxRate") }

// RXUtil is the percentage of airtime spent receiving.
func (s APRadioStatus) RXUtil() int { return MapInt(s.data, "rxUtil") }

// TXUtil is the percentage of airtime spent transmitting.
func (s APRadioStatus) TXUtil() int { return MapInt(s.data, "txUtil") }

// InterUtil is the percentage of airtime lost to interference.
func (s APRadioStatus) InterUtil() int { return MapInt(s.data, "interUtil") }

// WirelessMode is the 802.11 mode in use, e.g. "a/n/ac mixed".
func (s APRadioStatus) WirelessMode() string { return MapStr(s.data, "rdMode") }

// AccessPoint is the details of a managed access point.
type AccessPoint struct {
	Device
}

// NewAccessPoint builds an AccessPoint from a raw API object.
func NewAccessPoint(data map[string]any) AccessPoint {
	return AccessPoint{Device: NewDevice(data)}
}

// WirelessLinked reports whether the AP is connected wirelessly.
func (a AccessPoint) WirelessLinked() bool {
	return MapBool(a.data, "wirelessLinked")
}

func (a AccessPoint) miscBool(key string) bool {
	misc, ok := MapMap(a.data, "deviceMisc")
	if !ok {
		return false
	}
	return MapBool(misc, key)
}

// Supports5G reports whether 5 GHz wifi is supported.
func (a AccessPoint) Supports5G() bool { return a.miscBool("support5g") }

// Supports5G2 reports whether 5G2 wifi is supported.
func (a AccessPoint) Supports5G2() bool { return a.miscBool("support5g2") }

// Supports6G reports whether Wi-Fi 6 is supported.
func (a AccessPoint) Supports6G() bool { return a.miscBool("support6g") }

// Supports11AC reports whether 802.11ac is supported.
func (a AccessPoint) Supports11AC() bool { return a.miscBool("support11ac") }

// SupportsMesh reports whether mesh networking is supported.
func (a AccessPoint) SupportsMesh() bool { return a.miscBool("supportMesh") }

// LANPortSettings are the settings for the AP's LAN ports.
func (a AccessPoint) LANPortSettings() []APLANPortSettings {
	items := MapList(a.data, "lanPortSettings")
	out := make([]APLANPortSettings, 0, len(items))
	for _, p := range items {
		m, ok := p.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, NewAPLANPortSettings(m))
	}
	return out
}

// WiredUplink is the AP's wired uplink device, nil when absent.
func (a AccessPoint) WiredUplink() *Link {
	m, present := MapMap(a.data, "wiredUplink")
	if !present {
		return nil
	}
	up := NewLink(m)
	return &up
}

// RadioSettings returns the configured state of one radio, or nil if the AP
// has no such radio.
func (a AccessPoint) RadioSettings(band RadioId) *APRadioSettings {
	m, present := MapMap(a.data, band.SettingsKey())
	if !present {
		return nil
	}
	s := NewAPRadioSettings(m)
	return &s
}

// RadioStatus returns the live state of one radio, or nil if the AP has no
// such radio.
func (a AccessPoint) RadioStatus(band RadioId) *APRadioStatus {
	m, present := MapMap(a.data, band.StatusKey())
	if !present {
		return nil
	}
	s := NewAPRadioStatus(m)
	return &s
}

// RadioBands are the radios this AP actually has.
func (a AccessPoint) RadioBands() []RadioId {
	var out []RadioId
	for _, band := range []RadioId{RadioIdFreq24, RadioIdFreq51, RadioIdFreq52, RadioIdFreq6} {
		if _, ok := a.data[band.SettingsKey()]; ok {
			out = append(out, band)
		}
	}
	return out
}

// GatewayPortStatus is the live status of a gateway port.
type GatewayPortStatus struct {
	data map[string]any
}

// NewGatewayPortStatus builds a GatewayPortStatus from a raw API object.
func NewGatewayPortStatus(data map[string]any) GatewayPortStatus {
	return GatewayPortStatus{data: data}
}

// PortNumber is the port number.
func (p GatewayPortStatus) PortNumber() int { return MapInt(p.data, "port") }

// Name is the port name.
func (p GatewayPortStatus) Name() string { return MapStr(p.data, "name") }

// DisplayName is the port display name, falling back to Name.
func (p GatewayPortStatus) DisplayName() string {
	if d := MapStr(p.data, "portDesc"); d != "" {
		return d
	}
	return p.Name()
}

// Type is the port type: SFP, WAN, WAN/LAN, or LAN.
func (p GatewayPortStatus) Type() GatewayPortType {
	return GatewayPortType(MapInt(p.data, "type"))
}

// Mode reports whether the port operates in WAN or LAN mode.
func (p GatewayPortStatus) Mode() GatewayPortMode {
	return GatewayPortMode(MapInt(p.data, "mode"))
}

// LinkStatus is the low-level connectivity status of the link.
func (p GatewayPortStatus) LinkStatus() LinkStatus {
	return LinkStatus(MapInt(p.data, "status"))
}

// BytesTX is the number of bytes transmitted by the port.
func (p GatewayPortStatus) BytesTX() int { return MapInt(p.data, "tx") }

// BytesRX is the number of bytes received by the port.
func (p GatewayPortStatus) BytesRX() int { return MapInt(p.data, "rx") }

// PoEActive reports whether the port is powering a PoE device.
func (p GatewayPortStatus) PoEActive() bool { return MapInt(p.data, "poe") != 0 }

// WANConnected reports whether the port is connected to the internet.
func (p GatewayPortStatus) WANConnected() bool {
	return MapInt(p.data, "internetState") != 0
}

// IPv6WANConnected reports whether the port is connected to the internet
// over IPv6.
func (p GatewayPortStatus) IPv6WANConnected() bool {
	cfg, _ := MapMap(p.data, "wanPortIpv6Config")
	return MapInt(cfg, "internetState") != 0
}

// OnlineDetection reports whether regular internet ping tests are working.
func (p GatewayPortStatus) OnlineDetection() bool {
	return (p.WANConnected() || p.IPv6WANConnected()) && MapInt(p.data, "onlineDetection") != 0
}

// WANIPAddress is the WAN IPv4 address of the port.
func (p GatewayPortStatus) WANIPAddress() *string {
	return MapPtrStr(p.data, "ip")
}

// WANIPv6Enabled reports whether the WAN IPv6 address is enabled.
func (p GatewayPortStatus) WANIPv6Enabled() bool {
	cfg, _ := MapMap(p.data, "wanPortIpv6Config")
	return MapInt(cfg, "enable") != 0
}

// WANIPv6Address is the WAN IPv6 address of the port.
func (p GatewayPortStatus) WANIPv6Address() *string {
	cfg, _ := MapMap(p.data, "wanPortIpv6Config")
	return MapPtrStr(cfg, "addr")
}

// LinkSpeed is the established link speed of the port.
func (p GatewayPortStatus) LinkSpeed() LinkSpeed {
	if v := MapPtrInt(p.data, "speed"); v != nil {
		return LinkSpeed(*v)
	}
	return LinkSpeedSpeed10
}

// LinkDuplex is the actual duplex mode of the port.
func (p GatewayPortStatus) LinkDuplex() LinkDuplex {
	if v := MapPtrInt(p.data, "duplex"); v != nil {
		return LinkDuplex(*v)
	}
	return LinkDuplexFull
}

// WANProtocol is the WAN protocol: static, dhcp, pppoe, l2tp, or pptp.
func (p GatewayPortStatus) WANProtocol() *string {
	return MapPtrStr(p.data, "proto")
}

// GatewayPortConfig is the configuration of a gateway port, including its
// status.
type GatewayPortConfig struct {
	data       map[string]any
	poeEnabled *bool
}

// NewGatewayPortConfig builds a GatewayPortConfig from a raw API object.
// poeEnabled is the resolved PoE state for the port's number (nil when the
// gateway has no PoE settings for it).
func NewGatewayPortConfig(data map[string]any, poeEnabled *bool) GatewayPortConfig {
	return GatewayPortConfig{data: data, poeEnabled: poeEnabled}
}

// PortNumber is the port number.
func (p GatewayPortConfig) PortNumber() int { return MapInt(p.data, "port") }

// Duplex is the configured duplex mode of the port.
func (p GatewayPortConfig) Duplex() LinkDuplex {
	if v := MapPtrInt(p.data, "duplex"); v != nil {
		return LinkDuplex(*v)
	}
	return LinkDuplexAuto
}

// LinkSpeed is the configured link speed of the port.
func (p GatewayPortConfig) LinkSpeed() LinkSpeed {
	if v := MapPtrInt(p.data, "linkSpeed"); v != nil {
		return LinkSpeed(*v)
	}
	return LinkSpeedSpeedAuto
}

// MirrorEnable reports whether port mirroring is enabled.
func (p GatewayPortConfig) MirrorEnable() bool { return MapBool(p.data, "mirrorEnable") }

// PortStatus is the full status of the port.
func (p GatewayPortConfig) PortStatus() GatewayPortStatus {
	m, _ := MapMap(p.data, "portStat")
	return NewGatewayPortStatus(m)
}

// PoEMode is the PoE mode of the port, resolved from the gateway's PoE
// settings.
func (p GatewayPortConfig) PoEMode() PoEMode {
	switch {
	case p.poeEnabled == nil:
		return PoEModeNone
	case *p.poeEnabled:
		return PoEModeEnabled
	default:
		return PoEModeDisabled
	}
}

// Gateway is the details of a managed gateway.
type Gateway struct {
	Device
}

// NewGateway builds a Gateway from a raw API object.
func NewGateway(data map[string]any) Gateway {
	return Gateway{Device: NewDevice(data)}
}

// NumberPorts is the number of ports on the gateway.
func (g Gateway) NumberPorts() int { return MapInt(g.data, "portNum") }

// SupportsPoE reports whether the device supports PoE.
func (g Gateway) SupportsPoE() bool { return MapBool(g.data, "supportPoe") }

// IP is the gateway's LAN IP address.
func (g Gateway) IP() string { return MapStr(g.data, "ip") }

// PortStatuses are the live statuses of the gateway's ports.
func (g Gateway) PortStatuses() []GatewayPortStatus {
	items := MapList(g.data, "portStats")
	out := make([]GatewayPortStatus, 0, len(items))
	for _, p := range items {
		m, ok := p.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, NewGatewayPortStatus(m))
	}
	return out
}

// PortConfigs are the configurations of the gateway's ports, each including
// its status.
func (g Gateway) PortConfigs() []GatewayPortConfig {
	poeData := map[int]bool{}
	if g.SupportsPoE() {
		// Combined gateway+PoE switches carry this extra data.
		for _, entry := range MapList(g.data, "poeSettings") {
			m, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			poeData[MapInt(m, "portId")] = MapBool(m, "enable")
		}
	}
	items := MapList(g.data, "portConfigs")
	out := make([]GatewayPortConfig, 0, len(items))
	for _, entry := range items {
		m, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		var poe *bool
		if enabled, exists := poeData[MapInt(m, "port")]; exists {
			poe = &enabled
		}
		out = append(out, NewGatewayPortConfig(m, poe))
	}
	return out
}

// LLDPEnabled reports whether LLDP is enabled on the gateway.
func (g Gateway) LLDPEnabled() bool { return MapBool(g.data, "lldpEnable") }

// EchoServer is the address of the server pinged for online detection.
func (g Gateway) EchoServer() *string { return MapPtrStr(g.data, "echoServer") }

// IsCombinedGateway reports whether this is a combined gateway/switch.
func (g Gateway) IsCombinedGateway() bool {
	return MapBool(g.data, "combinedGateway")
}
