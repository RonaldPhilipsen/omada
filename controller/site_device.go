package controller

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// Devices lists the devices visible to the site.
func (s *SiteClient) Devices(ctx context.Context) ([]Device, error) {
	entries, err := s.c.pagesV2(ctx, "devices", s.site, url.Values{})
	if err != nil {
		return nil, err
	}
	devices := make([]Device, 0, len(entries))
	for _, entry := range entries {
		devices = append(devices, NewDevice(entry))
	}
	return devices, nil
}

// Device returns the device at the given device ID.
func (s *SiteClient) Device(ctx context.Context, deviceID string) (Device, error) {
	if deviceID == "" {
		return Device{}, &ValidationError{Msg: "device ID must not be empty"}
	}
	m, err := s.c.requestEndpoint(ctx, "GET", "v2", "devices/"+deviceID, s.site, nil, nil)
	if err != nil {
		return Device{}, err
	}
	if len(m) == 0 {
		return Device{}, fmt.Errorf("%w: device %q", ErrInvalidDevice, deviceID)
	}
	return NewDevice(m), nil
}

// DeviceByMAC returns the device with the given MAC.
func (s *SiteClient) DeviceByMAC(ctx context.Context, mac string) (Device, error) {
	if mac == "" {
		return Device{}, &ValidationError{Msg: "device MAC must not be empty"}
	}
	m, err := s.c.openapiRequest(ctx, "GET", "v2", "devices", s.site, url.Values{"mac": {mac}}, nil)
	if err != nil {
		return Device{}, err
	}
	if len(m) == 0 {
		return Device{}, fmt.Errorf("%w: device with MAC %q", ErrInvalidDevice, mac)
	}
	return NewDevice(m), nil
}

// Topology fetches the site's physical topology tree.
func (s *SiteClient) Topology(ctx context.Context) (Topology, error) {
	m, err := s.c.requestEndpoint(ctx, "GET", "v2", "topology", s.site, nil, nil)
	if err != nil {
		return Topology{}, err
	}
	return NewTopology(m), nil
}

// TopologyNode returns the topology node for the given network ID.
func (s *SiteClient) TopologyNode(ctx context.Context, networkID string) (TopologyNode, error) {
	if networkID == "" {
		return TopologyNode{}, &ValidationError{Msg: "network ID must not be empty"}
	}
	m, err := s.c.requestEndpoint(ctx, "GET", "v2", "topology/"+networkID, s.site, nil, nil)
	if err != nil {
		return TopologyNode{}, err
	}
	if len(m) == 0 {
		return TopologyNode{}, &ValidationError{Msg: "topology node not found"}
	}
	return NewTopologyNode(m), nil
}

// SetDeviceLED sets the onboard LED state of the device.
func (s *SiteClient) SetDeviceLED(ctx context.Context, deviceID string, state LedSetting, duration time.Duration) error {
	if deviceID == "" {
		return &ValidationError{Msg: "device ID must not be empty"}
	}
	body := map[string]any{
		"state":    int(state),
		"duration": int(duration / time.Second),
	}
	_, err := s.c.requestEndpoint(ctx, "POST", "v2", "devices/"+deviceID+"/led", s.site, nil, body)
	return err
}

// RebootDevice reboots the device.
func (s *SiteClient) RebootDevice(ctx context.Context, deviceID string) error {
	if deviceID == "" {
		return &ValidationError{Msg: "device ID must not be empty"}
	}
	_, err := s.c.requestEndpoint(ctx, "POST", "v2", "devices/"+deviceID+"/cmd/reboot", s.site, nil, nil)
	return err
}

// UpgradeDeviceFirmware upgrades the firmware of the given devices.
func (s *SiteClient) UpgradeDeviceFirmware(ctx context.Context, deviceIDs []string) error {
	if len(deviceIDs) == 0 {
		return &ValidationError{Msg: "device IDs must not be empty"}
	}
	body := map[string]any{
		"deviceIds": deviceIDs,
	}
	_, err := s.c.requestEndpoint(ctx, "POST", "v2", "cmd/upgrade", s.site, nil, body)
	return err
}

// GetFirmwareUpdates lists the firmware updates for the site's devices.
func (s *SiteClient) GetFirmwareUpdates(ctx context.Context) ([]FirmwareUpdate, error) {
	m, err := s.c.requestEndpoint(ctx, "GET", "v2", "upgrade", s.site, nil, nil)
	if err != nil {
		return nil, err
	}
	entries := mapList(m, "data")
	updates := make([]FirmwareUpdate, 0, len(entries))
	for _, entry := range entries {
		if em, ok := entry.(map[string]any); ok {
			updates = append(updates, NewFirmwareUpdate(em))
		}
	}
	return updates, nil
}

// UpdateFirmwareChannel selects the firmware release channel to check.
func (s *SiteClient) UpdateFirmwareChannel(ctx context.Context, section UpdateSection, channel string) error {
	body := map[string]any{
		"section": string(section),
		"channel": channel,
	}
	_, err := s.c.requestEndpoint(ctx, "POST", "v2", "upgrade", s.site, nil, body)
	return err
}

// RestoreDevice restores the device.
func (s *SiteClient) RestoreDevice(ctx context.Context, deviceID string) error {
	if deviceID == "" {
		return &ValidationError{Msg: "device ID must not be empty"}
	}
	_, err := s.c.requestEndpoint(ctx, "POST", "v2", "devices/"+deviceID+"/cmd/restore", s.site, nil, nil)
	return err
}

// FactoryResetDevice resets the device to factory defaults.
func (s *SiteClient) FactoryResetDevice(ctx context.Context, deviceID string) error {
	if deviceID == "" {
		return &ValidationError{Msg: "device ID must not be empty"}
	}
	_, err := s.c.requestEndpoint(ctx, "POST", "v2", "devices/"+deviceID+"/cmd/factoryReset", s.site, nil, nil)
	return err
}
