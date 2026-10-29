package controller

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// This file ports the conservative VLAN-only write contract from upstream
// tplink_omada_client vlans.py, verified there on controller 5.15.8.12.

var (
	// computedFields describe identity, binding, or references in the list
	// response; the VLAN-only editor does not send them in its write payload.
	computedFields = []string{
		"id", "site", "interfaceIds", "portal", "accessControlRule",
		"rateLimit", "allLan", "resource", "origName", "interface", "primary",
	}
	writableFields = []string{
		"name", "purpose", "application", "vlan", "dhcpL2RelayEnable",
		"igmpSnoopEnable", "fastLeaveEnable", "mldSnoopEnable", "dhcpGuard",
		"dhcpv6Guard",
	}
	guardServers = map[string][]string{
		"dhcpGuard":   {"dhcpSvr1", "dhcpSvr2"},
		"dhcpv6Guard": {"dhcpv6Svr1", "dhcpv6Svr2"},
	}
	booleanFields = []string{
		"dhcpL2RelayEnable", "igmpSnoopEnable", "fastLeaveEnable", "mldSnoopEnable",
	}
)

func errVLAN(format string, args ...any) error {
	return &ValidationError{Msg: fmt.Sprintf(format, args...)}
}

func validateVLANName(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", errVLAN("VLAN name must contain 1–128 UTF-8 bytes and no control characters")
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return "", errVLAN("VLAN name must contain 1–128 UTF-8 bytes and no control characters")
		}
	}
	if len([]byte(trimmed)) > 128 {
		return "", errVLAN("VLAN name must contain 1–128 UTF-8 bytes and no control characters")
	}
	switch strings.ToLower(trimmed) {
	case "default", "management", "management(default)":
		return "", errVLAN("Default/management VLAN names are protected")
	}
	return trimmed, nil
}

func validateVLANID(value int) (int, error) {
	if value < 1 || value > 4090 {
		return 0, errVLAN("VLAN ID must be an integer from 1 to 4090")
	}
	if value == 1 {
		return 0, errVLAN("VLAN 1 is protected")
	}
	return value, nil
}

// ValidateVlanTarget rejects protected, routed, incomplete, or unverified
// VLAN targets.
func ValidateVlanTarget(network LanNetwork) error {
	raw := network.Raw()
	identity, _ := raw["id"].(string)
	if identity == "" || !isASCII(identity) {
		return errVLAN("Missing or unsafe VLAN network identity")
	}
	for _, c := range identity {
		if !(unicode.IsLetter(c) || unicode.IsNumber(c) || c == '-' || c == '_') {
			return errVLAN("Missing or unsafe VLAN network identity")
		}
	}

	primary, isPrimary := raw["primary"].(bool)
	var application float64
	isApplication := true
	switch a := raw["application"].(type) {
	case float64:
		application = a
	case int:
		application = float64(a)
	default:
		isApplication = false
	}
	if network.Purpose() != "vlan" || !isPrimary || primary || !isApplication || application != 0 {
		return errVLAN("Only non-primary VLAN-only networks with application=0 can be changed")
	}
	if _, err := validateVLANID(valueOf(network.VLANID())); err != nil {
		return err
	}
	if _, err := validateVLANName(network.Name()); err != nil {
		return err
	}

	known := map[string]bool{}
	for _, f := range append(append([]string{}, computedFields...), writableFields...) {
		known[f] = true
	}
	var unknown []string
	for key := range raw {
		if !known[key] {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return errVLAN("Unsupported VLAN fields: %s", strings.Join(unknown, ", "))
	}
	for _, field := range booleanFields {
		if value, present := raw[field]; present && value != nil {
			if _, isBool := value.(bool); !isBool {
				return errVLAN("Invalid VLAN boolean settings")
			}
		}
	}
	for field, servers := range guardServers {
		guardValue, present := raw[field]
		if !present {
			continue
		}
		guard, isMap := guardValue.(map[string]any)
		if !isMap {
			return errVLAN("Invalid VLAN %s: expected a guard with boolean enable", field)
		}
		if _, isBool := guard["enable"].(bool); !isBool {
			return errVLAN("Invalid VLAN %s: expected a guard with boolean enable", field)
		}
		var gUnknown []string
		allowed := append(append([]string{}, servers...), "enable")
		for key := range guard {
			if !contains(allowed, key) {
				gUnknown = append(gUnknown, key)
			}
		}
		if len(gUnknown) > 0 {
			sort.Strings(gUnknown)
			return errVLAN("Unsupported VLAN fields in %s: %s", field, strings.Join(gUnknown, ", "))
		}
		for _, server := range servers {
			if value, ok := guard[server]; ok && value != nil {
				if _, isStr := value.(string); !isStr {
					return errVLAN("Invalid VLAN %s: server values must be strings or null", field)
				}
			}
		}
	}
	return nil
}

// ValidateVlanConflicts rejects collisions against a fresh inventory,
// including routed networks.
func ValidateVlanConflicts(payload map[string]any, networks []LanNetwork, excludeID *string) error {
	vlanID, ok := payloadVLANID(payload)
	if !ok {
		return errVLAN("Invalid VLAN payload identifier")
	}
	nameValue, isStr := payload["name"].(string)
	if !isStr {
		return errVLAN("Invalid VLAN payload identifier")
	}
	name := strings.ToLower(nameValue)
	for _, network := range networks {
		identity, _ := network.Raw()["id"].(string)
		if identity == "" {
			return errVLAN("LAN inventory contains a missing or invalid network identity")
		}
		if excludeID != nil && identity == *excludeID {
			continue
		}
		networkVLAN := network.VLANID()
		if (networkVLAN != nil && *networkVLAN == vlanID) || strings.ToLower(network.Name()) == name {
			return errVLAN("VLAN ID or name already exists")
		}
	}
	return nil
}

// ResolveVlanTarget resolves an exact ID/name, or strictly an ID after an
// earlier selection.
func ResolveVlanTarget(target string, networks []LanNetwork, byID bool) (LanNetwork, error) {
	var matches []LanNetwork
	for _, n := range networks {
		id, _ := n.Raw()["id"].(string)
		if id == target || (!byID && n.Name() == target) {
			matches = append(matches, n)
		}
	}
	if len(matches) != 1 {
		return LanNetwork{}, errVLAN("VLAN target is missing or ambiguous; use an exact network ID")
	}
	identity, _ := matches[0].Raw()["id"].(string)
	count := 0
	for _, n := range networks {
		if id, _ := n.Raw()["id"].(string); id == identity {
			count++
		}
	}
	if count != 1 {
		return LanNetwork{}, errVLAN("VLAN target identity is ambiguous; inspect controller inventory")
	}
	return matches[0], nil
}

// VlanCreate creates a single VLAN-only network; it does not configure
// routing or DHCP.
type VlanCreate struct {
	name   string
	vlanID int
}

// NewVlanCreate builds a VlanCreate, validating name and VLAN ID.
func NewVlanCreate(name string, vlanID int) (VlanCreate, error) {
	trimmed, err := validateVLANName(name)
	if err != nil {
		return VlanCreate{}, err
	}
	id, err := validateVLANID(vlanID)
	if err != nil {
		return VlanCreate{}, err
	}
	return VlanCreate{name: trimmed, vlanID: id}, nil
}

// Payload is the range-string create payload used by the installed editor.
func (v VlanCreate) Payload() map[string]any {
	return map[string]any{
		"application":       0,
		"name":              v.name,
		"purpose":           "vlan",
		"vlans":             strconv.Itoa(v.vlanID),
		"dhcpL2RelayEnable": false,
		"igmpSnoopEnable":   false,
		"mldSnoopEnable":    false,
		"dhcpGuard": map[string]any{
			"enable": false, "dhcpSvr1": nil, "dhcpSvr2": nil,
		},
		"dhcpv6Guard": map[string]any{
			"enable": false, "dhcpv6Svr1": nil, "dhcpv6Svr2": nil,
		},
	}
}

// VlanUpdate changes the name and/or VLAN ID of a network while preserving
// its existing writable settings.
type VlanUpdate struct {
	name   *string
	vlanID *int
}

// NewVlanUpdate builds a VlanUpdate. At least one of name/vlanID is
// required.
func NewVlanUpdate(name *string, vlanID *int) (VlanUpdate, error) {
	if name == nil && vlanID == nil {
		return VlanUpdate{}, errVLAN("At least one VLAN change is required")
	}
	update := VlanUpdate{}
	if name != nil {
		trimmed, err := validateVLANName(*name)
		if err != nil {
			return VlanUpdate{}, err
		}
		update.name = &trimmed
	}
	if vlanID != nil {
		id, err := validateVLANID(*vlanID)
		if err != nil {
			return VlanUpdate{}, err
		}
		update.vlanID = &id
	}
	return update, nil
}

// Payload merges the verified writable settings of network into a
// defensive copy, applying the requested changes.
func (v VlanUpdate) Payload(network LanNetwork) (map[string]any, error) {
	if err := ValidateVlanTarget(network); err != nil {
		return nil, err
	}
	payload := map[string]any{}
	for key, value := range network.Raw() {
		if contains(computedFields, key) {
			continue
		}
		payload[key] = cloneValue(value)
	}
	if v.name != nil {
		payload["name"] = *v.name
	}
	if v.vlanID != nil {
		payload["vlan"] = *v.vlanID
	}
	return payload, nil
}

func payloadVLANID(payload map[string]any) (int, bool) {
	switch value := payload["vlan"].(type) {
	case float64:
		return int(value), true
	case int:
		return value, true
	}
	if s, ok := payload["vlans"].(string); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
			return n, true
		}
	}
	return 0, false
}

func valueOf(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func isASCII(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func cloneValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = cloneValue(item)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = cloneValue(item)
		}
		return out
	default:
		return value
	}
}
