package controller

import (
	"context"
	"fmt"
	models "github.com/RonaldPhilipsen/omada/controller/models"
	"net/url"
	"time"
)

// Devices lists the devices visible to the site.
func (s *SiteClient) Devices(ctx context.Context) ([]models.Device, error) {
	entries, err := s.c.PagesV2(ctx, "devices", s.site, url.Values{})
	if err != nil {
		return nil, err
	}
	devices := make([]models.Device, 0, len(entries))
	for _, entry := range entries {
		devices = append(devices, models.NewDevice(entry))
	}
	return devices, nil
}

// models.Device returns the device at the given device ID.
func (s *SiteClient) Device(ctx context.Context, deviceID string) (models.Device, error) {
	if deviceID == "" {
		return models.Device{}, &models.ValidationError{Msg: "device ID must not be empty"}
	}
	m, err := s.c.RequestEndpoint(ctx, "GET", "v2", "devices/"+deviceID, s.site, nil, nil)
	if err != nil {
		return models.Device{}, err
	}
	if len(m) == 0 {
		return models.Device{}, fmt.Errorf("%w: device %q", models.ErrInvalidDevice, deviceID)
	}
	return models.NewDevice(m), nil
}

// DeviceByMAC returns the device with the given MAC.
func (s *SiteClient) DeviceByMAC(ctx context.Context, mac string) (models.Device, error) {
	if mac == "" {
		return models.Device{}, &models.ValidationError{Msg: "device MAC must not be empty"}
	}
	m, err := s.c.OpenapiRequest(ctx, "GET", "v2", "devices", s.site, url.Values{"mac": {mac}}, nil)
	if err != nil {
		return models.Device{}, err
	}
	if len(m) == 0 {
		return models.Device{}, fmt.Errorf("%w: device with MAC %q", models.ErrInvalidDevice, mac)
	}
	return models.NewDevice(m), nil
}

// models.Topology fetches the site's physical topology tree.
func (s *SiteClient) Topology(ctx context.Context) (models.Topology, error) {
	m, err := s.c.RequestEndpoint(ctx, "GET", "v2", "topology", s.site, nil, nil)
	if err != nil {
		return models.Topology{}, err
	}
	return models.NewTopology(m), nil
}

// models.TopologyNode returns the topology node for the given network ID.
func (s *SiteClient) TopologyNode(ctx context.Context, networkID string) (models.TopologyNode, error) {
	if networkID == "" {
		return models.TopologyNode{}, &models.ValidationError{Msg: "network ID must not be empty"}
	}
	m, err := s.c.RequestEndpoint(ctx, "GET", "v2", "topology/"+networkID, s.site, nil, nil)
	if err != nil {
		return models.TopologyNode{}, err
	}
	if len(m) == 0 {
		return models.TopologyNode{}, &models.ValidationError{Msg: "topology node not found"}
	}
	return models.NewTopologyNode(m), nil
}

// SetDeviceLED sets the onboard LED state of the device.
func (s *SiteClient) SetDeviceLED(ctx context.Context, deviceID string, state models.LedSetting, duration time.Duration) error {
	if deviceID == "" {
		return &models.ValidationError{Msg: "device ID must not be empty"}
	}
	body := map[string]any{
		"state":    int(state),
		"duration": int(duration / time.Second),
	}
	_, err := s.c.RequestEndpoint(ctx, "POST", "v2", "devices/"+deviceID+"/led", s.site, nil, body)
	return err
}

// RebootDevice reboots the device.
func (s *SiteClient) RebootDevice(ctx context.Context, deviceID string) error {
	if deviceID == "" {
		return &models.ValidationError{Msg: "device ID must not be empty"}
	}
	_, err := s.c.RequestEndpoint(ctx, "POST", "v2", "devices/"+deviceID+"/cmd/reboot", s.site, nil, nil)
	return err
}

// UpgradeDeviceFirmware upgrades the firmware of the given devices.
func (s *SiteClient) UpgradeDeviceFirmware(ctx context.Context, deviceIDs []string) error {
	if len(deviceIDs) == 0 {
		return &models.ValidationError{Msg: "device IDs must not be empty"}
	}
	body := map[string]any{
		"deviceIds": deviceIDs,
	}
	_, err := s.c.RequestEndpoint(ctx, "POST", "v2", "cmd/upgrade", s.site, nil, body)
	return err
}

// GetFirmwareUpdates lists the firmware updates for the site's devices.
func (s *SiteClient) GetFirmwareUpdates(ctx context.Context) ([]models.FirmwareUpdate, error) {
	m, err := s.c.RequestEndpoint(ctx, "GET", "v2", "upgrade", s.site, nil, nil)
	if err != nil {
		return nil, err
	}
	entries := models.MapList(m, "data")
	updates := make([]models.FirmwareUpdate, 0, len(entries))
	for _, entry := range entries {
		if em, ok := entry.(map[string]any); ok {
			updates = append(updates, models.NewFirmwareUpdate(em))
		}
	}
	return updates, nil
}

// GetFirmwareDetails returns the firmware details of a single device.
func (s *SiteClient) GetFirmwareDetails(ctx context.Context, deviceID string) (models.FirmwareUpdate, error) {
	if deviceID == "" {
		return models.FirmwareUpdate{}, &models.ValidationError{Msg: "device ID must not be empty"}
	}
	m, err := s.c.RequestEndpoint(ctx, "GET", "v2", "devices/"+deviceID+"/firmware", s.site, nil, nil)
	if err != nil {
		return models.FirmwareUpdate{}, err
	}
	if len(m) == 0 {
		return models.FirmwareUpdate{}, fmt.Errorf("%w: device %q not found", models.ErrInvalidDevice, deviceID)
	}
	return models.NewFirmwareUpdate(m), nil
}

// UpdateFirmwareChannel selects the firmware release channel to check.
func (s *SiteClient) UpdateFirmwareChannel(ctx context.Context, section UpdateSection, channel string) error {
	body := map[string]any{
		"section": string(section),
		"channel": channel,
	}
	_, err := s.c.RequestEndpoint(ctx, "POST", "v2", "upgrade", s.site, nil, body)
	return err
}

// RestoreDevice restores the device.
func (s *SiteClient) RestoreDevice(ctx context.Context, deviceID string) error {
	if deviceID == "" {
		return &models.ValidationError{Msg: "device ID must not be empty"}
	}
	_, err := s.c.RequestEndpoint(ctx, "POST", "v2", "devices/"+deviceID+"/cmd/restore", s.site, nil, nil)
	return err
}

// FactoryResetDevice resets the device to factory defaults.
func (s *SiteClient) FactoryResetDevice(ctx context.Context, deviceID string) error {
	if deviceID == "" {
		return &models.ValidationError{Msg: "device ID must not be empty"}
	}
	_, err := s.c.RequestEndpoint(ctx, "POST", "v2", "devices/"+deviceID+"/cmd/factoryReset", s.site, nil, nil)
	return err
}
