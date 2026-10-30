package controller

import (
	"context"
	"fmt"
	"net/url"
)

// Gateways lists the gateways visible to the site.
func (s *SiteClient) Gateways(ctx context.Context) ([]Gateway, error) {
	entries, err := s.c.pagesV2(ctx, "gateways", s.site, url.Values{"deviceType": {"gateway"}})
	if err != nil {
		return nil, err
	}
	gateways := make([]Gateway, 0, len(entries))
	for _, entry := range entries {
		gateways = append(gateways, NewGateway(entry))
	}
	return gateways, nil
}

// Gateway returns the gateway at the given device ID.
func (s *SiteClient) Gateway(ctx context.Context, deviceID string) (Gateway, error) {
	if deviceID == "" {
		return Gateway{}, &ValidationError{Msg: "device ID must not be empty"}
	}
	m, err := s.c.requestEndpoint(ctx, "GET", "v2", "gateways/"+deviceID, s.site, nil, nil)
	if err != nil {
		return Gateway{}, err
	}
	if len(m) == 0 {
		return Gateway{}, fmt.Errorf("%w: device %q", ErrInvalidDevice, deviceID)
	}
	return NewGateway(m), nil
}

// GatewayPortSettings is the effective configuration of a gateway port.
type GatewayPortSettings struct {
	data map[string]any
}

// NewGatewayPortSettings builds a GatewayPortSettings from a raw API object.
func NewGatewayPortSettings(data map[string]any) GatewayPortSettings {
	return GatewayPortSettings{data: data}
}

// Index is the port number.
func (p GatewayPortSettings) Index() int { return mapInt(p.data, "port") }

// Type is the port type.
func (p GatewayPortSettings) Type() GatewayPortType { return GatewayPortType(mapInt(p.data, "type")) }

// PoE is the PoE mode of the port.
func (p GatewayPortSettings) PoE() PoEMode {
	if v := mapPtrInt(p.data, "poe"); v != nil {
		return PoEMode(*v)
	}
	return PoEModeNone
}

// Speed is the port speed.
func (p GatewayPortSettings) Speed() string { return mapStr(p.data, "speed") }

// Enabled reports whether the port is enabled.
func (p GatewayPortSettings) Enabled() bool { return mapBool(p.data, "enable") }

// GatewayPorts returns the port settings of the gateway.
func (s *SiteClient) GatewayPorts(ctx context.Context, deviceID string) ([]GatewayPortSettings, error) {
	if deviceID == "" {
		return nil, &ValidationError{Msg: "device ID must not be empty"}
	}
	m, err := s.c.requestEndpoint(ctx, "GET", "v2", "gateways/"+deviceID+"/portConfig", s.site, nil, nil)
	if err != nil {
		return nil, err
	}
	entries := mapList(m, "data")
	ports := make([]GatewayPortSettings, 0, len(entries))
	for _, entry := range entries {
		if pm, ok := entry.(map[string]any); ok {
			ports = append(ports, NewGatewayPortSettings(pm))
		}
	}
	return ports, nil
}

// SetGatewayPortSettings changes the configuration of a gateway port, merging
// the requested settings into the current port configuration.
func (s *SiteClient) SetGatewayPortSettings(ctx context.Context, deviceID string, portIndex int, poe PoEMode, speed string, enabled *bool) error {
	if deviceID == "" {
		return &ValidationError{Msg: "device ID must not be empty"}
	}
	ports, err := s.GatewayPorts(ctx, deviceID)
	if err != nil {
		return err
	}
	var target *GatewayPortSettings
	for i := range ports {
		if ports[i].Index() == portIndex {
			target = &ports[i]
			break
		}
	}
	if target == nil {
		return fmt.Errorf("%w: port %d not found", ErrInvalidDevice, portIndex)
	}
	if enabled == nil && poe == target.PoE() && speed == target.Speed() {
		return &ValidationError{Msg: "no gateway port changes requested"}
	}
	merged := map[string]any{
		"port":  portIndex,
		"type":  int(target.Type()),
		"poe":   int(poe),
		"speed": speed,
	}
	if enabled != nil {
		merged["enable"] = *enabled
	}
	_, err = s.c.requestEndpoint(ctx, "PATCH", "v2", "gateways/"+deviceID+"/portConfig", s.site, nil, merged)
	return err
}

// SetGatewayPoE sets the PoE state of a gateway port.
func (s *SiteClient) SetGatewayPoE(ctx context.Context, deviceID string, portIndex int, state PoEMode) error {
	if deviceID == "" {
		return &ValidationError{Msg: "device ID must not be empty"}
	}
	body := map[string]any{
		"port": portIndex,
		"poe":  int(state),
	}
	_, err := s.c.requestEndpoint(ctx, "PATCH", "v2", "gateways/"+deviceID+"/portConfig", s.site, nil, body)
	return err
}
