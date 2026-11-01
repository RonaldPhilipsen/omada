package controller_test

import (
	"context"
	"errors"
	models "github.com/RonaldPhilipsen/omada/controller/models"
	"net/http"
	"strings"
	"testing"

	"encoding/json"

	"github.com/RonaldPhilipsen/omada/controller/internal/testutil"
)

func TestNewLogsInOnce(t *testing.T) {
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer cl.Close()
	if testutil.LoginCount() != 1 {
		t.Errorf("login calls = %d, want 1", testutil.LoginCount())
	}
}

func TestOmadacID(t *testing.T) {
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer cl.Close()
	if got := cl.OmadacID(); got != testutil.TestControllerID {
		t.Errorf("OmadacID() = %q, want %q", got, testutil.TestControllerID)
	}
}

func TestClientControllerInfo(t *testing.T) {
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/info" {
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "controllerVer": testutil.TestControllerVer, "omadacId": testutil.TestControllerID})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	info, err := cl.ControllerInfo(context.Background())
	if err != nil {
		t.Fatalf("models.ControllerInfo() error = %v", err)
	}
	if info.ID() != testutil.TestControllerID {
		t.Errorf("models.ControllerInfo().ID() = %q, want %q", info.ID(), testutil.TestControllerID)
	}
	if info.ControllerVersion() != testutil.TestControllerVer {
		t.Errorf("models.ControllerInfo().ControllerVersion() = %q, want %q", info.ControllerVersion(), testutil.TestControllerVer)
	}
}

func TestSites(t *testing.T) {
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/"+testutil.TestControllerID+"/api/v2/users/current" {
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"privilege": map[string]any{
					"sites": []map[string]any{
						{"name": "Office", "key": "site-1"},
						{"name": "Home", "key": "site-2"},
					},
				},
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sites, err := cl.Sites(context.Background())
	if err != nil {
		t.Fatalf("Sites() error = %v", err)
	}
	if len(sites) != 2 {
		t.Fatalf("Sites() returned %d sites, want 2: %v", len(sites), sites)
	}
	if sites[0].Name != "Office" || sites[0].ID != "site-1" {
		t.Errorf("Sites()[0] = %+v, want Office/site-1", sites[0])
	}
	if sites[1].Name != "Home" || sites[1].ID != "site-2" {
		t.Errorf("Sites()[1] = %+v, want Home/site-2", sites[1])
	}
}

func TestSiteByNameMiss(t *testing.T) {
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/"+testutil.TestControllerID+"/api/v2/users/current" {
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"privilege": map[string]any{
					"sites": []map[string]any{
						{"name": "Home", "key": "site-2"},
					},
				},
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	site, err := cl.SiteByName(context.Background(), "Missing")
	if err == nil {
		t.Fatalf("SiteByName() expected error, got site %+v", site)
	}
	if !errors.Is(err, models.ErrSiteNotFound) {
		t.Errorf("SiteByName() error = %v, want models.ErrSiteNotFound", err)
	}
}

func TestSiteEmptyID(t *testing.T) {
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	_, err := cl.Site(context.Background(), "")
	if err == nil {
		t.Fatal("models.Site() expected error for empty site ID")
	}
	var verr *models.ValidationError
	if !errors.As(err, &verr) {
		t.Errorf("models.Site() error = %v, want *models.ValidationError", err)
	}
}

func TestRebootAndShutdown(t *testing.T) {
	var paths []string
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		switch {
		case r.URL.Path == "/"+testutil.TestControllerID+"/api/v2/cmd/reboot":
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"rebootTime": 300}})
		case r.URL.Path == "/"+testutil.TestControllerID+"/api/v2/cmd/shutdown":
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"success": true}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	if err := cl.Reboot(context.Background()); err != nil {
		t.Fatalf("Reboot() error = %v", err)
	}
	if err := cl.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	found := map[string]bool{}
	for _, p := range paths {
		found[p] = true
	}
	if !found["POST "+"/"+testutil.TestControllerID+"/api/v2/cmd/reboot"] {
		t.Errorf("missing reboot call, got %v", paths)
	}
	if !found["POST "+"/"+testutil.TestControllerID+"/api/v2/cmd/shutdown"] {
		t.Errorf("missing shutdown call, got %v", paths)
	}
}

func TestSiteClientValidatesSite(t *testing.T) {
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/"+testutil.TestControllerID+"/api/v2/sites/site-1":
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"id": "site-1", "name": "Office"}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	if _, err := cl.SiteClient(context.Background(), "site-1"); err != nil {
		t.Fatalf("SiteClient() error = %v", err)
	}
	sc2, err := cl.SiteClient(context.Background(), "unknown")
	if err == nil {
		t.Fatalf("SiteClient() expected error for unknown site, got %+v", sc2)
	}
}

func TestSetCertificate(t *testing.T) {
	patched := false
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/"+testutil.TestControllerID+"/api/v2/cmd/uploadCert":
			if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "multipart/form-data") {
				t.Errorf("upload content type = %q, want multipart/form-data", got)
			}
			if err := r.ParseMultipartForm(32 << 20); err != nil {
				t.Errorf("parse multipart form: %v", err)
				return
			}
			if got := r.FormValue("key"); got != "key-pem" {
				t.Errorf("upload key = %q, want %q", got, "key-pem")
			}
			if got := r.FormValue("cert"); got != "cert-pem" {
				t.Errorf("upload cert = %q, want %q", got, "cert-pem")
			}
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"cerId": "cer-1", "cerName": "cert"}})
		case r.URL.Path == "/"+testutil.TestControllerID+"/api/v2/controller/setting":
			patched = true
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode patch body: %v", err)
				return
			}
			cert, ok := body["certificate"].(map[string]any)
			if !ok {
				t.Fatalf("patch body missing certificate: %v", body)
			}
			if cert["cerType"] != "PFX" {
				t.Errorf("patch certificate cerType = %v, want PFX", cert["cerType"])
			}
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"success": true}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	if err := cl.SetCertificate(context.Background(), "key-pem", "cert-pem"); err != nil {
		t.Fatalf("SetCertificate() error = %v", err)
	}
	if !patched {
		t.Error("controller/setting was not patched after upload")
	}
}

func TestInstallControllerFirmware(t *testing.T) {
	var body map[string]any
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/"+testutil.TestControllerID+"/api/v2/cmd/upgradeFirmware" {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode body: %v", err)
				return
			}
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"success": true}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	if err := cl.InstallControllerFirmware(context.Background(), "6.4.0.1234"); err != nil {
		t.Fatalf("InstallControllerFirmware() error = %v", err)
	}
	if body["targetVersion"] != "6.4.0.1234" {
		t.Errorf("InstallControllerFirmware body targetVersion = %v, want 6.4.0.1234", body["targetVersion"])
	}
}

func TestInstallControllerFirmwareEmptyVersion(t *testing.T) {
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	if err := cl.InstallControllerFirmware(context.Background(), ""); err == nil {
		t.Fatal("InstallControllerFirmware() expected error for empty target version")
	} else {
		var verr *models.ValidationError
		if !errors.As(err, &verr) {
			t.Fatalf("InstallControllerFirmware() error = %v, want *models.ValidationError", err)
		}
	}
}

func TestGetControllerUpgradeStatus(t *testing.T) {
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/"+testutil.TestControllerID+"/api/v2/maintenance/hardware/upgradeStatus" {
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"upgradeStatus": 3,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	status, err := cl.GetControllerUpgradeStatus(context.Background())
	if err != nil {
		t.Fatalf("GetControllerUpgradeStatus() error = %v", err)
	}
	if status.UpgradeStatus() != 3 {
		t.Errorf("GetControllerUpgradeStatus() UpgradeStatus = %d, want 3", status.UpgradeStatus())
	}
}
