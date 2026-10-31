package conn

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// pagedHandler serves a sequence of pages from the given pages, recording the
// page number and (for POST) the body seen on each call.
type pagedHandler struct {
	mu       sync.Mutex
	pages    []map[string]any
	calls    []pageCall
	bodyKey  string // "GET" style uses query page; set "" for query, "body" for POST
	lastBody map[string]any
}

type pageCall struct {
	Page  int
	Body  map[string]any
	Query string
}

func (h *pagedHandler) serve(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()
	var page int
	if r.Method == http.MethodPost {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		h.lastBody = body
		page = asIntAny(body["page"])
	} else {
		h.lastBody = map[string]any{}
		v := r.URL.Query().Get("page")
		if v == "" {
			v = r.URL.Query().Get("currentPage")
		}
		page, _ = strconv.Atoi(v)
	}
	call := pageCall{Page: page, Body: h.lastBody, Query: r.URL.RawQuery}
	h.calls = append(h.calls, call)

	idx := page - 1
	if idx < 0 || idx >= len(h.pages) {
		http.Error(w, "page out of range", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.pages[idx])
}

func asIntAny(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return 0
}

func setupPaged(t *testing.T, method, path string, pages []map[string]any) (*testServer, *conn, *pagedHandler) {
	t.Helper()
	ts := newTestServer(t)
	h := &pagedHandler{pages: pages}
	ts.on(method+" "+path, h.serve)
	c := newTestConn(t, ts, 0)
	return ts, c, h
}

func TestPagesV2CollectsAllPages(t *testing.T) {
	// 250 rows over pages of 100: 100 + 100 + 50. The partial last page
	// triggers one more (empty) fetch, where the unbounded upstream loop would
	// spin forever; the bounded collector stops there.
	empty := map[string]any{"errorCode": 0, "data": []any{}, "currentSize": 0, "totalRows": 250}
	_, c, h := setupPaged(t, "GET", "/abc/api/v2/devices", []map[string]any{
		map[string]any{"errorCode": 0, "data": data100(), "currentSize": 100, "totalRows": 250},
		map[string]any{"errorCode": 0, "data": data100(), "currentSize": 100, "totalRows": 250},
		page50(),
		empty,
	})

	got, err := c.PagesV2(context.Background(), "devices", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 250 {
		t.Errorf("entries = %d, want 250", len(got))
	}
	if len(h.calls) != 4 {
		t.Errorf("calls = %d, want 4 (incl. the empty final page)", len(h.calls))
	}
	seen := map[int]bool{}
	for _, call := range h.calls {
		seen[call.Page] = true
		if !strings.Contains(call.Query, "currentPageSize=100") {
			t.Errorf("query missing currentPageSize: %s", call.Query)
		}
	}
	if !seen[1] || !seen[2] || !seen[3] {
		t.Errorf("pages seen: %v", seen)
	}
}

func (h *pagedHandler) pages0Data(n int) []map[string]any {
	data := make([]map[string]any, n)
	for i := range data {
		data[i] = map[string]any{"id": i}
	}
	return data
}

func TestPagesV2StopsOnTotalRows(t *testing.T) {
	_, c, h := setupPaged(t, "GET", "/abc/api/v2/devices", []map[string]any{page100(100)})

	got, err := c.PagesV2(context.Background(), "devices", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 100 {
		t.Errorf("entries = %d, want 100", len(got))
	}
	if len(h.calls) != 1 {
		t.Errorf("calls = %d, want 1", len(h.calls))
	}
}

func TestPagesV2BoundedOnEmptyPage(t *testing.T) {
	_, c, h := setupPaged(t, "GET", "/abc/api/v2/devices", []map[string]any{
		{"errorCode": 0, "data": data100(), "currentSize": 100, "totalRows": 5000},
		{"errorCode": 0, "data": []any{}, "currentSize": 0, "totalRows": 5000},
	})

	got, err := c.PagesV2(context.Background(), "devices", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 100 {
		t.Errorf("entries = %d, want 100", len(got))
	}
	if len(h.calls) > 2 {
		t.Errorf("calls = %d, bounded loop must stop after the empty page", len(h.calls))
	}
}

func TestPagesOpenAPIPostCollectsAllPages(t *testing.T) {
	_, c, h := setupPaged(t, "POST", "/openapi/v1/abc/lan-networks", []map[string]any{
		map[string]any{"errorCode": 0, "data": data100(), "currentSize": 100, "totalRows": 140},
		page40(),
		map[string]any{"errorCode": 0, "data": []any{}, "currentSize": 0, "totalRows": 140},
	})

	got, err := c.PagesOpenAPIPost(context.Background(), "v1", "lan-networks", "", map[string]any{"type": "lan"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 140 {
		t.Errorf("entries = %d, want 140", len(got))
	}
	for i, call := range h.calls {
		if call.Body["page"] != float64(i+1) {
			t.Errorf("call %d body page = %v", i, call.Body["page"])
		}
		if call.Body["pageSize"] != float64(100) {
			t.Errorf("call %d body pageSize = %v", i, call.Body["pageSize"])
		}
		if call.Body["type"] != "lan" {
			t.Errorf("call %d lost original body keys: %v", i, call.Body)
		}
	}
}

func TestPagesOpenAPIGetCollectsAllPages(t *testing.T) {
	_, c, h := setupPaged(t, "GET", "/openapi/v1/abc/devices", []map[string]any{page100(100)})

	got, err := c.PagesOpenAPIGet(context.Background(), "v1", "devices", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 100 {
		t.Errorf("entries = %d, want 100", len(got))
	}
	if len(h.calls) != 1 {
		t.Errorf("calls = %d, want 1", len(h.calls))
	}
}

func TestPagesHardCap(t *testing.T) {
	// totalRows is so large the totalRows condition never trips; the hard cap
	// of 1000 pages must end the loop.
	pages := make([]map[string]any, 1001)
	for i := range pages {
		pages[i] = map[string]any{"errorCode": 0, "data": data100(), "currentSize": 100, "totalRows": 1000000000}
	}
	_, c, h := setupPaged(t, "GET", "/abc/api/v2/devices", pages)

	got, err := c.PagesV2(context.Background(), "devices", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1000*100 {
		t.Errorf("entries = %d, want %d", len(got), 1000*100)
	}
	if len(h.calls) != 1000 {
		t.Errorf("calls = %d, want the hard cap of 1000", len(h.calls))
	}
}

func data100() []any {
	data := make([]any, 100)
	for i := range data {
		data[i] = map[string]any{"id": i}
	}
	return data
}

func page50() map[string]any {
	return map[string]any{
		"errorCode":   0,
		"data":        dataN(50),
		"currentSize": 50,
		"totalRows":   250,
	}
}

func page40() map[string]any {
	return map[string]any{
		"errorCode":   0,
		"data":        dataN(40),
		"currentSize": 40,
		"totalRows":   140,
	}
}

func dataN(n int) []any {
	data := make([]any, n)
	for i := range data {
		data[i] = map[string]any{"id": i}
	}
	return data
}

func page100(n int) map[string]any {
	data := dataN(n)
	return map[string]any{"errorCode": 0, "data": data, "currentSize": n, "totalRows": n}
}
