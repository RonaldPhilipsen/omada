package site

import (
	"context"
	"fmt"
	models "github.com/RonaldPhilipsen/omada/controller/models"
	"net/url"
)

// Gateways lists the gateways visible to the site.
func (s *SiteClient) Gateways(ctx context.Context) ([]models.Gateway, error) {
	entries, err := s.c.PagesV2(ctx, "gateways", s.site, url.Values{"deviceType": {"gateway"}})
	if err != nil {
		return nil, err
	}
	gateways := make([]models.Gateway, 0, len(entries))
	for _, entry := range entries {
		gateways = append(gateways, models.NewGateway(entry))
	}
	return gateways, nil
}

// models.Gateway returns the gateway at the given device ID.
func (s *SiteClient) Gateway(ctx context.Context, deviceID string) (models.Gateway, error) {
	if deviceID == "" {
		return models.Gateway{}, &models.ValidationError{Msg: "device ID must not be empty"}
	}
	m, err := s.c.RequestEndpoint(ctx, "GET", "v2", "gateways/"+deviceID, s.site, nil, nil)
	if err != nil {
		return models.Gateway{}, err
	}
	if len(m) == 0 {
		return models.Gateway{}, fmt.Errorf("%w: device %q", models.ErrInvalidDevice, deviceID)
	}
	return models.NewGateway(m), nil
}

// GetGatewayPort returns the configuration of a single gateway port.
func (s *SiteClient) GetGatewayPort(ctx context.Context, deviceID string, portIndex int) (*models.GatewayPortConfig, error) {
	if deviceID == "" {
		return nil, &models.ValidationError{Msg: "device ID must not be empty"}
	}
	gw, err := s.Gateway(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	for _, entry := range gw.PortConfigs() {
		if entry.PortNumber() == portIndex {
			cfg := entry
			return &cfg, nil
		}
	}
	return nil, &models.ValidationError{Msg: "gateway port not found"}
}

// SetGatewayWanPortConnectState connects or disconnects a gateway WAN port
// and returns the resulting port status.
func (s *SiteClient) SetGatewayWanPortConnectState(ctx context.Context, deviceID string, portIndex int, connect bool) (models.GatewayPortStatus, error) {
	if deviceID == "" {
		return models.GatewayPortStatus{}, &models.ValidationError{Msg: "device ID must not be empty"}
	}
	operation := 0
	if connect {
		operation = 1
	}
	body := map[string]any{"portId": portIndex, "operation": operation}
	m, err := s.c.RequestEndpoint(ctx, "POST", "v2", "gateways/"+deviceID+"/internetState", s.site, nil, body)
	if err != nil {
		return models.GatewayPortStatus{}, err
	}
	return models.NewGatewayPortStatus(m), nil
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
func (p GatewayPortSettings) Index() int { return models.MapInt(p.data, "port") }

// Type is the port type.
func (p GatewayPortSettings) Type() models.GatewayPortType {
	return models.GatewayPortType(models.MapInt(p.data, "type"))
}

// PoE is the PoE mode of the port.
func (p GatewayPortSettings) PoE() models.PoEMode {
	if v := models.MapPtrInt(p.data, "poe"); v != nil {
		return models.PoEMode(*v)
	}
	return models.PoEModeNone
}

// Speed is the port speed.
func (p GatewayPortSettings) Speed() string { return models.MapStr(p.data, "speed") }

// Enabled reports whether the port is enabled.
func (p GatewayPortSettings) Enabled() bool { return models.MapBool(p.data, "enable") }

// GatewayPorts returns the port settings of the gateway.
func (s *SiteClient) GatewayPorts(ctx context.Context, deviceID string) ([]GatewayPortSettings, error) {
	if deviceID == "" {
		return nil, &models.ValidationError{Msg: "device ID must not be empty"}
	}
	m, err := s.c.RequestEndpoint(ctx, "GET", "v2", "gateways/"+deviceID+"/portConfig", s.site, nil, nil)
	if err != nil {
		return nil, err
	}
	entries := models.MapList(m, "data")
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
func (s *SiteClient) SetGatewayPortSettings(ctx context.Context, deviceID string, portIndex int, poe models.PoEMode, speed string, enabled *bool) error {
	if deviceID == "" {
		return &models.ValidationError{Msg: "device ID must not be empty"}
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
		return fmt.Errorf("%w: port %d not found", models.ErrInvalidDevice, portIndex)
	}
	if enabled == nil && poe == target.PoE() && speed == target.Speed() {
		return &models.ValidationError{Msg: "no gateway port changes requested"}
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
	_, err = s.c.RequestEndpoint(ctx, "PATCH", "v2", "gateways/"+deviceID+"/portConfig", s.site, nil, merged)
	return err
}

// SetGatewayPoE sets the PoE state of a gateway port.
func (s *SiteClient) SetGatewayPoE(ctx context.Context, deviceID string, portIndex int, state models.PoEMode) error {
	if deviceID == "" {
		return &models.ValidationError{Msg: "device ID must not be empty"}
	}
	body := map[string]any{
		"port": portIndex,
		"poe":  int(state),
	}
	_, err := s.c.RequestEndpoint(ctx, "PATCH", "v2", "gateways/"+deviceID+"/portConfig", s.site, nil, body)
	return err
}
