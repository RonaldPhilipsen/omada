package controller

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func loadFixture(t *testing.T, name string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestControllerInfo(t *testing.T) {
	m := loadFixture(t, "controller-info.json")
	c := NewControllerInfo(m)
	if c.ID() != "abc" {
		t.Errorf("ID: %q", c.ID())
	}
	if c.ControllerVersion() != "6.2.10.18" {
		t.Errorf("version: %q", c.ControllerVersion())
	}
	if c.SupportApp() == nil || !*c.SupportApp() {
		t.Errorf("supportApp: %v", c.SupportApp())
	}
	if c.MspMode() == nil || *c.MspMode() {
		t.Errorf("mspMode: %v", c.MspMode())
	}
}

func TestControllerInfoSparse(t *testing.T) {
	c := NewControllerInfo(map[string]any{})
	if c.ID() != "" || c.ControllerVersion() != "" {
		t.Error("zero values expected")
	}
	if c.SupportApp() != nil || c.MspMode() != nil || c.RegisteredRoot() != nil {
		t.Error("nil pointers expected")
	}
}

func TestControllerTypeModel(t *testing.T) {
	m := map[string]any{"isSoftController": true, "combinedGateway": false}
	c := NewControllerTypeModel(m)
	if !c.IsSoftController() || c.CombinedGateway() {
		t.Error("flags wrong")
	}
	sparse := NewControllerTypeModel(map[string]any{})
	if sparse.IsSoftController() || sparse.CombinedGateway() {
		t.Error("zero values expected")
	}
}

func TestControllerStatusModel(t *testing.T) {
	m := map[string]any{
		"name":              "  My Controller  ",
		"macAddress":        "aa:bb:cc:dd:ee:ff",
		"upTime":            float64(123456),
		"controllerVersion": "6.2.10.18",
		"model":             "EAP670",
	}
	s := NewControllerStatus(m)
	if s.Name() == nil || *s.Name() != "My Controller" {
		t.Errorf("name: %v", s.Name())
	}
	if s.MAC() != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("mac: %q", s.MAC())
	}
	if s.Uptime() != 123456 {
		t.Errorf("uptime: %d", s.Uptime())
	}
	if s.Model() != "EAP670" {
		t.Errorf("model: %q", s.Model())
	}
	noModel := NewControllerStatus(map[string]any{})
	if noModel.Model() != "Controller" {
		t.Errorf("fallback model: %q", noModel.Model())
	}
	if noModel.Name() != nil {
		t.Error("nil name expected")
	}
}

func TestHardwareUpgradeStatus(t *testing.T) {
	m := map[string]any{
		"upgradeStatus":    float64(1),
		"upgradeMsg":       "downloading",
		"downloadProgress": float64(42),
	}
	s := NewHardwareUpgradeStatus(m)
	if s.UpgradeStatus() != 1 || s.UpgradeMsg() != "downloading" || s.DownloadProgress() != 42 {
		t.Errorf("got %+v", s)
	}
	if s.RebootTime() != 300 {
		t.Errorf("rebootTime default: %d", s.RebootTime())
	}
	if s.UpgradeTime() != 0 {
		t.Errorf("upgradeTime: %d", s.UpgradeTime())
	}
}

func TestControllerUpdateInfo(t *testing.T) {
	m := map[string]any{
		"hardware": map[string]any{
			"upgrade":        false,
			"currentVersion": "6.2.10.0",
			"fwReleaseLog":   "hw notes",
			"downloadLink":   "  https://x/hw  ",
		},
		"software": map[string]any{
			"upgrade":       true,
			"latestVersion": "6.3.0.0",
			"releaseLog":    "sw notes",
			"releaseUrl":    "https://x/sw",
		},
	}
	u := NewControllerUpdateInfo(m)
	if u.Hardware() == nil || u.Software() == nil {
		t.Fatal("both sections expected")
	}
	// The hardware section is authoritative when present (upstream:
	// self.hardware or self.software) - no fall-through to software.
	if u.Upgrade() {
		t.Error("upgrade must come from the hardware section (false here)")
	}
	if u.CurrentVersion() == nil || *u.CurrentVersion() != "6.2.10.0" {
		t.Errorf("current: %v", u.CurrentVersion())
	}
	if u.LatestVersion() == nil || *u.LatestVersion() != "6.2.10.0" {
		t.Errorf("latest falls back to hardware current: %v", u.LatestVersion())
	}
	if u.ReleaseNotes() == nil || *u.ReleaseNotes() != "hw notes" {
		t.Errorf("notes: %v", u.ReleaseNotes())
	}
	if u.Hardware().DownloadLink() == nil || *u.Hardware().DownloadLink() != "https://x/hw" {
		t.Errorf("download link strip: %v", u.Hardware().DownloadLink())
	}
	if u.Hardware().LatestVersion() == nil || *u.Hardware().LatestVersion() != "6.2.10.0" {
		t.Errorf("hardware latest falls back to current: %v", u.Hardware().LatestVersion())
	}
	if u.Software().LatestVersion() == nil || *u.Software().LatestVersion() != "6.3.0.0" {
		t.Errorf("software latest: %v", u.Software().LatestVersion())
	}

	softwareOnly := NewControllerUpdateInfo(map[string]any{
		"software": map[string]any{"upgrade": true, "latestVersion": "6.3.0.0"},
	})
	if !softwareOnly.Upgrade() {
		t.Error("software-only report must surface the upgrade")
	}

	empty := NewControllerUpdateInfo(map[string]any{})
	if empty.Upgrade() || empty.Update() != nil {
		t.Error("empty update info must report no upgrade")
	}
}

func TestInterfaceDetails(t *testing.T) {
	d := NewInterfaceDetails(map[string]any{"controllerName": "Main"})
	if d.ControllerName() != "Main" {
		t.Errorf("name: %q", d.ControllerName())
	}
	if NewInterfaceDetails(map[string]any{}).ControllerName() != "" {
		t.Error("zero value expected")
	}
}

func TestSite(t *testing.T) {
	s := Site{Name: "Office", ID: "s1"}
	if s.Name != "Office" || s.ID != "s1" {
		t.Error("site fields")
	}
}
