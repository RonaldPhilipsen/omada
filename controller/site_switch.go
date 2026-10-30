package controller

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// Switches lists the switches visible to the site.
func (s *SiteClient) Switches(ctx context.Context) ([]Switch, error) {
	entries, err := s.c.pagesV2(ctx, "switches", s.site, url.Values{"deviceType": {"switch"}})
	if err != nil {
		return nil, err
	}
	switches := make([]Switch, 0, len(entries))
	for _, entry := range entries {
		switches = append(switches, NewSwitch(entry))
	}
	return switches, nil
}

// Switch returns the switch at the given device ID.
func (s *SiteClient) Switch(ctx context.Context, deviceID string) (Switch, error) {
	if deviceID == "" {
		return Switch{}, &ValidationError{Msg: "device ID must not be empty"}
	}
	m, err := s.c.requestEndpoint(ctx, "GET", "v2", "switches/"+deviceID, s.site, nil, nil)
	if err != nil {
		return Switch{}, err
	}
	if len(m) == 0 {
		return Switch{}, fmt.Errorf("%w: device %q", ErrInvalidDevice, deviceID)
	}
	return NewSwitch(m), nil
}

// SwitchesByMAC returns the switches matching the given MACs.
func (s *SiteClient) SwitchesByMAC(ctx context.Context, macs []string) ([]Switch, error) {
	if len(macs) == 0 {
		return nil, &ValidationError{Msg: "MAC list must not be empty"}
	}
	joined := strings.Join(macs, ",")
	m, err := s.c.openapiRequest(ctx, "GET", "v2", "switches", s.site, url.Values{"mac": {joined}}, nil)
	if err != nil {
		return nil, err
	}
	entries := mapList(m, "data")
	switches := make([]Switch, 0, len(entries))
	for _, entry := range entries {
		if sm, ok := entry.(map[string]any); ok {
			switches = append(switches, NewSwitch(sm))
		}
	}
	return switches, nil
}

// SwitchPortSettings is the effective configuration of a switch port.
type SwitchPortSettings struct {
	data map[string]any
}

// NewSwitchPortSettings builds a SwitchPortSettings from a raw API object.
func NewSwitchPortSettings(data map[string]any) SwitchPortSettings {
	return SwitchPortSettings{data: data}
}

// Index is the port number.
func (p SwitchPortSettings) Index() int { return mapInt(p.data, "port") }

// Name is the port name.
func (p SwitchPortSettings) Name() string { return mapStr(p.data, "name") }

// Mode is the port mode.
func (p SwitchPortSettings) Mode() PortType { return PortType(mapInt(p.data, "type")) }

// PVID is the port PVID.
func (p SwitchPortSettings) PVID() int { return mapInt(p.data, "pvid") }

// SwitchPorts returns the port settings of the switch.
func (s *SiteClient) SwitchPorts(ctx context.Context, deviceID string) ([]SwitchPortSettings, error) {
	if deviceID == "" {
		return nil, &ValidationError{Msg: "device ID must not be empty"}
	}
	m, err := s.c.requestEndpoint(ctx, "GET", "v2", "switches/"+deviceID+"/portConfig", s.site, nil, nil)
	if err != nil {
		return nil, err
	}
	entries := mapList(m, "data")
	settings := make([]SwitchPortSettings, 0, len(entries))
	for _, entry := range entries {
		if pm, ok := entry.(map[string]any); ok {
			settings = append(settings, NewSwitchPortSettings(pm))
		}
	}
	return settings, nil
}

// SetSwitchPortSettings changes the configuration of a switch port, merging
// the requested settings into the current port configuration.
func (s *SiteClient) SetSwitchPortSettings(ctx context.Context, deviceID string, portIndex int, mode PortType, pvid int, tagged []int, untagged *int, poe *bool) error {
	if deviceID == "" {
		return &ValidationError{Msg: "device ID must not be empty"}
	}
	version, err := s.SiteVersion(ctx)
	if err != nil {
		return err
	}
	endpoint := "switches/" + deviceID + "/portConfig"
	if version.Compare(ParseVersionMust("6")) < 0 {
		endpoint = "switch/" + deviceID + "/portConfig"
	}
	current, err := s.SwitchPorts(ctx, deviceID)
	if err != nil {
		return err
	}
	var target *SwitchPortSettings
	for i := range current {
		if current[i].Index() == portIndex {
			target = &current[i]
			break
		}
	}
	if target == nil {
		return fmt.Errorf("%w: port %d not found", ErrInvalidDevice, portIndex)
	}
	if untagged == nil && poe == nil && len(tagged) == 0 && target.Mode() == mode && target.PVID() == pvid {
		return &ValidationError{Msg: "no switch port changes requested"}
	}
	merged := map[string]any{
		"port":        portIndex,
		"name":        target.Name(),
		"type":        int(mode),
		"pvid":        pvid,
		"taggedVLANs": tagged,
		"untagged":    untagged,
		"poe":         poe,
	}
	_, err = s.c.requestEndpoint(ctx, "PATCH", "v2", endpoint, s.site, nil, merged)
	return err
}

// PortProfileOverrides is the create request for a port profile.
type PortProfileOverrides struct {
	Name        string
	Description string
	Ports       []int
	PVID        int
	VLANs       []int
}

// PortProfiles lists the port profiles visible to the site.
func (s *SiteClient) PortProfiles(ctx context.Context) ([]PortProfile, error) {
	entries, err := s.c.pagesV2(ctx, "portProfiles", s.site, url.Values{})
	if err != nil {
		return nil, err
	}
	profiles := make([]PortProfile, 0, len(entries))
	for _, entry := range entries {
		profiles = append(profiles, NewPortProfile(entry))
	}
	return profiles, nil
}

// PortProfile returns the port profile with the given ID.
func (s *SiteClient) PortProfile(ctx context.Context, id string) (PortProfile, error) {
	if id == "" {
		return PortProfile{}, &ValidationError{Msg: "port profile ID must not be empty"}
	}
	m, err := s.c.requestEndpoint(ctx, "GET", "v2", "portProfiles/"+id, s.site, nil, nil)
	if err != nil {
		return PortProfile{}, err
	}
	if len(m) == 0 {
		return PortProfile{}, &ValidationError{Msg: "port profile not found"}
	}
	return NewPortProfile(m), nil
}

// CreatePortProfile creates a port profile.
func (s *SiteClient) CreatePortProfile(ctx context.Context, profile PortProfileOverrides) (PortProfile, error) {
	if profile.Name == "" {
		return PortProfile{}, &ValidationError{Msg: "port profile name must not be empty"}
	}
	body := map[string]any{
		"name":  profile.Name,
		"ports": profile.Ports,
		"pvid":  profile.PVID,
		"vlans": profile.VLANs,
	}
	m, err := s.c.requestEndpoint(ctx, "POST", "v2", "portProfiles", s.site, nil, body)
	if err != nil {
		return PortProfile{}, err
	}
	if len(m) == 0 {
		return PortProfile{}, &ValidationError{Msg: "port profile not found"}
	}
	return NewPortProfile(m), nil
}

// UpdatePortProfile updates a port profile.
func (s *SiteClient) UpdatePortProfile(ctx context.Context, id string, profile PortProfileOverrides) (PortProfile, error) {
	if id == "" {
		return PortProfile{}, &ValidationError{Msg: "port profile ID must not be empty"}
	}
	body := map[string]any{}
	if profile.Name != "" {
		body["name"] = profile.Name
	}
	if profile.Description != "" {
		body["description"] = profile.Description
	}
	if len(profile.Ports) > 0 {
		body["ports"] = profile.Ports
	}
	if profile.PVID != 0 {
		body["pvid"] = profile.PVID
	}
	if len(profile.VLANs) > 0 {
		body["vlans"] = profile.VLANs
	}
	m, err := s.c.requestEndpoint(ctx, "PATCH", "v2", "portProfiles/"+id, s.site, nil, body)
	if err != nil {
		return PortProfile{}, err
	}
	if len(m) == 0 {
		return PortProfile{}, &ValidationError{Msg: "port profile not found"}
	}
	return NewPortProfile(m), nil
}

// DeletePortProfile deletes a port profile.
func (s *SiteClient) DeletePortProfile(ctx context.Context, id string) error {
	if id == "" {
		return &ValidationError{Msg: "port profile ID must not be empty"}
	}
	_, err := s.c.requestEndpoint(ctx, "DELETE", "v2", "portProfiles/"+id, s.site, nil, nil)
	return err
}

// ApplyPortProfile applies a port profile to the given ports.
func (s *SiteClient) ApplyPortProfile(ctx context.Context, id string, ports []int) error {
	if id == "" {
		return &ValidationError{Msg: "port profile ID must not be empty"}
	}
	if len(ports) == 0 {
		return &ValidationError{Msg: "port profile apply requires ports"}
	}
	body := map[string]any{
		"profileId":   id,
		"portIndexes": ports,
	}
	_, err := s.c.requestEndpoint(ctx, "POST", "v2", "portProfiles/"+id+"/cmd/apply", s.site, nil, body)
	return err
}
