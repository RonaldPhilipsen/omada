package controller

import "strings"

// Site identifies a site controlled by the controller.
type Site struct {
	Name string
	ID   string
}

// ControllerInfo wraps the /api/info response.
type ControllerInfo struct {
	raw map[string]any
}

func NewControllerInfo(m map[string]any) ControllerInfo {
	return ControllerInfo{raw: m}
}

// Raw returns the original response map.
func (c ControllerInfo) Raw() map[string]any { return c.raw }

// ID is the controller's unique ID (omadacId).
func (c ControllerInfo) ID() string { return mapStr(c.raw, "omadacId") }

// ControllerVersion is the controller software version.
func (c ControllerInfo) ControllerVersion() string { return mapStr(c.raw, "controllerVer") }

// APIVersion is the controller API version, when reported.
func (c ControllerInfo) APIVersion() string { return mapStr(c.raw, "apiVer") }

// Category is the controller category, when reported.
func (c ControllerInfo) Category() string { return mapStr(c.raw, "omadacCategory") }

// CloudURL is the Omada cloud URL, when reported.
func (c ControllerInfo) CloudURL() string { return mapStr(c.raw, "omadaCloudUrl") }

// Configured reports whether the setup wizard has been completed.
func (c ControllerInfo) Configured() *bool { return mapPtrBool(c.raw, "configured") }

// SupportApp reports whether the Omada app is supported.
func (c ControllerInfo) SupportApp() *bool { return mapPtrBool(c.raw, "supportApp") }

// RegisteredRoot reports whether the root account is registered.
func (c ControllerInfo) RegisteredRoot() *bool { return mapPtrBool(c.raw, "registeredRoot") }

// MspMode reports whether the controller runs in MSP mode.
func (c ControllerInfo) MspMode() *bool { return mapPtrBool(c.raw, "mspMode") }

// Type is the controller type code as returned by the API.
func (c ControllerInfo) Type() *int { return mapPtrInt(c.raw, "type") }

// ControllerTypeModel wraps the anon/controllerType response.
type ControllerTypeModel struct {
	raw map[string]any
}

func NewControllerTypeModel(m map[string]any) ControllerTypeModel {
	return ControllerTypeModel{raw: m}
}

func (c ControllerTypeModel) Raw() map[string]any { return c.raw }

// IsSoftController reports whether this is an Omada Software Controller.
func (c ControllerTypeModel) IsSoftController() bool { return mapBool(c.raw, "isSoftController") }

// CombinedGateway reports whether the controller runs on a combined gateway.
func (c ControllerTypeModel) CombinedGateway() bool { return mapBool(c.raw, "combinedGateway") }

// ControllerStatus wraps the maintenance/controllerStatus response.
type ControllerStatus struct {
	raw map[string]any
}

func NewControllerStatus(m map[string]any) ControllerStatus {
	return ControllerStatus{raw: m}
}

func (c ControllerStatus) Raw() map[string]any { return c.raw }

// Name is the controller display name, stripped; nil when absent.
func (c ControllerStatus) Name() *string {
	name := strings.TrimSpace(mapStr(c.raw, "name"))
	if name == "" {
		return nil
	}
	return &name
}

// MAC is the controller MAC address.
func (c ControllerStatus) MAC() string { return mapStr(c.raw, "macAddress") }

// Uptime is the controller uptime in seconds.
func (c ControllerStatus) Uptime() int64 { return int64(mapInt(c.raw, "upTime")) }

// ControllerVersion is the version reported by the status endpoint.
func (c ControllerStatus) ControllerVersion() string {
	return mapStr(c.raw, "controllerVersion")
}

// Model is the hardware model, falling back to "Controller".
func (c ControllerStatus) Model() string {
	if model := strings.TrimSpace(mapStr(c.raw, "model")); model != "" {
		return model
	}
	return "Controller"
}

// HardwareUpgradeStatus wraps the maintenance/hardware/upgradeStatus response.
type HardwareUpgradeStatus struct {
	raw map[string]any
}

func NewHardwareUpgradeStatus(m map[string]any) HardwareUpgradeStatus {
	return HardwareUpgradeStatus{raw: m}
}

func (c HardwareUpgradeStatus) Raw() map[string]any { return c.raw }

// UpgradeStatus is the current status of the firmware upgrade process.
func (c HardwareUpgradeStatus) UpgradeStatus() int { return mapInt(c.raw, "upgradeStatus") }

// UpgradeMsg is the message associated with the current status.
func (c HardwareUpgradeStatus) UpgradeMsg() string {
	return mapStr(c.raw, "upgradeMsg")
}

// UpgradeTime is the suggested refresh interval for watching progress.
func (c HardwareUpgradeStatus) UpgradeTime() int { return mapInt(c.raw, "upgradeTime") }

// DownloadProgress is the firmware download progress percentage.
func (c HardwareUpgradeStatus) DownloadProgress() int { return mapInt(c.raw, "downloadProgress") }

// RebootTime is the expected seconds before the controller is back online.
func (c HardwareUpgradeStatus) RebootTime() int {
	if v := mapInt(c.raw, "rebootTime"); v != 0 {
		return v
	}
	return 300
}

// updateInfoSection wraps one half of a controller update report.
type updateInfoSection struct {
	raw            map[string]any
	notesFirstKey  string
	notesSecondKey string
}

func (u updateInfoSection) Raw() map[string]any { return u.raw }

// Upgrade reports whether an update is available.
func (u updateInfoSection) Upgrade() bool { return mapBool(u.raw, "upgrade") }

// CurrentVersion is the currently installed version, when reported.
func (u updateInfoSection) CurrentVersion() *string {
	v := mapStr(u.raw, "currentVersion")
	if v == "" {
		return nil
	}
	return &v
}

// LatestVersion is the latest available version, falling back to the current one.
func (u updateInfoSection) LatestVersion() *string {
	v := mapStr(u.raw, "latestVersion")
	if v == "" {
		v = mapStr(u.raw, "currentVersion")
	}
	if v == "" {
		return nil
	}
	return &v
}

// ReleaseNotes is the release notes, checking both known keys.
func (u updateInfoSection) ReleaseNotes() *string {
	for _, key := range []string{u.notesFirstKey, u.notesSecondKey} {
		if v := mapStr(u.raw, key); v != "" {
			return &v
		}
	}
	return nil
}

// DownloadLink is the update download link, stripped; nil when absent.
func (u updateInfoSection) DownloadLink() *string {
	return strippedPtr(mapStr(u.raw, "downloadLink"))
}

// ReleaseURL is the release information URL, stripped; nil when absent.
func (u updateInfoSection) ReleaseURL() *string {
	return strippedPtr(mapStr(u.raw, "releaseUrl"))
}

type HardwareUpdateInfo struct{ updateInfoSection }
type SoftwareUpdateInfo struct{ updateInfoSection }

// NewHardwareUpdateInfo wraps a controller hardware update section.
func NewHardwareUpdateInfo(m map[string]any) HardwareUpdateInfo {
	return HardwareUpdateInfo{updateInfoSection{m, "fwReleaseLog", "releaseLog"}}
}

// NewSoftwareUpdateInfo wraps a controller software update section.
func NewSoftwareUpdateInfo(m map[string]any) SoftwareUpdateInfo {
	return SoftwareUpdateInfo{updateInfoSection{m, "releaseLog", "fwReleaseLog"}}
}

// ControllerUpdateInfo wraps a normalized controller update report.
type ControllerUpdateInfo struct {
	raw map[string]any
}

func NewControllerUpdateInfo(m map[string]any) ControllerUpdateInfo {
	return ControllerUpdateInfo{raw: m}
}

func (c ControllerUpdateInfo) Raw() map[string]any { return c.raw }

// Hardware is the hardware (firmware) update section, when present.
func (c ControllerUpdateInfo) Hardware() *HardwareUpdateInfo {
	m, ok := mapMap(c.raw, "hardware")
	if !ok || m == nil {
		return nil
	}
	h := NewHardwareUpdateInfo(m)
	return &h
}

// Software is the software update section, when present.
func (c ControllerUpdateInfo) Software() *SoftwareUpdateInfo {
	m, ok := mapMap(c.raw, "software")
	if !ok || m == nil {
		return nil
	}
	s := NewSoftwareUpdateInfo(m)
	return &s
}

// Update is the section matching the current controller type (hardware first).
func (c ControllerUpdateInfo) Update() *updateInfoSection {
	if h := c.Hardware(); h != nil {
		return &h.updateInfoSection
	}
	if s := c.Software(); s != nil {
		return &s.updateInfoSection
	}
	return nil
}

// Upgrade reports whether any update is available.
func (c ControllerUpdateInfo) Upgrade() bool {
	u := c.Update()
	return u != nil && u.Upgrade()
}

// CurrentVersion is the current version of the relevant update section.
func (c ControllerUpdateInfo) CurrentVersion() *string {
	if u := c.Update(); u != nil {
		return u.CurrentVersion()
	}
	return nil
}

// LatestVersion is the latest version of the relevant update section.
func (c ControllerUpdateInfo) LatestVersion() *string {
	if u := c.Update(); u != nil {
		return u.LatestVersion()
	}
	return nil
}

// ReleaseNotes are the notes of the relevant update section.
func (c ControllerUpdateInfo) ReleaseNotes() *string {
	if u := c.Update(); u != nil {
		return u.ReleaseNotes()
	}
	return nil
}

// ReleaseURL is the release URL of the relevant update section.
func (c ControllerUpdateInfo) ReleaseURL() *string {
	if u := c.Update(); u != nil {
		return u.ReleaseURL()
	}
	return nil
}

// InterfaceDetails wraps maintenance/uiInterface.
type InterfaceDetails struct {
	raw map[string]any
}

func NewInterfaceDetails(m map[string]any) InterfaceDetails {
	return InterfaceDetails{raw: m}
}

func (c InterfaceDetails) Raw() map[string]any { return c.raw }

// ControllerName is the display name of the controller.
func (c InterfaceDetails) ControllerName() string { return mapStr(c.raw, "controllerName") }

// FirmwareUpdate wraps a device firmware update entry.
type FirmwareUpdate struct {
	raw map[string]any
}

func NewFirmwareUpdate(m map[string]any) FirmwareUpdate {
	return FirmwareUpdate{raw: m}
}

func (c FirmwareUpdate) Raw() map[string]any { return c.raw }

// CurrentVersion is the device's current firmware version.
func (c FirmwareUpdate) CurrentVersion() string { return mapStr(c.raw, "curFwVer") }

// LatestVersion is the latest available firmware version.
func (c FirmwareUpdate) LatestVersion() string {
	if v := mapStr(c.raw, "lastFwVer"); v != "" {
		return v
	}
	return c.CurrentVersion()
}

// ReleaseNotes are the release notes for the new firmware.
func (c FirmwareUpdate) ReleaseNotes() string { return mapStr(c.raw, "fwReleaseLog") }

func strippedPtr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
