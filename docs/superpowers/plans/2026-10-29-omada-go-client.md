# Omada Go Client (Phase 1: Controller) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A stdlib-only Go package `github.com/RonaldPhilipsen/omada/controller` that is a faithful, synchronous port of the Python library `tplink_omada_client` v1.5.14 (controller API only), with three documented upstream-defect fixes.

**Architecture:** One `controller` package. An unexported `conn` type owns the HTTP session, cookie jar (IP-host-accepting), login state, response unwrapping, auto re-login, and pagination. `Client` (controller level) and `SiteClient` (site level) are thin synchronous facades over `conn`. All API data arrives as `map[string]any` from the unwrapper; model structs wrap that map and expose typed accessors with `Raw()` returning the original.

**Tech Stack:** Go 1.24+ (toolchain here is 1.26.2), stdlib only (`net/http`, `testing`, `httptest`, `sync`, `context`, `mime/multipart`). No third-party dependencies.

**Spec:** `docs/superpowers/specs/2026-10-29-omada-go-client-design.md` (approved 2026-10-29, includes five builder-check decisions).

**Reference:** upstream Python source is checked out at `/tmp/opencode/tplink-omada-api` (tag `release/v1.5.14`). Every porting task names the upstream function(s) it ports; where this plan and upstream agree, upstream is the source of truth for payload keys, URLs, and edge behavior. Re-clone if missing: `git clone --depth 1 --branch release/v1.5.14 https://github.com/MarkGodwin/tplink-omada-api /tmp/opencode/tplink-omada-api`.

## Global Constraints

- Module path: `github.com/RonaldPhilipsen/omada`; single package `controller/` plus root `go.mod`, `README.md`, `LICENSE`.
- Stdlib only. No third-party imports anywhere.
- Synchronous API. Every public method takes `ctx context.Context` as its first argument.
- Paged endpoints return fully collected `[]T` slices (no iterators).
- JSON null/optional = pointer fields. Models keep raw JSON internally, exposed via `Raw() map[string]any`.
- Unknown enum values map to the enum's `Unknown`/`None` member (mirrors Python `_missing_`).
- Concurrency: `Client`/`SiteClient` are safe for concurrent use; a mutex serializes login/re-login. Tests run with `-race`.
- Defect fixes (port the behavior, fix these three, document in README):
  1. Pagination is bounded: stop when a page is empty, when `page*size >= totalRows`, and at a hard cap of 1000 pages (upstream loops forever on an empty page with `totalRows > 0`).
  2. `VpnPolicies` fetches all pages of each category list (upstream fetched only page 1 of 100).
  3. Re-login triggers only on auth-class failures (`-30109` / non-JSON 200). Transient `loginStatus` errors propagate instead of forcing a re-login (upstream `_check_login` swallowed everything).
- `go vet ./...` clean, `gofmt -l .` empty, `go test ./...` green (plus `-race` from Task 3 onward) - the completion bar.
- Commit after every task. Messages: `feat(controller): ...`, `docs: ...`.

## Review Focus

Inputs the spec implies but that a naive port would silently break; each line is pinned by a test in the owning task.

1. **IP-address controller URLs** (`New("192.168.1.10", ...)`) - scheme auto-prepended, and the cookie jar must store/return the session cookie for an IP host (Go's default policy would reject it), or login "succeeds" but every later request 401s. -> Task 2, Task 3.
2. **Concurrent first-request + re-login** - two goroutines hitting `request()` before any login must not race on `csrf`/`lastLogon` and must not double-leak state; `-race` must be clean. -> Task 3.
3. **Sparse/missing JSON fields** - gateway without `portConfigs`, AP without `deviceMisc`, client with `block` (not `blocked`), switch with `portNum` nested under `deviceMisc`: model accessors must return zero values/nil, never panic. -> Tasks 7-8.
4. **VLAN write acknowledged, read-back fails** - network drops after the 200: error text must carry "write may have succeeded; inspect controller state before retrying" and the code must NOT auto-retry the write. -> Task 15.
5. **Version-gate boundaries** - `6` vs `5.20` and `6.2.0.0` vs `6.3` must compare numerically per segment, never lexicographically ("5.20" > "5.9" string-wise); endpoint selection flips exactly at `6`, `6.2.0.0`, `6.3`. -> Task 1, Task 9, Task 10.

## File Structure

```
omada/
├── go.mod                    module github.com/RonaldPhilipsen/omada (go 1.24)
├── LICENSE                   MIT, attribution to Mark Godwin / tplink-omada-api
├── README.md                 written in Task 17
└── controller/
    ├── version.go            Version type + ParseVersion + Compare
    ├── errors.go             RequestError, ValidationError, sentinels
    ├── cookiejar.go          permissiveJar (accepts IP hosts)
    ├── connection.go         conn: newConn, URL formatting, login, checkLogin,
    │                         request (auto re-login), do (unwrap), checkAppErrors
    ├── pagination.go         pagesV2, pagesOpenAPIPost, pagesOpenAPIGet
    ├── enums.go              all int enums + RadioId key maps + VpnType/VpnCategory
    ├── models_controller.go  ControllerInfo/Type/Status, ControllerUpdateInfo,
    │                         UpdateSection, HardwareUpgradeStatus, InterfaceDetails, Site
    ├── models_device.go      Device, Link/Uplink/Downlink
    ├── models_client.go      Client (flat), WiredDetails, WirelessDetails,
    │                         IpSetting, RateLimit
    ├── models_topology.go    Topology, TopologyNode, TopologyWanPort
    ├── models_switch.go      Switch, SwitchPort, SwitchPortDetails, PortProfile,
    │                         SwitchDeviceCaps, SwitchPortStatus
    ├── models_ap.go          AccessPoint, ApLanPort, ApRadioSettings, ApRadioStatus,
    │                         ApChannel, ApRadioChannels
    ├── models_gateway.go     Gateway, GatewayPortStatus, GatewayPortConfig
    ├── models_network.go     LanNetwork, DhcpReservation, FirmwareUpdate
    ├── models_vpn.go         VpnPolicy
    ├── controller.go         Client, New, Option, controller-level methods
    ├── site_client.go        SiteClient + client/device/topology/LED/firmware methods
    ├── site_ap.go            AP methods + ApPortSettings, ApRadioSettings types
    ├── site_switch.go        switch methods + SwitchPortSettings, PortProfileOverrides
    ├── site_gateway.go       gateway methods + GatewayPortSettings
    ├── site_lan.go           LanNetworks + Vlan request types + preview/create/update/delete
    ├── site_dhcp.go          DHCP reservation CRUD
    ├── site_vpn.go           VpnPolicies, SetVpnPolicyEnabled
    └── testdata/             fixture JSON (written fresh, shaped like the API)
```

Notes:
- Python's `OmadaListDevice`/`OmadaDetailedDevice` fold into one `Device` struct (builder call: the split buys nothing in Go; optional fields are pointers).
- Python's `NetworkClient` inheritance hierarchy ports as one flat `Client` struct with `*WiredDetails`/`*WirelessDetails` pointer fields (spec decision 4).
- `models_*.go` files contain only structs + constructors + accessors; all methods that hit the network live in `controller.go`/`site_*.go`.

---

### Task 1: Module scaffold, version comparator, errors

**Files:**
- Create: `go.mod`, `LICENSE`, `controller/version.go`, `controller/errors.go`
- Test: `controller/version_test.go`, `controller/errors_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `type Version struct` (unexported fields), `func ParseVersion(s string) (Version, error)`, `func (v Version) Compare(o Version) int` (-1/0/1), `func (v Version) String() string`.
  - `type RequestError struct { Code int; Msg string }` with `Error() string`, `Is(target error) bool`.
  - `type ValidationError struct { Msg string }` with `Error() string`.
  - Sentinels: `ErrLoginFailed` (`*RequestError{Code: -30109}`), `ErrLoginSessionClosed`, `ErrUnsupportedControllerVersion`, `ErrSiteNotFound`, `ErrConnectionFailed`, `ErrBadControllerURL`, `ErrInvalidDevice` (all `errors.New`, except the first).

- [ ] **Step 1: Initialize repo and module**

```bash
git init -b main
printf 'module github.com/RonaldPhilipsen/omada\n\ngo 1.24\n' > go.mod
```

Write `LICENSE`: MIT, copyright 2026 Ronald Philipsen, plus the line:
`Ported from tplink-omada-client (https://github.com/MarkGodwin/tplink-omada-api) by Mark Godwin, MIT licensed.`

- [ ] **Step 2: Write the failing tests**

`controller/version_test.go` - table-driven:

```go
func TestVersionCompare(t *testing.T) {
    cases := []struct{ a, b string; want int }{
        {"5.1.0", "5.1.0", 0},
        {"5", "5.0.1", -1},
        {"6", "5.20", 1},            // numeric, not lexicographic
        {"5.9", "5.10", -1},         // numeric, not lexicographic
        {"6.2.0.0", "6.3", -1},
        {"5.15.8.12", "5.15.8.9", 1},
        {"6", "6.0.0.0", 0},
        {"5.1.0-beta", "5.1.0", 0},  // non-numeric tail ignored
    }
    for _, c := range cases {
        a, errA := ParseVersion(c.a)
        b, errB := ParseVersion(c.b)
        if errA != nil || errB != nil { t.Fatalf("parse: %v, %v", errA, errB) }
        if got := a.Compare(b); got != c.want { t.Errorf("%s vs %s = %d, want %d", c.a, c.b, got, c.want) }
    }
}
```

Plus: `ParseVersion("")` and `ParseVersion("x")` return an error.

`controller/errors_test.go`:

```go
func TestRequestErrorMessage(t *testing.T) {
    e := &RequestError{Code: -30109, Msg: "session expired"}
    if e.Error() != "Omada controller responded 'session expired' (-30109)" {
        t.Errorf("got %q", e.Error())
    }
}

func TestRequestErrorIs(t *testing.T) {
    if !errors.Is(&RequestError{Code: -30109, Msg: "m"}, ErrLoginFailed) {
        t.Error("-30109 should match ErrLoginFailed")
    }
    if errors.Is(&RequestError{Code: 123, Msg: "m"}, ErrLoginFailed) {
        t.Error("other codes must not match")
    }
}

func TestValidationError(t *testing.T) {
    var verr *ValidationError
    err := &ValidationError{Msg: "VLAN 1 is protected"}
    if !errors.As(err, &verr) || verr.Msg != "VLAN 1 is protected" { t.Error("errors.As failed") }
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./controller/ -run 'TestVersion|TestRequest|TestValidation'`
Expected: FAIL - `undefined: ParseVersion`, `undefined: RequestError`.

- [ ] **Step 4: Implement**

`controller/version.go`: `Version` holds `segments []int`. `ParseVersion` splits on `.`; each segment contributes its leading run of digits (none -> 0); empty input is an error. `Compare` walks segments, missing = 0, first difference wins.

`controller/errors.go`: as in Interfaces. `RequestError.Is` returns `target == ErrLoginFailed && e.Code == -30109`. `RequestError.Error()` uses the upstream format exactly: `fmt.Sprintf("Omada controller responded '%s' (%d)", e.Msg, e.Code)`.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./controller/`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add go.mod LICENSE controller/version.go controller/version_test.go controller/errors.go controller/errors_test.go
git commit -m "feat(controller): scaffold module, add version comparator and error types"
```

---

### Task 2: Cookie jar and connection scaffolding (URLs)

**Files:**
- Create: `controller/cookiejar.go`, `controller/connection.go`
- Test: `controller/cookiejar_test.go`, `controller/connection_test.go`

**Interfaces:**
- Consumes: nothing beyond stdlib.
- Produces:
  - `type permissiveJar struct` implementing `http.CookieJar` (`SetCookies(*url.URL, []*http.Cookie)`, `Cookies(*url.URL) []*http.Cookie`); `func newPermissiveJar() http.CookieJar`. Accepts cookies for IP-address hosts and plain hosts alike (Go's default `cookiejar` rejects IPs - this is the review-focus bug); scopes per `u.Host`; `SetCookies` replaces existing cookies of same name+path and drops expired ones; `Cookies` filters by path prefix.
  - `type conn struct` with fields `base, host, username, password string`, `client *http.Client`, `authMu sync.Mutex`, `controllerID, version, csrf string`, `lastLogon time.Time`, `now func() time.Time` (defaults to `time.Now`; tests inject a clock).
  - `func newConn(rawURL, username, password string, httpClient *http.Client) (*conn, error)` - prepends `https://` when the URL has no `http://`/`https://` scheme; stores normalized base (no trailing slash) and `host`; `httpClient` must be non-nil (built by `New`, Task 9).
  - `func (c *conn) url(endpoint, site string) string` -> `<base>/<controllerID>/api/v2/[sites/<site>/]<endpoint>`.
  - `func (c *conn) openapiURL(version, endpoint, site string) string` -> `<base>/openapi/<version>/<controllerID>/[sites/<site>/]<endpoint>`.

- [ ] **Step 1: Write the failing tests**

`controller/cookiejar_test.go`:

```go
func TestJarAcceptsIPHost(t *testing.T) {
    jar := newPermissiveJar()
    u, _ := url.Parse("https://192.168.1.10/login")
    jar.SetCookies(u, []*http.Cookie{{Name: "SESSION", Value: "abc", Path: "/"}})
    got := jar.Cookies(u)
    if len(got) != 1 || got[0].Value != "abc" { t.Errorf("got %v", got) }
}

func TestJarPathScoping(t *testing.T) {
    // cookie with Path /api is NOT returned for /other
}

func TestJarReplaceByName(t *testing.T) {
    // second SetCookies with same name+path replaces; no duplicates; expired dropped
}
```

`controller/connection_test.go`:

```go
func TestNewConnNormalizesURL(t *testing.T) {
    c, err := newConn("192.168.1.10", "u", "p", &http.Client{})
    // err nil, c.base == "https://192.168.1.10", c.host == "192.168.1.10"
    c2, err2 := newConn("http://omada.local:8043/", "u", "p", &http.Client{})
    // err2 nil, c2.base == "http://omada.local:8043" (scheme kept, slash trimmed)
}

func TestConnURLs(t *testing.T) {
    c, _ := newConn("https://omada", "u", "p", &http.Client{})
    c.controllerID = "abc"
    // c.url("cmd/reboot", "")            == "https://omada/abc/api/v2/cmd/reboot"
    // c.url("clients", "s1")             == "https://omada/abc/api/v2/sites/s1/clients"
    // c.openapiURL("v2", "clients", "s1")== "https://omada/openapi/v2/abc/sites/s1/clients"
    // c.openapiURL("v1", "setting/service/dhcp", "s1") == "https://omada/openapi/v1/abc/sites/s1/setting/service/dhcp"
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./controller/ -run 'TestJar|TestNewConn|TestConnURLs'`
Expected: FAIL - `undefined: newPermissiveJar`, `undefined: newConn`.

- [ ] **Step 3: Implement `cookiejar.go` and `connection.go`**

Jar: `map[string][]*http.Cookie` keyed by host behind a mutex. `url()`/`openapiURL()` join segments onto the base; empty `site` omits the `sites/<site>/` segment. `newConn` trims trailing slashes and keeps an explicit `http://` scheme.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./controller/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add controller/cookiejar.go controller/cookiejar_test.go controller/connection.go controller/connection_test.go
git commit -m "feat(controller): add IP-accepting cookie jar and conn URL building"
```

---

### Task 3: Login, response unwrapping, auto re-login

**Files:**
- Modify: `controller/connection.go` (add auth + request plumbing)
- Test: `controller/connection_auth_test.go`

**Interfaces:**
- Consumes: `newConn`, `permissiveJar`, `RequestError`/sentinels (Tasks 1-2), `Version`/`ParseVersion` (Task 1).
- Produces:
  - `func (c *conn) login(ctx context.Context) (string, error)` - clears csrf + jar cookies for the host; `GET <base>/api/info` (unauthenticated); version < `5.1.0` -> wrap `ErrUnsupportedControllerVersion` with the version string; `POST <base>/<id>/api/v2/login` body `{"username":...,"password":...}`; stores `controllerID` (info `omadacId`), `version` (info `controllerVer`), `csrf` (login response `token` inside `result`), `lastLogon = c.now()`. Returns controllerID.
  - `func (c *conn) checkLogin(ctx context.Context) (bool, error)` - no csrf -> `(false, nil)`; `c.now()-lastLogon < 1h` -> `(true, nil)`; `GET <base>/<id>/api/v2/loginStatus` -> `(response["login"] != false, nil)`; auth-class failure (`-30109` / `ErrLoginSessionClosed`) -> `(false, nil)`; any other error -> `(false, err)` - propagated, NO re-login (spec decision 3).
  - `func (c *conn) request(ctx context.Context, method, u string, query url.Values, body any) (map[string]any, error)` - under `authMu`: `checkLogin`; if not valid and no err, `login()`. Then `do()`; on `errors.Is(ErrLoginFailed)` or `errors.Is(ErrLoginSessionClosed)`: take `authMu`, `login()`, retry `do()` exactly once (a second auth failure propagates).
  - `func (c *conn) do(ctx context.Context, method, u string, query url.Values, body any) (map[string]any, error)` - builds the `*http.Request` (JSON-encoded body when `body != nil`; query via `query`); when `csrf != ""` sets headers `Csrf-Token: <csrf>`, `Omada-Request-Source: web-local`, `Referer: <base>/`, `Origin: <base>`. Non-200: if body parses as JSON, `checkAppErrors` result wins, else `&RequestError{Code: status, Msg: "HTTP Request Error"}`. 200 + non-JSON content type -> `ErrLoginSessionClosed`. 200 + JSON -> `checkAppErrors`, then unwrap: return `content["result"]` (as `map[string]any`) when a `result` key exists, else `content`. Network-level failure -> wrap `ErrConnectionFailed`; unparseable URL -> wrap `ErrBadControllerURL`.
  - `func (c *conn) checkAppErrors(content map[string]any) error` - no `errorCode` key -> `&RequestError{Code: -30109, Msg: "Unexpected response: ..."}`; `errorCode == 0` -> nil; `errorCode == -30109` -> `&RequestError{Code: -30109, Msg: msg}` (matches `ErrLoginFailed` via `Is`); otherwise -> `&RequestError{Code, Msg}`.
  - `func (c *conn) controllerInfo(ctx context.Context) (map[string]any, error)` - unauthenticated `GET <base>/api/info`, unwrapped.
  - `func (c *conn) controllerVersion(ctx context.Context) (Version, error)` - cached `c.version` when set, else fetch via `controllerInfo`.
  - `func (c *conn) siteInfo(ctx context.Context, site string) (map[string]any, error)` - `GET <base>/<id>/api/v2/sites/<site>`.
  - `func (c *conn) siteVersion(ctx context.Context, site string) (Version, error)`.

- [ ] **Step 1: Write the failing tests**

`controller/connection_auth_test.go` - one `httptest.Server` helper serves scripted routes and records requests (method, path, headers, body). Each test builds a conn against the server URL with an injected `now` clock:

```go
func TestLoginFlow(t *testing.T) {
    // GET /api/info -> {"controllerVer":"6.2.10.18","omadacId":"abc"}
    // POST /abc/api/v2/login (JSON username/password) -> {"result":{"token":"tok"}}
    id, err := c.login(ctx)
    // id == "abc"
    // then c.request(ctx, "GET", c.url("loginStatus", ""), nil, nil) succeeds and
    // the server asserts: Csrf-Token: tok, Omada-Request-Source: web-local,
    // Referer: <base>/, Origin: <base>, and a cookie round-tripped via the jar
}

func TestLoginRejectsOldVersion(t *testing.T) {
    // /api/info -> controllerVer "5.0.15.2" -> errors.Is(err, ErrUnsupportedControllerVersion)
}

func TestReLoginOnLoginFailedThenRetry(t *testing.T) {
    // login ok (token "t1"); first GET /abc/api/v2/devices
    //   -> 200 JSON {"errorCode":-30109,"msg":"session expired"}
    // re-login issues token "t2"; retried GET devices -> {"result":{"ok":true}}
    // assert: exactly 2 POST /abc/api/v2/login calls, devices GET made twice
}

func TestNonJSON200IsSessionClosed(t *testing.T) {
    // 200, Content-Type text/html, body "<html>login</html>" -> errors.Is(err, ErrLoginSessionClosed)
}

func TestApplicationErrorCodeNoRelogin(t *testing.T) {
    // 200 JSON {"errorCode":8101,"msg":"no permission"} -> *RequestError Code 8101
    // assert: login endpoint hit exactly once (no re-login)
}

func TestCheckLoginTransientErrorPropagates(t *testing.T) {
    // >1h since last logon; loginStatus returns 500 -> request() returns *RequestError
    // Code 500 and login is NOT called a second time
}

func TestConcurrentFirstRequestsNoRace(t *testing.T) {
    // 10 goroutines call c.request(ctx, "GET", ...loginStatus...) on a fresh conn
    // (loginStatus -> {"login":true}); all succeed. Run with -race.
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./controller/ -race -run 'TestLogin|TestReLogin|TestNonJSON|TestApplication|TestCheckLogin|TestConcurrent'`
Expected: FAIL - `c.login undefined`.

- [ ] **Step 3: Implement**

Follow the Interfaces block exactly. Key subtleties:
- `do()` reads the response body once into a `[]byte`; decides JSON vs non-JSON from the `Content-Type` prefix `application/json` (upstream behavior, `omadaapiconnection.py:_do_request`).
- `request()` holds `authMu` only around checkLogin/login decisions, never around `do()` itself (data requests run concurrently; only auth transitions serialize).
- Version gate in `login()` compares `ParseVersion(controllerVer)` against `ParseVersion("5.1.0")`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./controller/ -race`
Expected: PASS, race detector clean.

- [ ] **Step 5: Commit**

```bash
git add controller/connection.go controller/connection_auth_test.go
git commit -m "feat(controller): add login, response unwrapping and auto re-login"
```

---

### Task 4: Bounded pagination helpers

**Files:**
- Create: `controller/pagination.go`
- Test: `controller/pagination_test.go`

**Interfaces:**
- Consumes: `conn.request` (Task 3).
- Produces (all collect fully; `const pageSize = 100`, `const maxPages = 1000`):
  - `func (c *conn) pagesV2(ctx context.Context, u string, params url.Values) ([]map[string]any, error)` - GET loop; each iteration sets `currentPage`/`currentPageSize` query params; appends `data["data"]` entries; stops when: page returned 0 entries, `totalRows` present and `page*pageSize >= totalRows`, or `page > maxPages` (defect fix 1 - upstream `omadaapiconnection.py:page_results` loops forever when `totalRows > 0` but pages come back empty).
  - `func (c *conn) pagesOpenAPIPost(ctx context.Context, u string, body map[string]any) ([]map[string]any, error)` - POST loop; mutates a copy of `body` per page with `page`/`pageSize`; same stop conditions (defect fix 1).
  - `func (c *conn) pagesOpenAPIGet(ctx context.Context, u string, params url.Values) ([]map[string]any, error)` - GET loop with `page`/`pageSize` query params; same stop conditions.
  - All three return the collected entries; on any transport/auth error they return the error (partial results discarded - simpler and matches "the write/read contract").

- [ ] **Step 1: Write the failing tests**

`controller/pagination_test.go` - scripted server:

```go
func TestPagesV2CollectsAllPages(t *testing.T) {
    // page 1 -> 100 entries, totalRows 250; page 2 -> 100; page 3 -> 50
    // assert 250 entries collected, exactly 3 GETs, currentPage seen as 1,2,3
}

func TestPagesV2StopsOnTotalRows(t *testing.T) {
    // page 1 -> 100 entries, totalRows 100 -> only 1 GET
}

func TestPagesV2BoundedOnEmptyPage(t *testing.T) {
    // page 1 -> 100 entries, totalRows 5000; page 2 -> 0 entries
    // loop MUST stop (defect fix 1), returns 100 entries, no more than 2 GETs
}

func TestPagesOpenAPIPostCollectsAllPages(t *testing.T) {
    // page 1 -> 100, page 2 -> 40 -> 140 entries; assert body had page/pageSize set
}

func TestPagesOpenAPIGetCollectsAllPages(t *testing.T) {
    // page 1 -> 100 entries totalRows 100 -> 1 GET, 100 entries
}

func TestPagesHardCap(t *testing.T) {
    // endpoint that always returns 100 entries with totalRows 10000
    // loop stops at 1000 pages without hanging; returns 1000*100 entries
    // (keep runtime sane: make page size 100 and server respond instantly)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./controller/ -run 'TestPages'`
Expected: FAIL - `c.pagesV2 undefined`.

- [ ] **Step 3: Implement**

One shared helper `collectPages(ctx, fetch func(page int) ([]map[string]any, int64 /*totalRows, -1 if absent*/, error))` implements the stop conditions; the three flavors only differ in how they build the request (GET query vs POST body vs GET with page params).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./controller/ -race`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add controller/pagination.go controller/pagination_test.go
git commit -m "feat(controller): add bounded pagination collectors"
```

---

### Task 5: Enums

**Files:**
- Create: `controller/enums.go`
- Test: `controller/enums_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: int-based enums with `String()`, plus an unexported `parseEnum(name string, valid map[string]int, unknown int) int` helper (unknown input -> `unknown` value; mirrors Python `_missing_`). Every enum exports an `Unknown`/`None` member. Full set (upstream `definitions.py`):
  - `ControllerType` (standard/advanced/none/unknown), `ControllerStatus` (starting/running/upgrade/saving), `UpdateSection` (default/omadasw/...).
  - `DeviceType` (gateway/switch/accessPoint/client/omadaHardware), `DeviceModel` (from `deviceModel` string map).
  - `DeviceStatus` (normal/firmwareDown/firmwareUpgrade/offline/disconnect/maintenance), `DeviceSubStatus` (normal/... full upstream table).
  - `DeviceUptimeStatus` (0/1/2 mapping: stopped/unavailable/normal).
  - `ClientStatus` (none/online/... upstream table incl. `block` semantics), `ClientLinkState` (disconnected/connected).
  - `RadioId` string->int map from `definitions.RADIO_ID_MAP` (e.g. `"2g"=0` per upstream table - port the table verbatim) and `Radio` enum (radio0..radio3 per upstream).
  - `PortMode` (default/access/trunk/... full switch table), `PortStatus` (up/down/...), `PortVlanType`.
  - `ApPortType` (lan/wan/poe...), `ApRadioStatus` (enabled/disabled...), `ApWanType`, `ApInterfaceType`.
  - `GatewayType` (gateway/switch...), `GatewayPortType` (ethernet/poe/...), `GatewayPoEStatus` (on/off/auto...), `GatewayPortStatus`.
  - `LanNetworkStatus` (normal/...), `DhcpMode` (enabled/disabled/relay), `IpSettingType` (manual/dhcp/none).
  - `VpnType` strings (ipsec/l2tp/pptp/openvpn/wireguard as `int` constants + name lookup), `VpnCategory` (gateway/endpoint...), `VpnStatus` (on/off).
  - `ChannelWidth`, `SecurityMode`, `SsidStatus`.
  - `LedState` (on/off/blink...), `FirmwareUpdateStatus`.

Port rule: copy each upstream `IntEnum` table verbatim (name, value, display string). Where upstream uses a plain `str` enum (radio ids, VPN types), port as the int/string table it is used against.

- [ ] **Step 1: Write the failing tests**

`controller/enums_test.go`:

```go
func TestDeviceTypeStrings(t *testing.T) {
    if DeviceTypeGateway.String() != "gateway" { t.Fail() }
    if DeviceType(99).String() != "unknown" { t.Fail() } // Unknown member
}

func TestParseEnumUnknown(t *testing.T) {
    // parseEnum("nonsense", validTable, unknownValue) == unknownValue
}

func TestRadioIdMap(t *testing.T) {
    // port a sample of upstream RADIO_ID_MAP entries verbatim, e.g.
    // "1g" -> expected int from definitions.py
}

func TestControllerTypeRoundTrip(t *testing.T) {
    // every member: String() non-empty and unique within the enum
}
```

(Complete the tables with the real upstream values read from `/tmp/opencode/tplink-omada-api/src/tplink_omada_client/definitions.py`.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./controller/ -run 'TestDeviceType|TestParseEnum|TestRadioId|TestControllerType'`
Expected: FAIL - undefined types.

- [ ] **Step 3: Implement `enums.go`**

One file; each enum: `type X int32` + const block + `var xNames = map[int32]string{...}` + `String()`. Verify every value against the upstream file before finishing.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./controller/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add controller/enums.go controller/enums_test.go
git commit -m "feat(controller): port enum tables from upstream definitions"
```

---

### Task 6: Controller-level models

**Files:**
- Create: `controller/models_controller.go`
- Test: `controller/models_controller_test.go`, fixture `controller/testdata/controller-info.json`

**Interfaces:**
- Consumes: enums (Task 5).
- Produces (each wraps `map[string]any`; constructor `New*(m map[string]any)`; `Raw() map[string]any`):
  - `ControllerInfo`: `ID() string` (omadacId), `Version() string` (controllerVer), `Type() ControllerType`, `Status() ControllerStatus`, `Name() string`, `SiteCount() int` (siteNum), `Hardware() string`, `Serial() string`.
  - `Site`: `ID() string`, `Name() string`, `GroupID() string`, `Version() string`, `Online() bool`.
  - `ControllerUpdateInfo`: `UpdateInfo() map[string]any` passthrough + `HasUpdate() bool`.
  - `UpdateSection`: from `updateSections`.
  - `HardwareUpgradeStatus`: `Status() string`, `Progress() int`, `Description() string` (upstream `HardwareUpgradeStatus`).
  - `InterfaceDetails`: `Name()`, `IP()`, `Mask()`, `Gateway()`, `Type()`.
  - `ClientStats` (upstream top-level `client_stats` shape): `Online() int`, `Total() int`, `ByType map[string]int`.

- [ ] **Step 1: Write the failing tests**

`controller/testdata/controller-info.json` (shaped like the real API, hand-written):

```json
{"controllerVer": "6.2.10.18", "omadacId": "abc", "controllerType": 1,
 "controllerStatus": 1, "name": "Home", "siteNum": 2, "serial": "SN", "hw": "EAP670"}
```

```go
func TestControllerInfo(t *testing.T) {
    m := loadFixture(t, "controller-info.json")
    c := NewControllerInfo(m)
    // ID()=="abc", Version()=="6.2.10.18", Type()==ControllerType(1),
    // Status()==ControllerStatus(1), Name()=="Home", SiteCount()==2
}

func TestControllerInfoSparse(t *testing.T) {
    // NewControllerInfo(map[string]any{}) -> zero values, no panic
    // string fields "", int fields 0, enum -> Unknown member
}

func TestSiteSparse(t *testing.T) {
    // NewSite(map[string]any{"siteId":"s1","name":"Office"}) -> ID/Name set, Online()==false
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./controller/ -run 'TestControllerInfo|TestSite'`
Expected: FAIL - undefined constructors.

- [ ] **Step 3: Implement `models_controller.go`**

Accessor pattern (used by all model files): `func str(m map[string]any, k string) string`, `int`, `bool`, `map` unexported helpers; models store the map, accessors go through helpers. Missing key -> zero value, never panic (review focus 3).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./controller/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add controller/models_controller.go controller/models_controller_test.go controller/testdata/controller-info.json
git commit -m "feat(controller): add controller-level models"
```

---

### Task 7: Device, client, topology models

**Files:**
- Create: `controller/models_device.go`, `controller/models_client.go`, `controller/models_topology.go`
- Test: `controller/models_device_test.go`, `controller/models_client_test.go`, `controller/models_topology_test.go`
- Fixtures: `testdata/device-gateway.json`, `testdata/device-ap-sparse.json`, `testdata/client-wired.json`, `testdata/client-wireless.json`, `testdata/client-sparse.json`, `testdata/topology.json`

**Interfaces:**
- Consumes: enums (Task 5).
- Produces:
  - `Device` (folds upstream `OmadaListDevice` + `OmadaDetailedDevice`): `ID()`, `Name()`, `Type() DeviceType`, `Model() string` (deviceModel), `Status() DeviceStatus`, `SubStatus() DeviceSubStatus`, `Online() bool`, `IP() string` (ethmac/ip), `MAC() string`, `Uptime() int64` (seconds), `Firmware() string` (swVer), `PortNum() int` (top-level, else `deviceMisc.portNum` - sparse shape from review focus 3), `WanIP() string` (wanIp), `ModelName() string` (modelDescription), `Raw()`.
  - `Link`/`Uplink`/`Downlink` accessors on Device: `Uplinks() map[string]*Link`, `Downlinks() map[string]*Link` (port->link) from `deviceMisc`.
  - `Client` (flat; upstream `NetworkClient` + wired/wireless detail classes): `ID()` (mac), `Name()`, `Model()`, `IP() string` (lastSeenIP), `MAC()` (mac), `Status() ClientStatus` (port upstream `block`/`blocked` dual key), `Connected() bool`, `FirstOnline() time.Time`, `LastSeen() time.Time`, `SSID() string`, `BSSID() string`, `Signal() int`, `Rate() float64` (txRate), `VLAN() int`, `DeviceID() string` (devId), `SiteID() string`, `Wired() *WiredDetails` (nil when absent), `Wireless() *WirelessDetails` (nil when absent), `Raw()`.
  - `WiredDetails`: `SwitchID()`, `SwitchPort() int`, `Speed() string`, `Status() string`.
  - `WirelessDetails`: `Radio() string` (radId), `Band() string`, `Security() string`, `Channel() int`.
  - `IpSetting`: `Type() IpSettingType`, `IP()`, `Mask()`, `Gateway()`, `DNS() []string`.
  - `RateLimit`: `Up() int64`, `Down() int64`, `Enabled() bool`.
  - `Topology`: `Nodes() []*TopologyNode`, `WanPort() *TopologyWanPort`.
  - `TopologyNode`: `ID()`, `Name()`, `Type() DeviceType`, `Model()`, `IP()`, `Online() bool`, `Links() map[string]*Link`.
  - `TopologyWanPort`: `Type() string`, `IP()`, `Online() bool`.

- [ ] **Step 1: Write the failing tests**

Fixtures shaped like real API responses (hand-written, minimal but key-complete):
- `device-gateway.json`: gateway with `deviceMisc` (`portNum`, `uplinks`, `downlinks`), `ethmac`, `ip`, `swVer`, `wanIp`.
- `device-ap-sparse.json`: AP with NO `deviceMisc` (review focus 3) -> `PortNum()==0`, no panic.
- `client-wired.json`: client with `wired` details, `block:false`.
- `client-wireless.json`: client with `wireless` details, keys `mac`,`lastSeenIP`,`signal`,`txRate`,`vlan`,`devId`.
- `client-spaque.json`: `{"deviceIndex":"x","devId":"d","mac":"aa:bb:cc:dd:ee:ff","clientName":"laptop"}` only -> every optional accessor zero, `Wired()==nil`, `Wireless()==nil`.
- `topology.json`: 2 nodes + wanPort.

Tests: one per fixture asserting accessors; plus `TestClientSparse` (zero values, nil detail pointers, no panic) and `TestDeviceSparseNoDeviceMisc`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./controller/ -run 'TestDevice|TestClient|TestTopology'`
Expected: FAIL - undefined constructors.

- [ ] **Step 3: Implement the three model files**

Same accessor helpers as Task 6. `Client` detail detection: `wired` key present and non-null -> `NewWiredDetails(...)`; same for `wireless`. Timestamps: upstream uses epoch millis (`firstOnlineTime`, `lastSeenTime`) -> `time.UnixMilli`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./controller/ -race`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add controller/models_device.go controller/models_client.go controller/models_topology.go controller/*_test.go controller/testdata/
git commit -m "feat(controller): add device, client and topology models"
```

---

### Task 8: Switch, AP, gateway, network, VPN models

**Files:**
- Create: `controller/models_switch.go`, `controller/models_ap.go`, `controller/models_gateway.go`, `controller/models_network.go`, `controller/models_vpn.go`
- Test: one test file per model file + fixtures in `testdata/`

**Interfaces:**
- Consumes: enums (Task 5).
- Produces (all wrap `map[string]any`, sparse-safe):
  - `Switch`: from `portConfigs []map[string]any` -> `Ports() []*SwitchPort` (missing key -> nil slice).
  - `SwitchPort`: `Index() int` (portIndex), `Name()`, `Mode() PortMode`, `VLANType() PortVlanType`, `PVID() int`, `Status() PortStatus` (portStatus), `Speed() string`, `PoE() bool`, `Link() *Link`.
  - `SwitchPortDetails`: `Port()` (portInfo), `PVID`, `TaggedVLANs() []int`, `Untagged() *int`, `Mode()`.
  - `PortProfile`: `ID()`, `Name()`, `Description()`, `Ports() []int`, `PVID() int`, `VLANs() []int`.
  - `SwitchDeviceCaps`: `PoE() bool`, `Ports() int`, `VLAN() bool`.
  - `SwitchPortStatus`: `Port() int`, `Speed() string`, `Status() PortStatus`, `Duplex() string`.
  - `AccessPoint` (wraps device-style map): `RadioSettings() []*ApRadioSettings` (from `radioConfigList`), `RadioStatus() []*ApRadioStatus` (from `radioConfigStatusList`), `LanPort() *ApLanPort` (from `lanPort`), `WanType() ApWanType` (from `deviceMisc.wanType`).
  - `ApLanPort`: `IP()`, `Mask()`, `Gateway()`.
  - `ApRadioSettings`: `RadioID() int` (radId), `Name()`, `Enabled() bool`, `Band() string`, `ChannelWidth() ChannelWidth`, `Channel() int`, `SSIDs()` passthrough, `TxPower() int`.
  - `ApRadioStatus`: `RadioID() int`, `Enabled() bool`, `ConnectedClients() int` (connectedClients), `Channel() int`, `Utilization() float64`.
  - `ApChannel`: `Channel() int`, `Width() int`, `Band() string`.
  - `ApRadioChannels`: `Channels() []*ApChannel`.
  - `Gateway`: `Type() GatewayType`, `PortStatuses() []*GatewayPortStatus` (missing `portConfigs` -> nil, review focus 3), `WanPort() *GatewayPortConfig`, `PoE() bool`.
  - `GatewayPortStatus`: `Index() int`, `Status() GatewayPortStatus`, `Speed() string`, `PoE() GatewayPoEStatus`.
  - `GatewayPortConfig`: `Index() int`, `Type() GatewayPortType`, `PoE() GatewayPoEStatus`, `Enabled() bool`.
  - `LanNetwork`: `ID()`, `Name()`, `VLAN() int` (lanVlanId), `Subnet()`, `Mask()`, `Gateway()`, `DHCPServer() interface passthrough` (`Dhcp() bool` from `lanDhcpServersEnabled`), `Enabled() bool`.
  - `DhcpReservation`: `Host() string` (hostName), `IP()`, `MAC()`.
  - `FirmwareUpdate`: `Name()`, `Current()`, `Available()`, `Progress() int`.
  - `VpnPolicy`: `ID()`, `Name()`, `Type() VpnType`, `Category() VpnCategory`, `Enabled() bool` (isPolicyEnabled), `Status() VpnStatus`.

- [ ] **Step 1: Write the failing tests**

One fixture per model (minimally key-complete) + a sparse fixture per model (empty map) asserting zero values / nil slices and no panics:

```go
func TestSwitchPorts(t *testing.T) {
    // fixture: portConfigs with 2 entries (portIndex, mode, pvid, portStatus)
    // Ports() length 2, PortMode/PortStatus enum values correct
}
func TestSwitchSparse(t *testing.T) {
    // NewSwitch(map[string]any{}) -> Ports()==nil, no panic
}
func TestApRadioSettings(t *testing.T) { /* radioConfigList -> entries */ }
func TestApSpare(t *testing.T) {
    // NewAccessPoint(map[string]any{}) -> RadioSettings()==nil, RadioStatus()==nil, LanPort()==nil
}
func TestGatewayNoPortConfigs(t *testing.T) {
    // NewGateway(map[string]any{"wanPort": map[string]any{...}}) -> PortStatuses()==nil, WanPort()!=nil
}
func TestLanNetwork(t *testing.T) { /* vlan, subnet, dhcp enabled */ }
func TestVpnPolicy(t *testing.T) { /* type ipsec, enabled */ }
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./controller/ -run 'TestSwitch|TestAp|TestGateway|TestLanNetwork|TestVpnPolicy'`
Expected: FAIL - undefined constructors.

- [ ] **Step 3: Implement the five model files**

Same accessor pattern. List-typed accessors iterate `m[key]` as `[]any` (JSON decode of `map[string]any` yields `[]any`, not `[]map[string]any` - cast per item) and return nil when the key is missing or not a list.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./controller/ -race`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add controller/models_switch.go controller/models_ap.go controller/models_gateway.go controller/models_network.go controller/models_vpn.go controller/testdata/
git commit -m "feat(controller): add switch, AP, gateway, network and VPN models"
```

---

### Task 9: Client (controller level)

**Files:**
- Create: `controller/controller.go`
- Test: `controller/controller_test.go`

**Interfaces:**
- Consumes: `conn` (Tasks 2-4), models (Tasks 6-8), enums.
- Produces:
  - `type Client struct { c *conn }`.
  - `type Option func(*options)` with `WithHTTPClient(*http.Client)`, `WithTimeout(time.Duration)` (default 30s), `WithInsecureSkipVerify()`.
  - `func New(controllerURL, username, password string, opts ...Option) (*Client, error)` - builds an `*http.Client` (timeout, TLS skip-verify, `permissiveJar` as `Jar`), `newConn(...)`, then `login(ctx)` (uses `context.Background` here - `New` has no ctx, matching upstream constructor auth; document in README). Errors: `ErrBadControllerURL`, `ErrUnsupportedControllerVersion`, `ErrLoginFailed` propagate wrapped.
  - `func (cl *Client) Sites(ctx context.Context) ([]Site, error)` - `pagesV2(url("sites",""), {siteType:"1",...})` per upstream `omadaclient.py:list_sites` (params: `siteType=1,currentPage=1,currentPageSize=100` collected via pagesV2), maps to `Site` models.
  - `func (cl *Client) Site(ctx context.Context, siteID string) (Site, error)` - `GET sites/<id>`; `siteID==""` -> `ValidationError`.
  - `func (cl *Client) SiteByName(ctx context.Context, name string) (Site, error)` - list sites, match `Name()`; miss -> `ErrSiteNotFound` wrapped with name.
  - `func (cl *Client) ControllerInfo(ctx context.Context) (ControllerInfo, error)` - unauthenticated `controllerInfo` + model.
  - `func (cl *Client) ControllerVersion(ctx context.Context) (Version, error)` - `conn.controllerVersion`.
  - `func (cl *Client) Reboot(ctx context.Context) error` - `POST cmd/reboot`.
  - `func (cl *Client) Shutdown(ctx context.Context) error` - `POST cmd/shutdown`.
  - `func (cl *Client) SiteClient(ctx context.Context, siteID string) (*SiteClient, error)` - validates site exists (`siteInfo`), returns `&SiteClient{c: cl.c, site: siteID}`.
  - `func (cl *Client) FirmwareUpdates(ctx context.Context) (ControllerUpdateInfo, error)` - upstream `get_hardware_upgrade_status`-style fetch (url `upgrade`), model.
  - `func (cl *Client) UpdateFirmwareChannel(ctx context.Context, section UpdateSection, channel string) error` - upstream `change_upgrade_channel` body `{"section":..., "channel":...}` to `upgrade`.
  - `func (cl *Client) SetCertificate(ctx context.Context, keyPEM, certPEM string) error` - multipart/form-data POST to `cmd/uploadCert` fields `key`/`cert` (upstream `omadaclient.py:set_certificate`).
  - `func (cl *Client) Close() error` - `cl.c.client.CloseIdleConnections()`.

- [ ] **Step 1: Write the failing tests**

`controller/controller_test.go` - scripted `httptest.Server`; helper `newTestClient(t, handler)` wires `New(server.URL, "u", "p", WithHTTPClient(...))` with login fixture. Tests:

```go
func TestNewLogsInAndPrefersScheme(t *testing.T) {
    // New("192.168.1.10", ...) against a server whose URL has no IP -> use server.URL
    // assert login POST happened exactly once
}

func TestSites(t *testing.T) {
    // sites page 1 -> 2 sites -> []Site len 2 with IDs/names
}

func TestSiteByNameMiss(t *testing.T) {
    // errors.Is(err, ErrSiteNotFound), message contains the name
}

func TestSiteEmptyID(t *testing.T) {
    // *ValidationError
}

func TestRebootAndShutdown(t *testing.T) {
    // POSTs hit /<id>/api/v2/cmd/reboot and /cmd/shutdown
}

func TestSiteClientValidatesSite(t *testing.T) {
    // unknown site -> wrapped ErrSiteNotFound; known -> SiteClient usable (loginStatus GET with sites/<id> prefix)
}

func TestSetCertificate(t *testing.T) {
    // server asserts Content-Type prefix multipart/form-data and fields key/cert present
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./controller/ -run 'TestNew|TestSites|TestSite|TestReboot|TestSetCertificate'`
Expected: FAIL - `undefined: New`.

- [ ] **Step 3: Implement `controller.go`**

`New` is the only place that builds the HTTP client. `Sites` uses `pagesV2` with the upstream query params (port `omadaclient.py:list_sites` exactly, including the `siteType=1` filter). `SiteByName` linear-scan over `Sites`. Multipart via `mime/multipart`, buffer in memory.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./controller/ -race`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add controller/controller.go controller/controller_test.go
git commit -m "feat(controller): add controller-level Client"
```

---

### Task 10: SiteClient - clients

**Files:**
- Create: `controller/site_client.go`
- Test: `controller/site_client_test.go`

**Interfaces:**
- Consumes: `conn` (Tasks 2-4), `Client` (Task 9), client models (Task 7), enums.
- Produces:
  - `type SiteClient struct { c *conn; site string }` (constructed by `Client.SiteClient`; the pointer into `conn` is shared, so `Close`/lifecycle belongs to `Client`).
  - `func (s *SiteClient) SiteVersion(ctx context.Context) (Version, error)` - `conn.siteVersion`.
  - `func (s *SiteClient) Clients(ctx context.Context) ([]Client, error)` - `pagesV2(url("clients", site), {limit:"1000"})` per upstream `omadasiteclient.py:get_clients` (query `limit=1000`), map each to `Client` model.
  - `func (s *SiteClient) Client(ctx context.Context, mac string) (Client, error)` - upstream `get_client(mac)`: `mac==""` -> `ValidationError`; `GET clients?mac=<mac>`; empty -> `ValidationError` "not found" (upstream raises `ValueError`; use `*ValidationError` per spec decision 2).
  - `func (s *SiteClient) ClientByID(ctx context.Context, deviceIndex string) (Client, error)` - `GET clients/<deviceIndex>`.
  - `func (s *SiteClient) BlockClient(ctx context.Context, mac string) error` - `PATCH clients/<deviceIndex>` body `{"block": true}`; upstream resolves deviceIndex via `get_client` first.
  - `func (s *SiteClient) UnblockClient(ctx context.Context, mac string) error` - body `{"block": false}`.
  - `func (s *SiteClient) ReconnectClient(ctx context.Context, mac string) error` - upstream `reconnect_client`: `POST clients/<deviceIndex>/cmd/reconnect`.
  - `func (s *SiteClient) UpdateClientSettings(ctx context.Context, mac string, name string, ipSetting *IpSetting, rateLimit *RateLimit) error` - upstream `set_client_settings`: builds body from non-nil args (`clientName`, `ipSetting` map, `rateLimit` map) and `PATCH clients/<deviceIndex>`; all-nil -> `ValidationError`.
  - `func (s *SiteClient) ConnectedClients(ctx context.Context) ([]Client, error)` - **version gate (review focus 5)**: `siteVersion >= 6.2.0.0` -> `pagesV2(url("clients", site), {filter:"isOnline"})`-style upstream 6.2 query; else `Clients(ctx)` filtered in Go by `Connected()`; older controllers return the filtered list. Port `omadasiteclient.py:get_connected_clients` exactly.
  - `func (s *SiteClient) KnownClients(ctx context.Context) ([]Client, error)` - upstream `get_known_clients`: `pagesV2` with upstream query (`clientType` filter), models.

- [ ] **Step 1: Write the failing tests**

`controller/site_client_test.go` - helper `newTestSiteClient(t, handler, siteVersion string)` (login + `sites/<id>` fixture with `controllerVer`/`site` version as needed). Tests:

```go
func TestClientsAllPages(t *testing.T) { /* limit=1000 query seen; 2-page collect */ }
func TestClientByMAC(t *testing.T) {
    // GET clients?mac=... -> 1 result; empty result -> *ValidationError; "" -> *ValidationError
}
func TestBlockUnblockClient(t *testing.T) {
    // PATCH clients/<idx> body {"block":true} / {"block":false}; mac resolved via clients?mac first
}
func TestReconnectClient(t *testing.T) { /* POST clients/<idx>/cmd/reconnect */ }
func TestUpdateClientSettings(t *testing.T) {
    // name only -> body {"clientName":"x"}; ipSetting set -> nested map; all nil -> *ValidationError
}
func TestConnectedClientsVersionGate(t *testing.T) {
    // v6.2.0.0 -> server asserts the 6.2 query used; v6.1 -> client-side filter path
    // both return only connected clients
}
func TestKnownClients(t *testing.T) { /* upstream query params asserted */ }
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./controller/ -run 'TestClients|TestClient|TestBlock|TestReconnect|TestUpdateClient|TestConnectedClients|TestKnownClients'`
Expected: FAIL - `undefined: SiteClient`.

- [ ] **Step 3: Implement**

`SiteClient` holds the shared `*conn` (login state is per-controller; site ID is stored per client). `mac` -> deviceIndex resolution helper `resolveDeviceIndex(ctx, mac) (string, error)` used by block/unblock/reconnect/update. Version gate via `s.SiteVersion(ctx)`. Compare with `ParseVersion("6.2.0.0")` - numeric, per-segment (review focus 5).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./controller/ -race`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add controller/site_client.go controller/site_client_test.go
git commit -m "feat(controller): add SiteClient client management"
```

---

### Task 11: SiteClient - devices, topology, LED, firmware

**Files:**
- Create: `controller/site_device.go`
- Test: `controller/site_device_test.go`

**Interfaces:**
- Consumes: `SiteClient` (Task 10), `Device`/`Topology` models (Task 7), enums.
- Produces (methods on `*SiteClient`):
  - `Devices(ctx context.Context) ([]Device, error)` - `pagesV2(url("devices", site), ...)` per upstream `get_devices`.
  - `Device(ctx context.Context, deviceID string) (Device, error)` - `GET devices/<id>`; empty id -> `ValidationError`.
  - `DeviceByMAC(ctx context.Context, mac string) (Device, error)` - upstream `get_device_by_mac`: `GET devices?mac=<mac>` (openapi v2 route per upstream); empty -> `ValidationError`; miss -> `ErrInvalidDevice` wrapped with mac.
  - `Topology(ctx context.Context) (Topology, error)` - `GET topology` (upstream `get_topology`), `NewTopology`.
  - `TopologyNode(ctx context.Context, networkID string) (TopologyNode, error)` - upstream `get_topology_node`: `GET topology/<networkID>`; node not in result -> `ValidationError`.
  - `SetDeviceLED(ctx context.Context, deviceID string, state LedState, duration time.Duration) error` - upstream `set_led`: body `{"state": <int per enum>, "duration": seconds}` to `devices/<id>/led`.
  - `RebootDevice(ctx context.Context, deviceID string) error` - `POST devices/<id>/cmd/reboot`.
  - `UpgradeDeviceFirmware(ctx context.Context, deviceIDs []string) error` - upstream `upgrade_device_firmware`: body `{"deviceIds": [...]}` to `cmd/upgrade`; empty list -> `ValidationError`.
  - `GetFirmwareUpdates(ctx context.Context) ([]FirmwareUpdate, error)` - upstream `get_firmware_updates` (url `upgrade`), models.
  - `UpdateFirmwareChannel(ctx context.Context, section UpdateSection, channel string) error` - same shape as controller-level (site-scoped url).
  - `RestoreDevice(ctx context.Context, deviceID string) error` - upstream `restore_device`: `POST devices/<id>/cmd/restore`.
  - `FactoryResetDevice(ctx context.Context, deviceID string) error` - `POST devices/<id>/cmd/factoryReset`.

- [ ] **Step 1: Write the failing tests**

Scripted server per test; fixture `testdata/device-gateway.json` reused for device shapes:

```go
func TestDevices(t *testing.T) { /* 1 page of 2 devices -> model fields */ }
func TestDeviceEmptyID(t *testing.T) { /* *ValidationError */ }
func TestDeviceByMACMiss(t *testing.T) { /* empty list -> errors.Is(err, ErrInvalidDevice), msg has mac */ }
func TestTopology(t *testing.T) { /* nodes + wanPort */ }
func TestTopologyNodeMiss(t *testing.T) { /* networkID absent -> *ValidationError */ }
func TestSetDeviceLED(t *testing.T) { /* body state int + duration seconds; enum mapping */ }
func TestRebootRestoreReset(t *testing.T) { /* POST cmd/reboot, cmd/restore, cmd/factoryReset paths */ }
func TestUpgradeFirmware(t *testing.T) { /* deviceIds body; empty list -> *ValidationError */ }
func TestGetFirmwareUpdates(t *testing.T) { /* list -> models */ }
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./controller/ -run 'TestDevices|TestDevice|TestTopology|TestSetDeviceLED|TestRebootRestore|TestUpgradeFirmware|TestGetFirmware'`
Expected: FAIL - method undefined on *SiteClient.

- [ ] **Step 3: Implement `site_device.go`**

Port `omadasiteclient.py` device/topology/LED/firmware methods; keep the upstream URL strings exactly (verify against the clone at `/tmp/opencode/tplink-omada-api` as you write each method).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./controller/ -race`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add controller/site_device.go controller/site_device_test.go
git commit -m "feat(controller): add site device, topology, LED and firmware methods"
```

---

### Task 12: SiteClient - access points

**Files:**
- Create: `controller/site_ap.go`
- Test: `controller/site_ap_test.go`

**Interfaces:**
- Consumes: `SiteClient`, AP models (Task 8), enums.
- Produces (methods on `*SiteClient`):
  - `AccessPoints(ctx context.Context) ([]AccessPoint, error)` - upstream `get_access_points`: `pagesV2(url("accessPoints", site), {deviceType:"accessPoint"})`, map through device-style `NewAccessPoint`.
  - `AccessPoint(ctx context.Context, deviceID string) (AccessPoint, error)` - `GET accessPoints/<id>`; empty id -> `ValidationError`.
  - `AccessPointsByMAC(ctx context.Context, macs []string) ([]AccessPoint, error)` - upstream `get_access_points_by_mac`: single request `accessPoints?mac=<comma-joined>`; empty slice -> `ValidationError`.
  - `ApPorts(ctx context.Context, deviceID string) ([]ApPortSettings, error)` - define `ApPortSettings` (in `site_ap.go`): `Index() int`, `Enabled() bool`, `Name() string`, `Type() ApPortType`, `PoE() bool`, `Speed() string` - from upstream `get_ap_ports` (url `accessPoints/<id>/portConfig` list).
  - `SetApPortEnabled(ctx context.Context, deviceID string, portIndex int, enabled bool, poe bool) error` - upstream `set_ap_port_enabled`: `PATCH accessPoints/<id>/portConfig` body `{"index": n, "enable": bool, "poeEnable": bool}`-shape (port the exact keys from upstream).
  - `ApRadioSettings(ctx context.Context, deviceID string, radioID int) (ApRadioSettings, error)` - upstream `get_ap_radio_settings`: `GET accessPoints/<id>/radioConfig` filtered by `radId`; miss -> `ValidationError`.
  - `SetApRadioEnabled(ctx context.Context, deviceID string, radioID int, enabled bool) error` - upstream `set_ap_radio_enabled`: `PATCH accessPoints/<id>/radioConfig` body per upstream keys.
  - `SetApRadioChannel(ctx context.Context, deviceID string, radioID int, channel int, channelWidth ChannelWidth) error` - upstream `set_ap_radio_channel`: fetch current settings first, send `PATCH` with full updated settings (upstream merges and re-sends the whole radio config object).
  - `ApAvailableChannels(ctx context.Context, deviceID string, radioID int) (ApRadioChannels, error)` - upstream `get_ap_available_channels`: `GET accessPorts/<id>/...` per upstream URL - **verify the exact upstream URL string** before implementing.

- [ ] **Step 1: Write the failing tests**

```go
func TestAccessPoints(t *testing.T) { /* deviceType=accessPoint query; list -> models */ }
func TestAccessPointEmptyID(t *testing.T) { /* *ValidationError */ }
func TestAccessPointsByMAC(t *testing.T) {
    // two macs -> single request with comma-joined param; empty slice -> *ValidationError
}
func TestApPorts(t *testing.T) { /* portConfig list -> ApPortSettings slice */ }
func TestSetApPortEnabled(t *testing.T) { /* PATCH body keys asserted verbatim */ }
func TestApRadioSettingsMiss(t *testing.T) { /* radId not in list -> *ValidationError */ }
func TestSetApRadioEnabled(t *testing.T) { /* PATCH body */ }
func TestSetApRadioChannelMerges(t *testing.T) {
    // GET radioConfig returns radio {radId:0, channel:1, ...}; PATCH body contains
    // ALL original fields plus updated channel/width (upstream merge behavior)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./controller/ -run 'TestAccessPoint|TestAp'`
Expected: FAIL - methods undefined.

- [ ] **Step 3: Implement `site_ap.go`**

Port `omadasiteclient.py` AP methods; assert each URL and body key against the clone. `SetApRadioChannel` fetch-merge-patch must preserve unknown fields (round-trip the raw map, mutate only channel/width).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./controller/ -race`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add controller/site_ap.go controller/site_ap_test.go
git commit -m "feat(controller): add access point methods"
```

---

### Task 13: SiteClient - switches and port profiles

**Files:**
- Create: `controller/site_switch.go`
- Test: `controller/site_switch_test.go`

**Interfaces:**
- Consumes: `SiteClient`, switch models (Task 8), enums.
- Produces (methods on `*SiteClient`):
  - `Switches(ctx context.Context) ([]Switch, error)` - upstream `get_switches`: devices filtered by switch type per upstream query, `NewSwitch`.
  - `Switch(ctx context.Context, deviceID string) (Switch, error)` - `GET switches/<id>` (verify upstream URL), empty id -> `ValidationError`.
  - `SwitchesByMAC(ctx context.Context, macs []string) ([]Switch, error)` - upstream `get_switches_by_mac` (comma-joined mac param); empty slice -> `ValidationError`.
  - `SwitchPorts(ctx context.Context, deviceID string) ([]SwitchPortSettings) (error)` - define `SwitchPortSettings` in this file: `Index() int`, `Name()`, `Mode() PortMode`, `VLANType() PortVlanType`, `PVID() int`, `TaggedVLANs() []int`, `Untagged() *int`, `PoE() bool`, `Enabled() bool` - from upstream `get_switch_ports` (url `switches/<id>/portConfig`).
  - `SetSwitchPortSettings(ctx context.Context, deviceID string, portIndex int, mode PortMode, pvid int, tagged []int, untagged *int, poe *bool) error` - upstream `set_switch_port_settings`: fetch current config, merge, `PATCH switches/<id>/portConfig` with the full object (upstream re-sends the merged config); all pointers nil and no changes -> `ValidationError`.
  - `PortProfiles(ctx context.Context) ([]PortProfile, error)` - upstream `get_port_profiles` (url `portProfiles`), paginated.
  - `PortProfile(ctx context.Context, id string) (PortProfile, error)` - `GET portProfiles/<id>`; empty -> `ValidationError`.
  - `CreatePortProfile(ctx context.Context, profile PortProfileOverrides) (PortProfile, error)` - define `PortProfileOverrides`: `Name`, `Description string`, `Ports []int`, `PVID int`, `VLANs []int` - upstream `create_port_profile`: body per upstream keys to `portProfiles`; empty name -> `ValidationError`.
  - `UpdatePortProfile(ctx context.Context, id string, profile PortProfileOverrides) (PortProfile, error)` - upstream `update_port_profile` (`PATCH portProfiles/<id>`, partial - only set fields in body).
  - `DeletePortProfile(ctx context.Context, id string) error` - upstream `delete_port_profile` (`DELETE portProfiles/<id>`); empty id -> `ValidationError`.
  - `ApplyPortProfile(ctx context.Context, id string, ports []int) error` - upstream `apply_port_profile`: body `{"profileId": id, "portIndexes": ports}` to `portProfiles/<id>/cmd/apply`; empty ports -> `ValidationError`.
  - **Version gate (review focus 5):** upstream picks the old v1-style port-config route vs the newer v2-style one at controller version `6` (`port_config` URL changes). Port the gate: `siteVersion >= 6` -> new route, else legacy; assert both branches in tests with versions `5.15` and `6.0`.

- [ ] **Step 1: Write the failing tests**

```go
func TestSwitches(t *testing.T) { /* switchType query per upstream; models */ }
func TestSwitchEmptyID(t *testing.T) { /* *ValidationError */ }
func TestSwitchesByMAC(t *testing.T) { /* comma-joined; empty -> *ValidationError */ }
func TestSwitchPorts(t *testing.T) { /* portConfig list -> SwitchPortSettings */ }
func TestSetSwitchPortSettingsMerge(t *testing.T) {
    // GET returns full config; PATCH body = original fields + updated mode/pvid/tagged
    // no-change call -> *ValidationError
}
func TestPortProfileCRUD(t *testing.T) {
    // create: POST body keys; empty name -> *ValidationError
    // get: GET; empty id -> *ValidationError
    // update: PATCH with only set fields
    // delete: DELETE; apply: POST cmd/apply body
}
func TestPortConfigVersionGate(t *testing.T) {
    // v5.15 -> legacy path asserted; v6.0 -> new path asserted
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./controller/ -run 'TestSwitch|TestPortProfile|TestSetSwitchPort|TestPortConfigVersionGate'`
Expected: FAIL - methods undefined.

- [ ] **Step 3: Implement `site_switch.go`**

Largest porting payload task - port `omadasiteclient.py` switch methods one by one against the clone, keeping every key/URL verbatim. The version gate wraps the port-config URL builder.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./controller/ -race`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add controller/site_switch.go controller/site_switch_test.go
git commit -m "feat(controller): add switch and port profile methods"
```

---

### Task 14: SiteClient - gateway (PoE)

**Files:**
- Create: `controller/site_gateway.go`
- Test: `controller/site_gateway_test.go`

**Interfaces:**
- Consumes: `SiteClient`, gateway models (Task 8), enums.
- Produces (methods on `*SiteClient`):
  - `Gateways(ctx context.Context) ([]Gateway, error)` - upstream `get_gateways` (devices filtered by gateway type), `NewGateway`.
  - `Gateway(ctx context.Context, deviceID string) (Gateway, error)` - `GET gateways/<id>` (verify upstream URL), empty id -> `ValidationError`.
  - `GatewayPorts(ctx context.Context, deviceID string) ([]GatewayPortSettings, error)` - define `GatewayPortSettings` in this file: `Index() int`, `Type() GatewayPortType`, `PoE() GatewayPoEStatus`, `Speed() string`, `Enabled() bool` - from upstream `get_gateway_ports`.
  - `SetGatewayPortSettings(ctx context.Context, deviceID string, portIndex int, poe GatewayPoEStatus, speed string, enabled *bool) error` - upstream `set_gateway_port_settings`: merged fetch-then-PATCH per upstream (body keys verbatim); no effective change -> `ValidationError`.
  - `SetGatewayPoE(ctx context.Context, deviceID string, portIndex int, state GatewayPoEStatus) error` - upstream `set_gateway_poe`: body per upstream to the gateway port endpoint.

- [ ] **Step 1: Write the failing tests**

```go
func TestGateways(t *testing.T) { /* gateway-type filter; models */ }
func TestGatewayEmptyID(t *testing.T) { /* *ValidationError */ }
func TestGatewayPorts(t *testing.T) { /* portConfigs -> GatewayPortSettings */ }
func TestSetGatewayPortSettingsMerge(t *testing.T) {
    // GET full config; PATCH = merged; no-op call -> *ValidationError
}
func TestSetGatewayPoE(t *testing.T) { /* body keys asserted */ }
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./controller/ -run 'TestGateway'`
Expected: FAIL - methods undefined.

- [ ] **Step 3: Implement `site_gateway.go`**

Port the upstream gateway methods verbatim (URLs + keys).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./controller/ -race`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add controller/site_gateway.go controller/site_gateway_test.go
git commit -m "feat(controller): add gateway and PoE methods"
```

---

### Task 15: SiteClient - LAN networks and VLANs

**Files:**
- Create: `controller/site_lan.go`
- Test: `controller/site_lan_test.go`

**Interfaces:**
- Consumes: `SiteClient`, network models (Task 8), enums.
- Produces (methods on `*SiteClient`):
  - `LanNetworks(ctx context.Context) ([]LanNetwork, error)` - upstream `get_lan_networks` (url `lanNetworks`), paginated.
  - `LanNetwork(ctx context.Context, id string) (LanNetwork, error)` - `GET lanNetworks/<id>`; empty id -> `ValidationError`.
  - Request types (in this file):
    - `LanNetworkCreate { Name string; VLAN int; Subnet, Mask, Gateway string; DHCPEnabled *bool; StartIP, EndIP string; DNSServers []string }`
    - `LanNetworkUpdate` - same fields, all pointers (partial update); `ID() string`.
    - `VlanCreate { VLAN int; Name string }`, `VlanUpdate { VLAN int; Name *string; Status *bool }`, `VlanDelete { VLAN int }`.
  - `CreateLanNetwork(ctx context.Context, create LanNetworkCreate) (LanNetwork, error)` - upstream `lan_network_create`: validate (empty name -> `ValidationError`; VLAN 0/1 -> `ValidationError` "VLAN 1 is protected" per upstream), `POST lanNetworks` with upstream body keys, return the created network.
  - `UpdateLanNetwork(ctx context.Context, update LanNetworkUpdate) (LanNetwork, error)` - upstream `lan_network_update`: `PUT lanNetworks/<id>` with only set fields; no fields set -> `ValidationError`.
  - `PreviewVlanCreate(ctx context.Context, create VlanCreate) (map[string]any, error)` - upstream `vlan_preview_create`: `POST vlan/preview` per upstream body; returns the raw preview result.
  - `VlanCreate(ctx context.Context, create VlanCreate) (map[string]any, error)` - upstream `vlan_create`: `POST vlans`.
  - `PreviewVlanUpdate(ctx context.Context, update VlanUpdate) (map[string]any, error)` - upstream `vlan_preview_update`.
  - `VlanUpdate(ctx context.Context, update VlanUpdate) (map[string]any, error)` - upstream `vlan_update`.
  - `PreviewVlanDelete(ctx context.Context, del VlanDelete) (map[string]any, error)` - upstream `vlan_preview_delete`.
  - `VlanDelete(ctx context.Context, del VlanDelete) (map[string]any, error)` - upstream `vlan_delete`.
  - **Read-back contract (review focus 4):** every `Vlan*` write method, after a successful 200, performs a verification read (`GET vlans?vlan=<n>` / the network read). If the read fails or the VLAN/network is missing: return an error containing the substring `write may have succeeded; inspect controller state before retrying` and do NOT retry the write. If the read confirms the change: return the result. Implement as `verifyVlan(ctx, vlan int) error` / `verifyLanNetwork(ctx, id string) error` helpers.

- [ ] **Step 1: Write the failing tests**

```go
func TestLanNetworks(t *testing.T) { /* list + get; empty id -> *ValidationError */ }
func TestCreateLanNetworkValidation(t *testing.T) {
    // "" name, VLAN 0, VLAN 1 -> *ValidationError (VLAN 1 message contains "protected")
}
func TestCreateLanNetwork(t *testing.T) { /* POST body keys per upstream; returns model */ }
func TestUpdateLanNetworkPartial(t *testing.T) {
    // only VLAN set -> PUT body contains only vlan key + id; all-nil -> *ValidationError
}
func TestVlanCreatePreviewAndWrite(t *testing.T) {
    // preview POST body; create POST body; verify read happens after write (GET vlans?vlan=)
}
func TestVlanWriteSuccessReadback(t *testing.T) {
    // write 200 + read confirms -> success, no retry (write endpoint hit once)
}
func TestVlanWriteReadbackFailure(t *testing.T) {
    // write 200; read fails/missing -> error message contains
    // "write may have succeeded; inspect controller state before retrying";
    // write endpoint hit exactly once (no auto-retry)
}
func TestVlanDelete(t *testing.T) { /* delete + preview delete; readback contract */ }
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./controller/ -run 'TestLanNetwork|TestCreateLan|TestUpdateLan|TestVlan'`
Expected: FAIL - methods undefined.

- [ ] **Step 3: Implement `site_lan.go`**

Port `omadasiteclient.py` LAN/VLAN methods verbatim. Validation messages stay close to upstream wording. The read-back verification is the one deliberate divergence from upstream (upstream trusts the 200 silently... upstream actually re-reads in some paths - match upstream where it re-reads, add the explicit guidance message per spec decision 3 where it does not).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./controller/ -race`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add controller/site_lan.go controller/site_lan_test.go
git commit -m "feat(controller): add LAN network and VLAN methods with read-back contract"
```

---

### Task 16: SiteClient - DHCP and VPN

**Files:**
- Create: `controller/site_dhcp.go`, `controller/site_vpn.go`
- Test: `controller/site_dhcp_test.go`, `controller/site_vpn_test.go`

**Interfaces:**
- Consumes: `SiteClient`, network/VPN models (Task 8), enums.
- Produces (methods on `*SiteClient`):

DHCP (`site_dhcp.go`), porting upstream `get_dhcp_reservations`/`create_dhcp_reservation`/`update_dhcp_reservation`/`delete_dhcp_reservation`:
  - `DhcpReservations(ctx context.Context) ([]DhcpReservation, error)` - `openapiURL("v1", "setting/service/dhcp/reservation", site)`, paginated (`pagesOpenAPIGet`), models.
  - `DhcpReservation(ctx context.Context, ip string) (DhcpReservation, error)` - filter from list (upstream behavior); missing -> `ValidationError`.
  - `CreateDhcpReservation(ctx context.Context, host, ip, mac string) (DhcpReservation, error)` - `POST` body per upstream keys (`hostName`/`ip`/`mac`); empty any -> `ValidationError`.
  - `UpdateDhcpReservation(ctx context.Context, r DhcpReservationUpdate) (DhcpReservation, error)` - define `DhcpReservationUpdate { IP string; Host *string; MAC *string }`; `PATCH` per upstream; no fields -> `ValidationError`.
  - `DeleteDhcpReservation(ctx context.Context, ip string) error` - `DELETE` per upstream; empty ip -> `ValidationError`.

VPN (`site_vpn.go`), porting upstream `vpn.py`-shaped methods (`get_vpn_policies`, `set_vpn_policy_enabled`):
  - `VpnPolicies(ctx context.Context) ([]VpnPolicy, error)` - upstream `get_vpn_policies`: fetches each VPN category (the VpnCategory enum values) via its list endpoint; **defect fix 2**: collect ALL pages per category (upstream fetched only page 1 with `pageSize=100`) using `pagesOpenAPIPost`/`pagesOpenAPIGet` per upstream route shape.
  - `VpnPolicy(ctx context.Context, id string) (VpnPolicy, error)` - from the collected list; missing -> `ValidationError`.
  - `SetVpnPolicyEnabled(ctx context.Context, id string, enabled bool) (VpnPolicy, error)` - upstream `set_vpn_policy_enabled`: body per upstream to the policy endpoint; empty id -> `ValidationError`.

- [ ] **Step 1: Write the failing tests**

```go
func TestDhcpReservations(t *testing.T) {
    // openapi v1 route asserted; paginated collect; models
}
func TestDhcpCrud(t *testing.T) {
    // create body {hostName,ip,mac}; "" fields -> *ValidationError
    // update partial PATCH; delete DELETE; get-by-ip miss -> *ValidationError
}
func TestVpnPoliciesAllPages(t *testing.T) {
    // category list endpoint returns 100 + 30 -> 130 policies for that category (defect fix 2)
    // all categories requested; total collected == sum
}
func TestVpnPolicyMiss(t *testing.T) { /* *ValidationError */ }
func TestSetVpnPolicyEnabled(t *testing.T) {
    // body per upstream; empty id -> *ValidationError; returns updated policy
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./controller/ -run 'TestDhcp|TestVpn'`
Expected: FAIL - methods undefined.

- [ ] **Step 3: Implement `site_dhcp.go` and `site_vpn.go`**

Verify every route string and body key against the upstream clone (`vpn.py`, `omadasiteclient.py` DHCP section).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./controller/ -race`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add controller/site_dhcp.go controller/site_dhcp_test.go controller/site_vpn.go controller/site_vpn_test.go
git commit -m "feat(controller): add DHCP reservation and VPN policy methods"
```

---

### Task 17: README, final lint, full test suite

**Files:**
- Create: `README.md`

**Interfaces:**
- Consumes: the complete package.
- Produces: a user-facing README and a green final gate.

- [ ] **Step 1: Write `README.md`**

Sections:
1. **What it is** - Go client for the TP-Link Omada SDN controller API; synchronous port of `tplink-omada-client` v1.5.14 (controller scope).
2. **Supported** - Omada Software Controller 5.1.0 and later; list the method families (sites, clients, devices, topology, APs, switches/port profiles, gateways/PoE, LAN networks/VLANs, DHCP, VPN, firmware).
3. **Quick start** - `New(url, user, pass)`, `Sites`, `SiteClient`, one VLAN example, one client-block example.
4. **Concurrency** - `Client`/`SiteClient` are safe for concurrent use; auth transitions are serialized.
5. **Errors** - `*RequestError` (controller error codes, `errors.Is` against `ErrLoginFailed` et al.), `*ValidationError` (caller input), sentinel list.
6. **Divergences from the Python library** - the three defect fixes (bounded pagination, `VpnPolicies` all-pages, re-login scope) + the VLAN read-back contract message; plus the structural notes (single `Device` struct, flat `Client` struct, `ValueError` family folded into `*ValidationError`).
7. **Phase 2** - the northbound cloud API (`tplinkcloud.com`) is intentionally out of scope.

- [ ] **Step 2: Run the full gate**

```bash
gofmt -l .            # expected: no output
go vet ./...          # expected: clean
go test ./... -race -count=1
```

Fix anything that surfaces (formatting, vet, races).

- [ ] **Step 3: Commit**

```bash
git add README.md .
git commit -m "docs: add README with usage, concurrency and divergence notes"
```

- [ ] **Step 4: Final verification**

```bash
go test ./... -race -count=1   # full suite green
git log --oneline              # one commit per task, 17 + this one
```
