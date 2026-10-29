package controller

// This file ports the network model classes from upstream
// tplink_omada_client networks.py. The upstream placeholder classes
// (LanProfile, GatewayAcl, SwitchAcl, EapAcl, GroupProfile,
// IpMacBinding) carry no properties and are omitted.

// DhcpReservation is a DHCP reservation on an Omada controller.
type DhcpReservation struct {
	data map[string]any
}

// NewDhcpReservation builds a DhcpReservation from a raw API object.
func NewDhcpReservation(data map[string]any) DhcpReservation {
	return DhcpReservation{data: data}
}

// ID is the reservation ID.
func (d DhcpReservation) ID() string { return mapStr(d.data, "id") }

// MAC is the device MAC address.
func (d DhcpReservation) MAC() string { return mapStr(d.data, "mac") }

// IP is the reserved IP address.
func (d DhcpReservation) IP() string { return mapStr(d.data, "ip") }

// Description is the description/name for the reservation.
func (d DhcpReservation) Description() *string {
	return mapPtrStr(d.data, "description")
}

// Status reports whether the reservation is active.
func (d DhcpReservation) Status() *bool { return mapPtrBool(d.data, "status") }

// NetID is the ID of the network this reservation belongs to.
func (d DhcpReservation) NetID() *string { return mapPtrStr(d.data, "netId") }

// NetName is the name of the network this reservation belongs to.
func (d DhcpReservation) NetName() *string { return mapPtrStr(d.data, "netName") }

// ClientName is the name of the client associated with this reservation.
func (d DhcpReservation) ClientName() *string {
	return mapPtrStr(d.data, "clientName")
}

// ExportToIpMacBinding reports whether the reservation is exported to
// IP-MAC binding.
func (d DhcpReservation) ExportToIpMacBinding() *bool {
	return mapPtrBool(d.data, "exportToIpMacBinding")
}

// LanNetwork is a LAN network configuration.
type LanNetwork struct {
	data map[string]any
}

// NewLanNetwork builds a LanNetwork from a raw API object.
func NewLanNetwork(data map[string]any) LanNetwork {
	return LanNetwork{data: data}
}

// Raw returns the raw API object backing this network.
func (n LanNetwork) Raw() map[string]any { return n.data }

// ID is the controller ID of the LAN network.
func (n LanNetwork) ID() string { return mapStr(n.data, "id") }

// Name is the LAN network name.
func (n LanNetwork) Name() string { return mapStr(n.data, "name") }

// Purpose is the controller network purpose.
func (n LanNetwork) Purpose() string { return mapStr(n.data, "purpose") }

// VLANID is the VLAN ID, when configured.
func (n LanNetwork) VLANID() *int { return mapPtrInt(n.data, "vlan") }

// GatewaySubnet is the gateway IPv4 subnet in CIDR notation.
func (n LanNetwork) GatewaySubnet() *string {
	return mapPtrStr(n.data, "gatewaySubnet")
}

// DHCPEnabled reports whether DHCP is enabled for the network, nil when
// not reported.
func (n LanNetwork) DHCPEnabled() *bool {
	settings, present := mapMap(n.data, "dhcpSettings")
	if !present {
		return nil
	}
	return mapPtrBool(settings, "enable")
}
