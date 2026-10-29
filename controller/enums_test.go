package controller

import "testing"

func TestDeviceStatusStrings(t *testing.T) {
	if DeviceStatusConnected.String() != "connected" {
		t.Errorf("got %q", DeviceStatusConnected.String())
	}
	if DeviceStatus(99).String() != "unknown" {
		t.Errorf("unknown member expected, got %q", DeviceStatus(99).String())
	}
	if DeviceStatusConnectedWirelessMigrating.String() != "connected_wireless_migrating" {
		t.Errorf("got %q", DeviceStatusConnectedWirelessMigrating.String())
	}
}

func TestEnumUnknownFallback(t *testing.T) {
	checks := []struct {
		name string
		got  string
		want string
	}{
		{"RadioId", RadioId(7).String(), "unknown"},
		{"PoEMode", PoEMode(9).String(), "none"},
		{"GatewayPortMode", GatewayPortMode(9).String(), "disabled"},
		{"ChannelWidth", ChannelWidth(0).String(), "unknown"},
		{"LinkSpeed", LinkSpeed(9).String(), "unknown"},
		{"ConnectType", ConnectType(9).String(), "unknown"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: got %q want %q", c.name, c.got, c.want)
		}
	}
}

func TestRadioIdKeys(t *testing.T) {
	cases := []struct {
		id       RadioId
		settings string
		status   string
	}{
		{RadioIdFreq24, "radioSetting2g", "wp2g"},
		{RadioIdFreq51, "radioSetting5g", "wp5g"},
		{RadioIdFreq52, "radioSetting5g2", "wp5g2"},
		{RadioIdFreq6, "radioSetting6g", "wp6g"},
	}
	for _, c := range cases {
		if c.id.SettingsKey() != c.settings {
			t.Errorf("settings key for %v: %q", c.id, c.id.SettingsKey())
		}
		if c.id.StatusKey() != c.status {
			t.Errorf("status key for %v: %q", c.id, c.id.StatusKey())
		}
	}
	if RadioIdUnknown.SettingsKey() != "" {
		t.Error("unknown radio must have no settings key")
	}
}

func TestEnumValuesUnique(t *testing.T) {
	// Spot-check that constants within an enum keep distinct values.
	distinct := func(vals ...int32) {
		seen := map[int32]bool{}
		for _, v := range vals {
			if seen[v] {
				t.Errorf("duplicate value %d within enum", v)
			}
			seen[v] = true
		}
	}
	distinct(int32(DeviceStatusUnknown), int32(DeviceStatusDisconnected), int32(DeviceStatusConnected), int32(DeviceStatusIsolated))
	distinct(int32(ChannelWidthWidth20), int32(ChannelWidthAuto4020), int32(ChannelWidthWidth320))
	distinct(int32(LinkSpeedSpeedAuto), int32(LinkSpeedSpeed1G), int32(LinkSpeedSpeed10G))
	distinct(int32(PoEModeDisabled), int32(PoEModeEnabled), int32(PoEModeUseDeviceSettings))
}
