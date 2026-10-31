package models

// This file ports the VPN policy models from upstream tplink_omada_client
// vpn.py.

// VpnCategory is the VPN tab a policy belongs to.
type VpnCategory string

const (
	VpnServer     VpnCategory = "server"
	VpnClient     VpnCategory = "client"
	VpnSiteToSite VpnCategory = "site_to_site"
)

func (c VpnCategory) String() string { return string(c) }

// listEndpoints are the OpenAPI list endpoints per VPN tab, relative to
// .../sites/{site}/vpn/.
var listEndpoints = map[VpnCategory]string{
	VpnServer:     "client-to-site-vpn-servers",
	VpnClient:     "client-to-site-vpn-clients",
	VpnSiteToSite: "site-to-site-vpns",
}

// ListEndpoint is the OpenAPI list endpoint for this tab.
func (c VpnCategory) ListEndpoint() string { return listEndpoints[c] }

type VpnType int32

const (
	VpnTypeUnknown   VpnType = -1
	VpnTypeL2TP      VpnType = 0
	VpnTypePPTP      VpnType = 1
	VpnTypeIPSec     VpnType = 2
	VpnTypeOpenVPN   VpnType = 3
	VpnTypeWireGuard VpnType = 4
	VpnTypeSSL       VpnType = 5
)

var vpnTypeNames = map[VpnType]string{
	VpnTypeUnknown:   "unknown",
	VpnTypeL2TP:      "l2tp",
	VpnTypePPTP:      "pptp",
	VpnTypeIPSec:     "ipsec",
	VpnTypeOpenVPN:   "openvpn",
	VpnTypeWireGuard: "wireguard",
	VpnTypeSSL:       "ssl",
}

func (t VpnType) String() string {
	if n, ok := vpnTypeNames[t]; ok {
		return n
	}
	return "unknown"
}

// TypeName is the human-readable protocol, falling back to the member name.
func (t VpnType) TypeName() string {
	switch t {
	case VpnTypeIPSec:
		return "IPsec"
	case VpnTypeOpenVPN:
		return "OpenVPN"
	case VpnTypeWireGuard:
		return "WireGuard"
	default:
		return t.String()
	}
}

// VpnPolicy is a single VPN entry that can be enabled/disabled via
// .../vpn/{id}/status.
type VpnPolicy struct {
	data     map[string]any
	category VpnCategory
}

// NewVpnPolicy builds a VpnPolicy from a raw API object and its tab.
func NewVpnPolicy(category VpnCategory, data map[string]any) VpnPolicy {
	return VpnPolicy{data: data, category: category}
}

// Category is the VPN tab this policy belongs to.
func (p VpnPolicy) Category() VpnCategory { return p.category }

// PolicyID is the controller-side id, used as {id} in the status PATCH.
func (p VpnPolicy) PolicyID() string { return MapStr(p.data, "id") }

// Name is the user-defined policy name, falling back to the id.
func (p VpnPolicy) Name() string {
	if n := MapStr(p.data, "name"); n != "" {
		return n
	}
	return p.PolicyID()
}

// Enabled is the current enabled state (the "status" flag).
func (p VpnPolicy) Enabled() bool { return MapBool(p.data, "status") }

// VpnType is the VPN protocol type. Unrecognized codes map to Unknown,
// mirroring the upstream _missing_ behavior.
func (p VpnPolicy) VpnType() VpnType {
	v := VpnTypeUnknown
	if n := MapPtrInt(p.data, "vpnType"); n != nil {
		v = VpnType(*n)
	}
	if _, ok := vpnTypeNames[v]; !ok {
		return VpnTypeUnknown
	}
	return v
}

// VpnTypeName is the human-readable protocol.
func (p VpnPolicy) VpnTypeName() string { return p.VpnType().TypeName() }

// UniqueID is the stable key: category + id.
func (p VpnPolicy) UniqueID() string {
	return string(p.category) + "_" + p.PolicyID()
}
