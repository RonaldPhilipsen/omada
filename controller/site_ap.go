package controller

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// AccessPoints lists the access points visible to the site.
func (s *SiteClient) AccessPoints(ctx context.Context) ([]AccessPoint, error) {
	entries, err := s.c.pagesV2(ctx, "accessPoints", s.site, url.Values{"deviceType": {"accessPoint"}})
	if err != nil {
		return nil, err
	}
	aps := make([]AccessPoint, 0, len(entries))
	for _, entry := range entries {
		aps = append(aps, NewAccessPoint(entry))
	}
	return aps, nil
}

// AccessPoint returns the access point at the given device ID.
func (s *SiteClient) AccessPoint(ctx context.Context, deviceID string) (AccessPoint, error) {
	if deviceID == "" {
		return AccessPoint{}, &ValidationError{Msg: "device ID must not be empty"}
	}
	m, err := s.c.requestEndpoint(ctx, "GET", "v2", "accessPoints/"+deviceID, s.site, nil, nil)
	if err != nil {
		return AccessPoint{}, err
	}
	if len(m) == 0 {
		return AccessPoint{}, fmt.Errorf("%w: device %q", ErrInvalidDevice, deviceID)
	}
	return NewAccessPoint(m), nil
}

// AccessPointsByMAC returns the access points matching the given MACs.
func (s *SiteClient) AccessPointsByMAC(ctx context.Context, macs []string) ([]AccessPoint, error) {
	if len(macs) == 0 {
		return nil, &ValidationError{Msg: "MAC list must not be empty"}
	}
	joined := strings.Join(macs, ",")
	m, err := s.c.openapiRequest(ctx, "GET", "v2", "accessPoints", s.site, url.Values{"mac": {joined}}, nil)
	if err != nil {
		return nil, err
	}
	entries := mapList(m, "data")
	aps := make([]AccessPoint, 0, len(entries))
	for _, entry := range entries {
		if am, ok := entry.(map[string]any); ok {
			aps = append(aps, NewAccessPoint(am))
		}
	}
	return aps, nil
}

// ApPorts returns the LAN port settings of the access point.
func (s *SiteClient) ApPorts(ctx context.Context, deviceID string) ([]APLANPortSettings, error) {
	if deviceID == "" {
		return nil, &ValidationError{Msg: "device ID must not be empty"}
	}
	m, err := s.c.openapiRequest(ctx, "GET", "v2", "accessPoints/"+deviceID+"/portConfig", s.site, nil, nil)
	if err != nil {
		return nil, err
	}
	entries := mapList(m, "data")
	ports := make([]APLANPortSettings, 0, len(entries))
	for _, entry := range entries {
		if pm, ok := entry.(map[string]any); ok {
			ports = append(ports, NewAPLANPortSettings(pm))
		}
	}
	return ports, nil
}

// SetApPortEnabled enables or disables an access point port.
func (s *SiteClient) SetApPortEnabled(ctx context.Context, deviceID string, portIndex int, enabled bool, poe bool) error {
	if deviceID == "" {
		return &ValidationError{Msg: "device ID must not be empty"}
	}
	body := map[string]any{
		"index":     portIndex,
		"enable":    enabled,
		"poeEnable": poe,
	}
	_, err := s.c.openapiRequest(ctx, "PATCH", "v2", "accessPoints/"+deviceID+"/portConfig", s.site, nil, body)
	return err
}

// ApRadioSettings returns the configured state of an access point radio.
func (s *SiteClient) ApRadioSettings(ctx context.Context, deviceID string, radioID int) (APRadioSettings, error) {
	ap, err := s.AccessPoint(ctx, deviceID)
	if err != nil {
		return APRadioSettings{}, err
	}
	settings := ap.RadioSettings(RadioId(radioID))
	if settings == nil {
		return APRadioSettings{}, &ValidationError{Msg: "radio settings not found"}
	}
	return *settings, nil
}

// SetApRadioEnabled enables or disables an access point radio.
func (s *SiteClient) SetApRadioEnabled(ctx context.Context, deviceID string, radioID int, enabled bool) error {
	settings, err := s.ApRadioSettings(ctx, deviceID, radioID)
	if err != nil {
		return err
	}
	settings.data["radioEnable"] = enabled
	_, err = s.c.requestEndpoint(ctx, "PATCH", "v2", "accessPoints/"+deviceID+"/radioConfig", s.site, nil, settings.data)
	return err
}

// SetApRadioChannel sets the channel and channel width of an access point radio.
func (s *SiteClient) SetApRadioChannel(ctx context.Context, deviceID string, radioID int, channel int, channelWidth ChannelWidth) error {
	settings, err := s.ApRadioSettings(ctx, deviceID, radioID)
	if err != nil {
		return err
	}
	settings.data["channel"] = channel
	settings.data["channelWidth"] = int(channelWidth)
	_, err = s.c.requestEndpoint(ctx, "PATCH", "v2", "accessPoints/"+deviceID+"/radioConfig", s.site, nil, settings.data)
	return err
}

// ApAvailableChannels returns the channels available on the access point radio.
func (s *SiteClient) ApAvailableChannels(ctx context.Context, deviceID string, radioID int) ([]int, error) {
	m, err := s.c.openapiRequest(ctx, "GET", "v2", "accessPoints/"+deviceID+"/radioConfig/availableChannels", s.site, nil, nil)
	if err != nil {
		return nil, err
	}
	raw, ok := m["data"].([]any)
	if !ok {
		return nil, &ValidationError{Msg: "unexpected available channels response"}
	}
	channels := make([]int, 0, len(raw))
	for _, entry := range raw {
		mm, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		if v, ok := mm["channel"].(float64); ok {
			channels = append(channels, int(v))
		}
	}
	return channels, nil
}
