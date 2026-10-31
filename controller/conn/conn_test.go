package conn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	models "github.com/RonaldPhilipsen/omada/controller/models"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type recordedRequest struct {
	Method string
	Path   string
	Req    *http.Request
}

type testServer struct {
	*httptest.Server
	mu       sync.Mutex
	requests []recordedRequest
	handlers map[string]http.HandlerFunc
}

func newTestServer(t *testing.T) *testServer {
	ts := &testServer{handlers: make(map[string]http.HandlerFunc)}
	ts.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts.mu.Lock()
		ts.requests = append(ts.requests, recordedRequest{Method: r.Method, Path: r.URL.Path, Req: r})
		h := ts.handlers[r.Method+" "+r.URL.Path]
		ts.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if h == nil {
			w.WriteHeader(500)
			fmt.Fprint(w, `{"errorCode":70000,"msg":"handler missing"}`)
			return
		}
		h(w, r)
	}))
	t.Cleanup(ts.Close)
	return ts
}

func (ts *testServer) on(spec string, h http.HandlerFunc) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.handlers[spec] = h
}

func (ts *testServer) json(w http.ResponseWriter, status int, v any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (ts *testServer) paths(prefix string) []string {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	var out []string
	for _, r := range ts.requests {
		if strings.HasPrefix(r.Path, prefix) {
			out = append(out, r.Method+" "+r.Path)
		}
	}
	return out
}

func (ts *testServer) countSpec(spec string) int {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	n := 0
	for _, r := range ts.requests {
		if r.Method+" "+r.Path == spec {
			n++
		}
	}
	return n
}

// newTestConn wires a conn against the server and performs an initial login.
func newTestConn(t *testing.T, ts *testServer, nowOffset time.Duration) *conn {
	t.Helper()
	ts.on("GET /api/info", func(w http.ResponseWriter, r *http.Request) {
		ts.json(w, 200, map[string]any{"errorCode": 0, "controllerVer": "6.2.10.18", "omadacId": "abc"})
	})
	var logins atomic.Int32
	ts.on("POST /abc/api/v2/login", func(w http.ResponseWriter, r *http.Request) {
		logins.Add(1)
		ts.json(w, 200, map[string]any{"errorCode": 0, "result": map[string]any{"token": "tok-" + fmt.Sprint(logins.Load())}})
	})
	c, err := newConn(ts.URL, "user", "pass", &http.Client{Jar: nil})
	if err != nil {
		t.Fatal(err)
	}
	c.now = func() time.Time { return time.Now().Add(nowOffset) }
	if _, err := c.login(context.Background()); err != nil {
		t.Fatalf("login: %v", err)
	}
	return c
}

func TestLoginFlow(t *testing.T) {
	ts := newTestServer(t)
	ts.on("GET /api/info", func(w http.ResponseWriter, r *http.Request) {
		ts.json(w, 200, map[string]any{"errorCode": 0, "controllerVer": "6.2.10.18", "omadacId": "abc"})
	})
	var loginCount atomic.Int32
	ts.on("POST /abc/api/v2/login", func(w http.ResponseWriter, r *http.Request) {
		loginCount.Add(1)
		body, _ := io.ReadAll(r.Body)
		var sent map[string]any
		_ = json.Unmarshal(body, &sent)
		if sent["username"] != "user" || sent["password"] != "pass" {
			t.Errorf("login body: %s", body)
		}
		ts.json(w, 200, map[string]any{"errorCode": 0, "result": map[string]any{"token": "tok1"}})
	})
	ts.on("GET /abc/api/v2/loginStatus", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Csrf-Token"); got != "tok1" {
			t.Errorf("Csrf-Token: %q", got)
		}
		if got := r.Header.Get("Omada-Request-Source"); got != "web-local" {
			t.Errorf("Omada-Request-Source: %q", got)
		}
		if got := r.Header.Get("Referer"); got != ts.URL+"/" {
			t.Errorf("Referer: %q", got)
		}
		if got := r.Header.Get("Origin"); got != ts.URL {
			t.Errorf("Origin: %q", got)
		}
		ts.json(w, 200, map[string]any{"errorCode": 0, "login": true})
	})

	c, err := newConn(ts.URL, "user", "pass", &http.Client{})
	if err != nil {
		t.Fatal(err)
	}
	id, err := c.login(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if id != "abc" {
		t.Fatalf("id = %q", id)
	}
	if c.csrf != "tok1" {
		t.Errorf("csrf = %q", c.csrf)
	}
	if _, err := c.RequestEndpoint(context.Background(), "GET", "v2", "loginStatus", "", nil, nil); err != nil {
		t.Fatalf("request: %v", err)
	}
	if got := ts.countSpec("POST /abc/api/v2/login"); got != 1 {
		t.Errorf("login count = %d", got)
	}
}

func TestLoginRejectsOldVersion(t *testing.T) {
	ts := newTestServer(t)
	ts.on("GET /api/info", func(w http.ResponseWriter, r *http.Request) {
		ts.json(w, 200, map[string]any{"errorCode": 0, "controllerVer": "5.0.15.2", "omadacId": "abc"})
	})
	c, err := newConn(ts.URL, "user", "pass", &http.Client{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.login(context.Background()); !errors.Is(err, models.ErrUnsupportedControllerVersion) {
		t.Errorf("got %v, want models.ErrUnsupportedControllerVersion", err)
	}
}

func TestLoginVersionBoundary(t *testing.T) {
	ts := newTestServer(t)
	ts.on("GET /api/info", func(w http.ResponseWriter, r *http.Request) {
		ts.json(w, 200, map[string]any{"errorCode": 0, "controllerVer": "5.1.0", "omadacId": "abc"})
	})
	ts.on("POST /abc/api/v2/login", func(w http.ResponseWriter, r *http.Request) {
		ts.json(w, 200, map[string]any{"errorCode": 0, "result": map[string]any{"token": "t"}})
	})
	c, err := newConn(ts.URL, "user", "pass", &http.Client{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.login(context.Background()); err != nil {
		t.Errorf("5.1.0 must be accepted, got %v", err)
	}
}

func TestReLoginOnLoginFailedThenRetry(t *testing.T) {
	ts := newTestServer(t)
	var devicesCalls atomic.Int32
	ts.on("GET /abc/api/v2/devices", func(w http.ResponseWriter, r *http.Request) {
		if devicesCalls.Add(1) == 1 {
			ts.json(w, 200, map[string]any{"errorCode": -30109, "msg": "session expired"})
			return
		}
		ts.json(w, 200, map[string]any{"errorCode": 0, "result": map[string]any{"ok": true}})
	})
	c := newTestConn(t, ts, 0)

	res, err := c.RequestEndpoint(context.Background(), "GET", "v2", "devices", "", nil, nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if res["ok"] != true {
		t.Errorf("result: %v", res)
	}
	if got := ts.countSpec("POST /abc/api/v2/login"); got != 2 {
		t.Errorf("login count = %d, want 2", got)
	}
	if got := devicesCalls.Load(); got != 2 {
		t.Errorf("devices calls = %d, want 2", got)
	}
}

func TestNonJSON200IsSessionClosed(t *testing.T) {
	ts := newTestServer(t)
	ts.on("GET /abc/api/v2/devices", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(200)
		fmt.Fprint(w, "<html>login</html>")
	})
	c := newTestConn(t, ts, 0)

	// First request fails with session closed; re-login happens; the retry
	// hits the same handler, which keeps returning HTML, so the second
	// attempt must surface models.ErrLoginSessionClosed.
	if _, err := c.RequestEndpoint(context.Background(), "GET", "v2", "devices", "", nil, nil); !errors.Is(err, models.ErrLoginSessionClosed) {
		t.Errorf("got %v, want models.ErrLoginSessionClosed", err)
	}
}

func TestApplicationErrorCodeNoRelogin(t *testing.T) {
	ts := newTestServer(t)
	ts.on("GET /abc/api/v2/devices", func(w http.ResponseWriter, r *http.Request) {
		ts.json(w, 200, map[string]any{"errorCode": 8101, "msg": "no permission"})
	})
	c := newTestConn(t, ts, 0)

	_, err := c.RequestEndpoint(context.Background(), "GET", "v2", "devices", "", nil, nil)
	var re *models.RequestError
	if !errors.As(err, &re) || re.Code != 8101 {
		t.Fatalf("got %v, want models.RequestError 8101", err)
	}
	if got := ts.countSpec("POST /abc/api/v2/login"); got != 1 {
		t.Errorf("login count = %d, application errors must not re-login", got)
	}
}

func TestCheckLoginTransientErrorPropagates(t *testing.T) {
	ts := newTestServer(t)
	ts.on("GET /abc/api/v2/loginStatus", func(w http.ResponseWriter, r *http.Request) {
		ts.json(w, 500, map[string]any{"errorCode": 500, "msg": "boom"})
	})
	// >1h since logon so checkLogin actually calls loginStatus
	c := newTestConn(t, ts, 0)
	c.lastLogon = c.now().Add(-2 * time.Hour)

	_, err := c.RequestEndpoint(context.Background(), "GET", "v2", "devices", "", nil, nil)
	var re *models.RequestError
	if !errors.As(err, &re) || re.Code != 500 {
		t.Fatalf("got %v, want models.RequestError 500", err)
	}
	// No re-login attempt after a transient checkLogin failure.
	if got := ts.countSpec("POST /abc/api/v2/login"); got != 1 {
		t.Errorf("login count = %d, transient errors must not re-login", got)
	}
}

func TestConcurrentFirstRequestsNoRace(t *testing.T) {
	ts := newTestServer(t)
	ts.on("GET /api/info", func(w http.ResponseWriter, r *http.Request) {
		ts.json(w, 200, map[string]any{"errorCode": 0, "controllerVer": "6.2.10.18", "omadacId": "abc"})
	})
	ts.on("POST /abc/api/v2/login", func(w http.ResponseWriter, r *http.Request) {
		ts.json(w, 200, map[string]any{"errorCode": 0, "result": map[string]any{"token": "tok"}})
	})
	ts.on("GET /abc/api/v2/loginStatus", func(w http.ResponseWriter, r *http.Request) {
		ts.json(w, 200, map[string]any{"errorCode": 0, "login": true})
	})

	c, err := newConn(ts.URL, "user", "pass", &http.Client{})
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.RequestEndpoint(context.Background(), "GET", "v2", "loginStatus", "", nil, nil); err != nil {
				t.Errorf("concurrent request: %v", err)
			}
		}()
	}
	wg.Wait()
}

func TestControllerInfoUnauthenticated(t *testing.T) {
	ts := newTestServer(t)
	ts.on("GET /api/info", func(w http.ResponseWriter, r *http.Request) {
		ts.json(w, 200, map[string]any{"errorCode": 0, "controllerVer": "6.2.10.18", "omadacId": "abc"})
	})
	c, err := newConn(ts.URL, "user", "pass", &http.Client{})
	if err != nil {
		t.Fatal(err)
	}
	info, err := c.ControllerInfo(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if info["omadacId"] != "abc" {
		t.Errorf("info: %v", info)
	}
	v, err := c.ControllerVersion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	min := models.ParseVersionMust("5.1.0")
	if v.Compare(min) < 0 {
		t.Errorf("version %v below minimum", v)
	}
}
