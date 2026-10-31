package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	testControllerID  = "abc"
	testControllerVer = "6.2.10.18"
)

var loginCount int

func writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Errorf("encode response: %v", err)
	}
}

func newTestClient(t *testing.T, handler http.HandlerFunc) *Controller {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/info":
			writeJSON(t, w, map[string]any{"errorCode": 0, "controllerVer": testControllerVer, "omadacId": testControllerID})
		case r.URL.Path == "/"+testControllerID+"/api/v2/login":
			if r.Method == http.MethodPost {
				loginCount++
			}
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"token": "csrf-token"}})
		default:
			handler(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	cl, err := New(srv.URL, "user", "pass")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return cl
}

func TestNewLogsInOnce(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer cl.Close()
	if loginCount != 1 {
		t.Errorf("login calls = %d, want 1", loginCount)
	}
}

func TestClientControllerInfo(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/info" {
			writeJSON(t, w, map[string]any{"errorCode": 0, "controllerVer": testControllerVer, "omadacId": testControllerID})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	info, err := cl.ControllerInfo(context.Background())
	if err != nil {
		t.Fatalf("ControllerInfo() error = %v", err)
	}
	if info.ID() != testControllerID {
		t.Errorf("ControllerInfo().ID() = %q, want %q", info.ID(), testControllerID)
	}
	if info.ControllerVersion() != testControllerVer {
		t.Errorf("ControllerInfo().ControllerVersion() = %q, want %q", info.ControllerVersion(), testControllerVer)
	}
}

func TestSites(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/"+testControllerID+"/api/v2/users/current" {
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
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
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/"+testControllerID+"/api/v2/users/current" {
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
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
	if !errors.Is(err, ErrSiteNotFound) {
		t.Errorf("SiteByName() error = %v, want ErrSiteNotFound", err)
	}
}

func TestSiteEmptyID(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	_, err := cl.Site(context.Background(), "")
	if err == nil {
		t.Fatal("Site() expected error for empty site ID")
	}
	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Errorf("Site() error = %v, want *ValidationError", err)
	}
}

func TestRebootAndShutdown(t *testing.T) {
	var paths []string
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		switch {
		case r.URL.Path == "/"+testControllerID+"/api/v2/cmd/reboot":
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"rebootTime": 300}})
		case r.URL.Path == "/"+testControllerID+"/api/v2/cmd/shutdown":
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"success": true}})
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
	if !found["POST "+"/"+testControllerID+"/api/v2/cmd/reboot"] {
		t.Errorf("missing reboot call, got %v", paths)
	}
	if !found["POST "+"/"+testControllerID+"/api/v2/cmd/shutdown"] {
		t.Errorf("missing shutdown call, got %v", paths)
	}
}

func TestSiteClientValidatesSite(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/"+testControllerID+"/api/v2/sites/site-1":
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"id": "site-1", "name": "Office"}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	sc, err := cl.SiteClient(context.Background(), "site-1")
	if err != nil {
		t.Fatalf("SiteClient() error = %v", err)
	}
	if sc.site != "site-1" {
		t.Errorf("SiteClient site = %q, want %q", sc.site, "site-1")
	}
	sc2, err := cl.SiteClient(context.Background(), "unknown")
	if err == nil {
		t.Fatalf("SiteClient() expected error for unknown site, got %+v", sc2)
	}
}

func TestSetCertificate(t *testing.T) {
	patched := false
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/"+testControllerID+"/api/v2/cmd/uploadCert":
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
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"cerId": "cer-1", "cerName": "cert"}})
		case r.URL.Path == "/"+testControllerID+"/api/v2/controller/setting":
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
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"success": true}})
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
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/"+testControllerID+"/api/v2/cmd/upgradeFirmware" {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode body: %v", err)
				return
			}
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"success": true}})
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
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	if err := cl.InstallControllerFirmware(context.Background(), ""); err == nil {
		t.Fatal("InstallControllerFirmware() expected error for empty target version")
	} else {
		var verr *ValidationError
		if !errors.As(err, &verr) {
			t.Fatalf("InstallControllerFirmware() error = %v, want *ValidationError", err)
		}
	}
}

func TestGetControllerUpgradeStatus(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/"+testControllerID+"/api/v2/maintenance/hardware/upgradeStatus" {
			writeJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
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
