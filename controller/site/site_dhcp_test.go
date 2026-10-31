package site_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/RonaldPhilipsen/omada/controller/internal/testutil"
	models "github.com/RonaldPhilipsen/omada/controller/models"
	"github.com/RonaldPhilipsen/omada/controller/site"
	"net/http"
	"testing"
)

func TestDhcpReservations(t *testing.T) {
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/"+testutil.TestControllerID+"/api/v2/sites/site-1":
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"id": "site-1", "name": "Office", "controllerVer": "6.2.10.18"}})
			return
		case r.Method == http.MethodGet && r.URL.Path == "/openapi/v1/"+testutil.TestControllerID+"/sites/site-1/setting/service/dhcp":
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data": []map[string]any{
					{"id": "res-1", "mac": "DE:AD:BE:EF:99:99", "ip": "192.168.10.50", "description": "Printer", "status": true, "netId": "net-lan-1"},
				},
				"currentSize": 1,
				"totalRows":   1,
			}})
			return
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	sc := mustSiteClient(t, cl)
	reservations, err := sc.DhcpReservations(context.Background())
	if err != nil {
		t.Fatalf("DhcpReservations() error = %v", err)
	}
	if len(reservations) != 1 {
		t.Fatalf("DhcpReservations() returned %d reservations, want 1", len(reservations))
	}
	if reservations[0].MAC() != "DE:AD:BE:EF:99:99" || reservations[0].IP() != "192.168.10.50" {
		t.Errorf("DhcpReservations()[0] = %+v, want DE:AD:BE:EF:99:99/192.168.10.50", reservations[0])
	}
}

func TestDhcpReservationsAllPages(t *testing.T) {
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/"+testutil.TestControllerID+"/api/v2/sites/site-1":
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"id": "site-1", "name": "Office", "controllerVer": "6.2.10.18"}})
			return
		case r.Method == http.MethodGet && r.URL.Path == "/openapi/v1/"+testutil.TestControllerID+"/sites/site-1/setting/service/dhcp":
			var data []map[string]any
			if r.URL.Query().Get("page") == "2" {
				data = []map[string]any{{"id": "res-2", "mac": "DE:AD:BE:EF:88:88", "ip": "192.168.10.51"}}
			} else {
				data = []map[string]any{{"id": "res-1", "mac": "DE:AD:BE:EF:99:99", "ip": "192.168.10.50"}}
			}
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data":        data,
				"currentSize": 1,
				"totalRows":   2,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := mustSiteClient(t, cl)
	reservations, err := sc.DhcpReservations(context.Background())
	if err != nil {
		t.Fatalf("DhcpReservations() error = %v", err)
	}
	if len(reservations) != 2 {
		t.Fatalf("DhcpReservations() returned %d reservations, want 2", len(reservations))
	}
}

func TestDhcpReservationsVersionGate(t *testing.T) {
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/"+testutil.TestControllerID+"/api/v2/sites/site-1":
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"id": "site-1", "name": "Office", "controllerVer": "6.1.0.5"}})
			return
		case r.Method == http.MethodGet && r.URL.Path == "/"+testutil.TestControllerID+"/api/v2/sites/site-1/setting/service/dhcp":
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data":        []map[string]any{{"id": "res-1", "mac": "DE:AD:BE:EF:99:99", "ip": "192.168.10.50"}},
				"currentSize": 1,
				"totalRows":   1,
			}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	sc := mustSiteClient(t, cl)
	reservations, err := sc.DhcpReservations(context.Background())
	if err != nil {
		t.Fatalf("DhcpReservations() error = %v", err)
	}
	if len(reservations) != 1 {
		t.Fatalf("DhcpReservations() returned %d reservations, want 1", len(reservations))
	}
}

func TestDhcpReservationMiss(t *testing.T) {
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/"+testutil.TestControllerID+"/api/v2/sites/site-1":
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"id": "site-1", "name": "Office", "controllerVer": "6.2.10.18"}})
			return
		case r.Method == http.MethodGet && r.URL.Path == "/openapi/v1/"+testutil.TestControllerID+"/sites/site-1/setting/service/dhcp":
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data":        []map[string]any{{"id": "res-1", "mac": "DE:AD:BE:EF:99:99", "ip": "192.168.10.50"}},
				"currentSize": 1,
				"totalRows":   1,
			}})
			return
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	sc := mustSiteClient(t, cl)
	if _, err := sc.DhcpReservation(context.Background(), "192.168.10.99"); err == nil {
		t.Fatal("models.DhcpReservation() expected error for unknown reservation IP")
	} else {
		var verr *models.ValidationError
		if !errors.As(err, &verr) {
			t.Fatalf("models.DhcpReservation() error = %v, want *models.ValidationError", err)
		}
	}
}

func TestCreateDhcpReservationValidation(t *testing.T) {
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })
	sc := mustSiteClient(t, cl)
	if _, err := sc.CreateDhcpReservation(context.Background(), "", "192.168.10.5", "DE:AD:BE:EF:99:99"); err == nil {
		t.Fatal("CreateDhcpReservation() expected error for empty host")
	} else {
		var verr *models.ValidationError
		if !errors.As(err, &verr) {
			t.Fatalf("CreateDhcpReservation() error = %v, want *models.ValidationError", err)
		}
	}
	if _, err := sc.CreateDhcpReservation(context.Background(), "Printer", "", "DE:AD:BE:EF:99:99"); err == nil {
		t.Fatal("CreateDhcpReservation() expected error for empty IP")
	} else {
		var verr *models.ValidationError
		if !errors.As(err, &verr) {
			t.Fatalf("CreateDhcpReservation() error = %v, want *models.ValidationError", err)
		}
	}
	if _, err := sc.CreateDhcpReservation(context.Background(), "Printer", "192.168.10.5", ""); err == nil {
		t.Fatal("CreateDhcpReservation() expected error for empty MAC")
	} else {
		var verr *models.ValidationError
		if !errors.As(err, &verr) {
			t.Fatalf("CreateDhcpReservation() error = %v, want *models.ValidationError", err)
		}
	}
}

func TestCreateDhcpReservation(t *testing.T) {
	var bodies []map[string]any
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/openapi/v1/"+testutil.TestControllerID+"/sites/site-1/setting/service/dhcp" {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode body: %v", err)
				return
			}
			bodies = append(bodies, body)
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"id": "res-3", "mac": "DE:AD:BE:EF:77:77", "ip": "192.168.10.52", "status": true,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := mustSiteClient(t, cl)
	reservation, err := sc.CreateDhcpReservation(context.Background(), "Printer", "192.168.10.52", "DE:AD:BE:EF:77:77")
	if err != nil {
		t.Fatalf("CreateDhcpReservation() error = %v", err)
	}
	if len(bodies) != 1 {
		t.Fatalf("create calls = %d, want 1", len(bodies))
	}
	if bodies[0]["hostName"] != "Printer" || bodies[0]["ip"] != "192.168.10.52" || bodies[0]["mac"] != "DE:AD:BE:EF:77:77" {
		t.Errorf("create body = %v, want hostName Printer ip 192.168.10.52 mac DE:AD:BE:EF:77:77", bodies[0])
	}
	if reservation.IP() != "192.168.10.52" {
		t.Errorf("CreateDhcpReservation() reservation IP = %q, want 192.168.10.52", reservation.IP())
	}
}

func TestUpdateDhcpReservation(t *testing.T) {
	var patched map[string]any
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch && r.URL.Path == "/openapi/v1/"+testutil.TestControllerID+"/sites/site-1/setting/service/dhcp/de:ad:be:ef:99:99" {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode body: %v", err)
				return
			}
			patched = body
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"id": "res-1", "mac": "DE:AD:BE:EF:99:99", "ip": "192.168.10.60", "status": true,
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	sc := mustSiteClient(t, cl)
	mac := "DE:AD:BE:EF:99:99"
	if _, err := sc.UpdateDhcpReservation(context.Background(), site.DhcpReservationUpdate{IP: "192.168.10.60", MAC: &mac}); err != nil {
		t.Fatalf("UpdateDhcpReservation() error = %v", err)
	}
	if patched == nil {
		t.Fatal("UpdateDhcpReservation() did not call the update endpoint")
	}
	if patched["ip"] != "192.168.10.60" {
		t.Errorf("UpdateDhcpReservation body = %v, want ip 192.168.10.60", patched)
	}
	if _, err := sc.UpdateDhcpReservation(context.Background(), site.DhcpReservationUpdate{}); err == nil {
		t.Fatal("UpdateDhcpReservation() expected error for empty update")
	} else {
		var verr *models.ValidationError
		if !errors.As(err, &verr) {
			t.Fatalf("UpdateDhcpReservation() error = %v, want *models.ValidationError", err)
		}
	}
}

func TestDeleteDhcpReservation(t *testing.T) {
	var deletes int
	cl := testutil.NewTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/"+testutil.TestControllerID+"/api/v2/sites/site-1":
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"id": "site-1", "name": "Office", "controllerVer": "6.2.10.18"}})
			return
		case r.Method == http.MethodGet && r.URL.Path == "/openapi/v1/"+testutil.TestControllerID+"/sites/site-1/setting/service/dhcp":
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{
				"data":        []map[string]any{{"id": "res-1", "mac": "DE:AD:BE:EF:99:99", "ip": "192.168.10.50"}},
				"currentSize": 1,
				"totalRows":   1,
			}})
		case r.Method == http.MethodDelete && r.URL.Path == "/openapi/v1/"+testutil.TestControllerID+"/sites/site-1/setting/service/dhcp/de:ad:be:ef:99:99":
			deletes++
			testutil.WriteJSON(t, w, map[string]any{"errorCode": 0, "result": map[string]any{"success": true}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	sc := mustSiteClient(t, cl)
	if err := sc.DeleteDhcpReservation(context.Background(), "192.168.10.50"); err != nil {
		t.Fatalf("DeleteDhcpReservation() error = %v", err)
	}
	if deletes != 1 {
		t.Errorf("DeleteDhcpReservation calls = %d, want 1", deletes)
	}
	if err := sc.DeleteDhcpReservation(context.Background(), ""); err == nil {
		t.Fatal("DeleteDhcpReservation() expected error for empty IP")
	} else {
		var verr *models.ValidationError
		if !errors.As(err, &verr) {
			t.Fatalf("DeleteDhcpReservation() error = %v, want *models.ValidationError", err)
		}
	}
}
