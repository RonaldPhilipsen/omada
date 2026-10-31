# Omada Cloud Northbound Client Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A Go `cloud/` package tree with a generated client for all 2,510 public TP-Link Omada northbound API operations, grouped into 19 domain packages, plus a hand-written root package for construction, auth, paging, and envelope errors.

**Architecture:** The public OpenAPI spec (fetched from the live service) is patched by a deterministic script (it has 16 ops missing path-param declarations), then oapi-codegen v2.8.1 generates one client+models file per domain package, selected by tag. A thin hand-written `cloud` root package provides `Client` (base URL + OAuth2 client-credentials token flow + 401-refresh transport) and small helpers. Generated code is checked in.

**Tech Stack:** Go (stdlib + `github.com/oapi-codegen/runtime` v1.8.0 only), oapi-codegen v2.8.1 (generator binary, pinned), Python 3 stdlib (gen tooling only).

**Spec:** `docs/superpowers/specs/2026-10-31-omada-cloud-northbound-design.md`

## Global Constraints

- Module: `github.com/RonaldPhilipsen/omada` (repo root).
- Runtime dependencies: Go stdlib + `github.com/oapi-codegen/runtime` v1.8.0 only. `controller/` stays stdlib-only and is never modified.
- Generator pinned: oapi-codegen v2.8.1 (`$(go env GOPATH)/bin/oapi-codegen`).
- 19 domain packages, exactly the keys of `gen/tags.json`: `ap, client, device, gateway, insight, lan, log, msp, remote, security, sim, site, stack, switch, topology, user, voip, vpn, wireless`.
- Every public operation (2,510 of them; `/internal` paths excluded) maps to exactly one package; `gen/verify_mapping.py` enforces this and fails the build otherwise.
- Generated files: `cloud/<pkg>/<pkg>.gen.go`, checked in, gofmt-clean, no hand-written code inside domain packages.
- Raw spec (`gen/api-docs.json`) and patched spec (`gen/api-docs.patched.json`) are gitignored build artifacts; `gen/configs/` is derived from `gen/tags.json` and gitignored.
- Default base URL: `https://use1-omada-northbound.tplinkcloud.com`. Token endpoint: `POST {base}/v1/token` with JSON `{"username","password"}`.
- Auth failure modes (verified against the live service): missing token → HTTP 401 with envelope `errorCode -44116`; business errors → HTTP 200 with non-zero `errorCode` envelope (`{"errorCode", "msg", "result"}`).
- Gate (every task that touches Go code): `gofmt -l .` empty, `go vet ./...` clean, `go test ./... -race -count=1` green.

## Review Focus

Failure modes the spec implies but no single task's tests fully exercise; each line's test is pinned to the owning task:

1. **Token expires mid-session** — the cached token goes stale between requests; the next request gets 401; the client force-refreshes and retries exactly once, and if the second attempt is also 401 it surfaces an error without looping. (Task 4, Step 1.)
2. **HTTP 200 with non-zero `errorCode`** (e.g. `-7131` "Controller ID not exist") — must be detectable as an API failure, not treated as success; `cloud.NewAPIError` carries code+msg and implements `error`. (Task 5, Step 1.)
3. **Token endpoint failure** (non-200 or non-JSON body) — the first authenticated request fails with a clear Go error; no half-cached token is stored. (Task 3, Step 1.)
4. **Unexpected status from an operation** (500, HTML body) — generated clients declare only 200; the caller gets the oapi-codegen default response or an error, never a panic. (Task 6, Step 1, in the `site` package.)
5. **Misbehaving pagination** (server returns the same page forever, or `totalRows` never reached) — `cloud.CollectAll` stops at its `maxRows` bound instead of looping forever. (Task 5, Step 1.)

---

### Task 1: `gen/` tooling — fetch, patch, mapping, verify

**Files:**
- Create: `gen/fetch_spec.py`
- Create: `gen/patch_spec.py`
- Create: `gen/tags.json`
- Create: `gen/verify_mapping.py`
- Modify: `.gitignore`

**Interfaces:**
- Consumes: nothing (first task).
- Produces: `gen/api-docs.json` (raw spec, gitignored), `gen/api-docs.patched.json` (patched spec, gitignored); `gen/tags.json` mapping package name → list of tag strings; verify command `python3 gen/verify_mapping.py gen/api-docs.patched.json gen/tags.json` printing `OK: 2510 public operations, 167 tags, 19 packages` on success and exiting non-zero on any unassigned/double-assigned operation.

- [ ] **Step 1: Add gitignore entries**

Append to `.gitignore`:

```
gen/api-docs.json
gen/api-docs.patched.json
gen/configs/
```

- [ ] **Step 2: Write `gen/fetch_spec.py`**

Stdlib only (`urllib.request`). Reads the spec URL from argv[1] (default `https://use1-omada-northbound.tplinkcloud.com/v3/api-docs`) and output path from argv[2] (default `api-docs.json`, relative to the script's dir's parent — i.e. run from `gen/`). Writes the raw JSON bytes. Fails (non-zero exit) on non-200 status.

- [ ] **Step 3: Write `gen/patch_spec.py`**

Stdlib only. Args: input path, output path. Loads the OpenAPI JSON; for every path under `/openapi/` and `/internal/`, for every operation (`get`,`post`,`put`,`patch`,`delete`), compares `{param}` placeholders in the path against declared `in: path` parameters (from both the path item and the operation); appends the missing ones as `{"name": <n>, "in": "path", "required": true, "description": <n>, "schema": {"type": "string"}}` to the operation's `parameters`. Prints the number of params added. Fails non-zero if a path item has an operation with no `operationId` (guards against spec shape drift). Writes the patched JSON. Must be deterministic: running it twice on the same input produces byte-identical output (`json.dump` with `sort_keys=True`).

- [ ] **Step 4: Write `gen/tags.json`**

Exact content (167 tags, 19 packages — verified complete against the spec's tag set):

```json
{
 "ap": ["Ap", "Rrm", "Wi-Fi Calling Traffic", "WLAN Optimization"],
 "client": ["Authorized Client", "Client", "Client Insight", "Hotspot Operators", "Intelli Recover Client", "Voucher"],
 "device": ["Backup and Restore", "Backup schedule", "Device", "Device Management Setting", "Firmware", "Intelli Recover Device", "Schedule", "Schedule (Template)"],
 "gateway": ["CertProfiles", "CertProfiles (Template)", "EoGRE Tunnel", "Gateway", "Gateway (Template)", "Gateway QOS (Template)", "Gateway QoS", "IPTV", "IPTV (Template)", "OSPF", "SD-WAN", "SNMP", "SNMP (Template)", "VRRP"],
 "insight": ["Dashboard", "Global Dashboard Overview", "Health", "History Data Retention", "Incident (Template)", "Incidents", "Insight", "Network Analyze", "Network Report", "Report v2", "Statistic"],
 "lan": ["DHCP Reservation", "DNS", "DNS (Template)", "Disable Nat", "Lan Multicast", "Lan Multicast (Template)", "NAT", "NAT (Template)", "Routing", "Routing (Template)", "Wired Network", "Wired Network (Template)", "mDNS", "mDNS (Template)"],
 "log": ["Audit Log", "Audit Log (Template)", "CLI", "CLI (Template)", "Log", "Log (Template)", "Webhook Setting"],
 "msp": ["MSP Device", "MSP History Data Retention", "MSP SSO", "MSP User and Role", "Msp License", "Msp Log", "Msp Webhook Setting", "Msp setting"],
 "remote": ["Remote Access", "SSH", "SSH (Template)", "Service (Template)", "UPnP", "UPnP (Template)"],
 "security": ["ACL", "ACL (Template)", "Arp Detection", "Access Control", "Access Control (Template)", "Application Control", "Attack Defense", "Attack Defense (Template)", "Bandwidth Control", "Bandwidth Control (Template)", "Dhcp Snooping", "Firewall", "Firewall (Template)", "Global Security", "IDS/IPS", "IDS/IPS (Template)", "IP-MAC Binding", "IP-MAC Binding (Template)", "MAC Filtering", "MAC Filtering (Template)", "Session Limit", "Session Limit (Template)", "Threat Management", "URL Filtering", "URL Filtering (Template)", "Wireless IDS/IPS", "Wireless IDS/IPS (Template)", "shield"],
 "sim": ["SIM", "SIM (Template)"],
 "site": ["Batch OpenAPI", "Cluster", "Customer", "Data Export", "MSP Batch OpenAPI", "MSP Site", "Msp Osw Port Layout", "Osw Port Layout", "Quick Action", "Site", "Site Configuration", "Site Template", "Site Template Configuration"],
 "stack": ["OLT", "OLT DBA", "OLT Gem Mapping", "OLT Gem Port", "OLT Line Profile", "OLT ONT Port", "OLT ONU Management", "OLT ONU Register", "OLT Pon Port", "OLT Service Port", "OLT Service Port Profile", "OLT Service Profile", "OLT T-cont", "OLT Traffic Profile", "Stack"],
 "switch": ["Mlag", "OUI Based VLAN", "OUI Based VLAN (Template)", "Switch", "Switch (Template)", "Switch QoS"],
 "topology": ["Topology"],
 "user": ["APP Controller Settings", "Authentication", "Authentication (Template)", "Cloud User", "Controller Settings", "Form Auth Data", "License", "Local User", "SSO", "System Settings", "User and Role", "app", "uisetting"],
 "voip": ["Bluetooth", "Bluetooth(Template)", "VoIP", "VoIP (Template)"],
 "vpn": ["SSL VPN", "VPN", "Wireguard VPN"],
 "wireless": ["Profiles", "Profiles (Template)", "Wireless Network", "Wireless Network (Template)"]
}
```

- [ ] **Step 5: Write `gen/verify_mapping.py`**

Args: patched spec path, tags.json path. Loads both. Collects every operation (all 5 methods) on every non-`/internal` path; for each operation, the set of packages claiming it = packages whose tag list intersects the operation's `tags`. Fails (non-zero, with the offending operation listed) if any operation maps to 0 packages or to more than 1. On success prints `OK: <nops> public operations, <ntags> tags, <npkgs> packages` where `<nops>` is the operation count, `<ntags>` the number of distinct public tags, `<npkgs>` the number of packages.

- [ ] **Step 6: Run the pipeline and verify**

```bash
cd gen && python3 fetch_spec.py && python3 patch_spec.py api-docs.json api-docs.patched.json && python3 verify_mapping.py api-docs.patched.json tags.json
```

Expected: patch reports `patched params: 16`; verify prints `OK: 2510 public operations, 167 tags, 19 packages`.

Also verify determinism:

```bash
cd gen && python3 patch_spec.py api-docs.json /tmp/a.json && python3 patch_spec.py api-docs.json /tmp/b.json && diff /tmp/a.json /tmp/b.json && echo DETERMINISTIC
```

Expected: `DETERMINISTIC`, no diff output.

- [ ] **Step 7: Commit**

```bash
git add .gitignore gen/fetch_spec.py gen/patch_spec.py gen/tags.json gen/verify_mapping.py
git commit -m "feat: add northbound spec tooling (fetch, patch, tag mapping, verify)"
```

---

### Task 2: Generate the 19 domain packages

**Files:**
- Create: `gen/gen_configs.py`
- Create: `gen/Makefile`
- Create: `cloud/<pkg>/<pkg>.gen.go` for each of the 19 packages (generated, checked in)
- Modify: `go.mod`, `go.sum`

**Interfaces:**
- Consumes: Task 1's `gen/api-docs.patched.json`, `gen/tags.json`, verify command.
- Produces: 19 Go packages, each exposing the oapi-codegen **v2** surface: concrete `type Client struct` (fields `Server string`, `Client HttpRequestDoer`), `func NewClient(server string, opts ...ClientOption) (*Client, error)`, options `WithHTTPClient(doer HttpRequestDoer) ClientOption` and `WithRequestEditorFn(fn RequestEditorFn) ClientOption`, plus all generated model types. Package names: `cloud/switch/` declares `package switchpkg` (`switch` is a Go reserved word); `cloud/log/` declares `package log`. Task 4's accessors call `NewClient` with `WithHTTPClient` on each.

- [ ] **Step 1: Write `gen/gen_configs.py`**

Stdlib only. Reads `tags.json` from its own directory; for each package writes `configs/<pkg>.yaml` with exactly this shape:

```yaml
package: <pkg>
generate:
  client: true
  models: true
output: ../cloud/<pkg>/<pkg>.gen.go
output-options:
  include-tags:
    - <tag 1>
    - <tag 2>
  resolve-type-name-collisions: true
```

Tags in the order they appear in `tags.json`. Creates `configs/` if missing.

- [ ] **Step 2: Write `gen/Makefile`**

Targets:
- `fetch`: `python3 fetch_spec.py`
- `patch`: `python3 patch_spec.py api-docs.json api-docs.patched.json`
- `verify`: `python3 verify_mapping.py api-docs.patched.json tags.json`
- `configs`: `python3 gen_configs.py`
- `gen`: depends on `fetch patch verify configs`, then loops `for f in configs/*.yaml; do $(OAPI) -config $$f api-docs.patched.json; done` with `OAPI ?= $(shell go env GOPATH)/bin/oapi-codegen`, then `gofmt -w ../cloud`. (oapi-codegen v2 takes the spec path only as a CLI positional argument — there is no `input:` config field.)

- [ ] **Step 3: Add the runtime dependency**

```bash
go get github.com/oapi-codegen/runtime@v1.8.0
```

Expected: `go.mod` gains `github.com/oapi-codegen/runtime v1.8.0`.

- [ ] **Step 4: Generate**

```bash
make -C gen gen
```

Expected: 19 files `cloud/<pkg>/<pkg>.gen.go` exist; total ~530k lines; no oapi-codegen errors.

- [ ] **Step 5: Verify build and vet**

```bash
go build ./... && go vet ./...
```

Expected: both clean. (If oapi-codegen emits a compile error in any package, fix the config for that package — e.g. missing `resolve-type-name-collisions` — and re-run `make -C gen gen`; do not hand-edit generated files.)

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum gen/gen_configs.py gen/Makefile cloud/
git commit -m "feat: generate 19 northbound domain client packages from OpenAPI spec"
```

---

### Task 3: Token provider

**Files:**
- Create: `cloud/token.go`
- Test: `cloud/token_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks (pure stdlib).
- Produces:

```go
type TokenProvider struct { /* baseURL, username, password string, hc *http.Client, cache fields */ }
func NewTokenProvider(baseURL, username, password string, hc *http.Client) *TokenProvider
func (tp *TokenProvider) Token(ctx context.Context) (string, error)        // cached; refreshes when < 60s to expiry
func (tp *TokenProvider) Invalidate()                                      // drops the cached token
```

`hc` may be nil → `http.DefaultClient`. Token endpoint: `POST {baseURL}/v1/token`, JSON body `{"username": ..., "password": ...}`, response `{"access_token": "...", "expires_in": <seconds>}`. Cache valid until `now + expires_in - 60s`.

- [ ] **Step 1: Write the failing tests** in `cloud/token_test.go`

Using `httptest.Server`:

```go
func TestTokenFetchAndCache(t *testing.T) // token endpoint hit exactly once across two Token() calls
func TestTokenRefreshOnExpiry(t *testing.T) // first response expires_in=1 (already past the 60s skew), second call hits endpoint again
func TestTokenEndpointFailure(t *testing.T) // 500 response -> Token() returns error, second Token() retries (no poisoned cache)
func TestTokenEndpointBadJSON(t *testing.T) // 200 with non-JSON body -> Token() returns error
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./cloud/ -run TestToken -v
```

Expected: build failure (`undefined: NewTokenProvider`).

- [ ] **Step 3: Implement `cloud/token.go`**

Struct + three methods per the Interfaces block. Single mutex-protected cache (`token string`, `expiresAt time.Time`). `Token` returns the cache when `time.Now().Before(expiresAt)`; otherwise fetches. `Invalidate` sets `expiresAt` to the zero time.

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./cloud/ -run TestToken -race -count=1
```

Expected: PASS (all four).

- [ ] **Step 5: Commit**

```bash
git add cloud/token.go cloud/token_test.go
git commit -m "feat: add northbound token provider (client-credentials, cached)"
```

---

### Task 4: Client, auth transport, accessors

**Files:**
- Create: `cloud/cloud.go`
- Test: `cloud/cloud_test.go`

**Interfaces:**
- Consumes: `NewTokenProvider`/`TokenProvider` (Task 3); the 19 generated packages' `NewClient(server string, opts ...ClientOption) (*Client, error)` and `WithHTTPClient(doer HttpRequestDoer) ClientOption` (Task 2).
- Produces:

```go
type Options struct {
    BaseURL    string        // default https://use1-omada-northbound.tplinkcloud.com
    Username   string        // required
    Password   string        // required
    HTTPClient *http.Client  // optional
}
func New(opts Options) (*Client, error)
type Client struct { /* baseURL string, tp *TokenProvider, httpClient *http.Client */ }
func (c *Client) Site() *site.Client
func (c *Client) Clients() *client.Client
func (c *Client) Devices() *device.Client
func (c *Client) Ap() *ap.Client
func (c *Client) Switch() *switchpkg.Client   // import "github.com/RonaldPhilipsen/omada/cloud/switch" (declares package switchpkg)
func (c *Client) Gateway() *gateway.Client
func (c *Client) Lan() *lan.Client
func (c *Client) Wireless() *wireless.Client
func (c *Client) Vpn() *vpn.Client
func (c *Client) Security() *security.Client
func (c *Client) Insight() *insight.Client
func (c *Client) Users() *user.Client
func (c *Client) Msp() *msp.Client
func (c *Client) Stack() *stack.Client
func (c *Client) Voip() *voip.Client
func (c *Client) Sim() *sim.Client
func (c *Client) OmadaLog() *omadalog.Client  // import "cloud/log" as omadalog (declares package log; name clash with stdlib log)
func (c *Client) Topology() *topology.Client
func (c *Client) Remote() *remote.Client
```

Auth: an `authTransport` (`http.RoundTripper`) wraps the base transport; on each `RoundTrip` it calls `tp.Token(ctx)` and sets `Authorization: Bearer <token>`; if the response is 401 it calls `tp.Invalidate()`, re-fetches, clones the request with the new header, and retries once; a second 401 is returned as-is. `New` wires `c.httpClient = &http.Client{Transport: authTransport{...}}` (or wraps `opts.HTTPClient`'s transport when provided). Each accessor passes that client to the generated one via `WithHTTPClient`, so every generated operation flows through the auth transport.

- [ ] **Step 1: Write the failing tests** in `cloud/cloud_test.go`

Using `httptest.NewServer` as a combined token+API mock (route on path: `/v1/token` → token JSON; anything else → API):

```go
func TestNewRequiresCredentials(t *testing.T)        // empty Username/Password -> New returns error
func TestRequestCarriesBearerToken(t *testing.T)     // API handler asserts Authorization header == "Bearer tok-1"
func Test401TriggersRefreshAndRetry(t *testing.T)    // first API call -> 401, second -> 200 with {"errorCode":0}; assert 2 API hits + 2 token hits (Review Focus #1)
func Test401TwiceFails(t *testing.T)                 // API always 401; assert exactly 2 API hits, 3 token hits, returned response is 401 (no loop)
```

Note: these tests exercise the transport via a generated client — use `c.Site()` (or any accessor) with a real generated operation from `cloud/site/site.gen.go` (pick the first GET operation with a single path parameter so the test is short).

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./cloud/ -run 'TestNew|TestRequest|Test401' -v
```

Expected: build failure (`undefined: New`).

- [ ] **Step 3: Implement `cloud/cloud.go`**

`Options`, `New`, `Client`, `authTransport`, and the 19 accessors per the Interfaces block. Each accessor calls `pkg.NewClient(c.baseURL, pkg.WithHTTPClient(c.httpClient))` (no request editors needed — the transport owns the header), discards the returned error (it cannot fail for a valid base URL), and returns the concrete `*pkg.Client`. `New` returns an error for empty `Username`/`Password`; default `BaseURL` when empty (strip trailing slash).

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./cloud/ -race -count=1
```

Expected: PASS (token tests from Task 3 + all four new tests).

- [ ] **Step 5: Commit**

```bash
git add cloud/cloud.go cloud/cloud_test.go
git commit -m "feat: add cloud Client with bearer-token transport and 401 refresh"
```

---

### Task 5: Envelope errors and paging helper

**Files:**
- Create: `cloud/errors.go`
- Create: `cloud/paging.go`
- Test: `cloud/errors_test.go`
- Test: `cloud/paging_test.go`

**Interfaces:**
- Consumes: nothing beyond stdlib.
- Produces:

```go
type APIError struct {
    Code int
    Msg  string
}
func (e *APIError) Error() string                 // "omada: API error <Code>: <Msg>"
func NewAPIError(code int, msg string) *APIError

// Page is one page of results; TotalRows <= 0 means "unknown".
type Page[T any] struct {
    Rows      []T
    TotalRows int64
}
// CollectAll fetches pages via fetch(ctx, page, pageSize) starting at page 1
// until it has TotalRows rows (when known), a short page, or maxRows rows.
func CollectAll[T any](ctx context.Context, fetch func(ctx context.Context, page, pageSize int) (Page[T], error), pageSize int, maxRows int64) ([]T, error)
```

`CollectAll` guards: `pageSize < 1` → error; `maxRows < 1` → error; stops on first error from `fetch`; a page with `len(Rows) < pageSize` ends collection even if `TotalRows` is unknown.

- [ ] **Step 1: Write the failing tests**

`cloud/errors_test.go`:

```go
func TestAPIErrorMessage(t *testing.T)   // NewAPIError(-44116, "Open API Authorized failed...").Error() == "omada: API error -44116: Open API Authorized failed..." (Review Focus #2)
```

`cloud/paging_test.go`:

```go
func TestCollectAllStopsAtTotalRows(t *testing.T)   // 3 pages of 2, totalRows=4 -> 4 rows, 3 fetches
func TestCollectAllStopsOnShortPage(t *testing.T)   // totalRows unknown (0) -> stops when a page has < pageSize rows
func TestCollectAllBoundsMaxRows(t *testing.T)      // fetch always returns a full page with totalRows=999999; maxRows=5 -> returns 5 rows, no infinite loop (Review Focus #5)
func TestCollectAllInvalidArgs(t *testing.T)        // pageSize=0 and maxRows=0 -> errors
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./cloud/ -run 'TestAPIError|TestCollectAll' -v
```

Expected: build failure (`undefined: NewAPIError`, `undefined: CollectAll`).

- [ ] **Step 3: Implement `cloud/errors.go` and `cloud/paging.go`**

Per the Interfaces block. `CollectAll` is a loop: `page := 1; for { p, err := fetch(ctx, page, pageSize); ... append; stop conditions; page++ }`.

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./cloud/ -race -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cloud/errors.go cloud/paging.go cloud/errors_test.go cloud/paging_test.go
git commit -m "feat: add APIError envelope helper and bounded paging collector"
```

---

### Task 6: Representative domain package tests

**Files:**
- Test: `cloud/<pkg>/<pkg>_test.go` for each of the 19 packages

**Interfaces:**
- Consumes: the generated packages' `NewClient(server, opts ...ClientOption) (*Client, error)` and operation methods (Task 2); `cloud.NewAPIError` (Task 5) in the `site` package's envelope test.
- Produces: nothing (verification layer).

Per package, write one test file with **three tests**:

1. `Test<Op>GET` — pick the first GET operation in `<pkg>.gen.go` that has at least one path parameter and at least one query parameter (fall back to: first GET with a path parameter, then first GET). `httptest.Server` handler asserts the exact request path (with concrete param values) and one query key, returns a 200 with a minimal valid envelope `{"errorCode":0,"msg":"OK","result": <minimal value for the generated result type>}`. Assert the decoded response fields match.
2. `Test<Op>POST` — same selection rule for the first POST operation with a request body; handler asserts method, path, and a body field; returns a minimal 200 envelope; assert decode.
3. `Test<Op>EnvelopeError` (site package only; skip in the other 19 — the helper is shared) — handler returns 200 with `{"errorCode":-7131,"msg":"Controller ID not exist.","result":null}`; assert the caller can build `cloud.NewAPIError(resp.<envelope field>.ErrorCode, resp.<envelope field>.Msg)` and its `Error()` contains `-7131`.

For packages where no operation matches a selection rule (unlikely; `topology` has 21 ops), use the package's first GET.

Envelope field name: in generated code the 200 response struct wraps the body — inspect `<pkg>.gen.go` for the operation's response type and use its actual field (the generated envelope struct has `ErrorCode`, `Msg`, `Result` fields).

- [ ] **Step 1: Write tests for all 19 packages**

Start with `site` (includes the envelope test), then the remaining 18. The `site` GET test doubles as the Review Focus #4 check: add a fourth test `TestSiteUnexpectedStatus` — handler returns 500 with `text/html` body; assert the generated call returns without panicking (oapi-codegen returns the default-status response; assert `err == nil` and the default response is non-nil, or assert the returned error is non-nil — whichever the generated signature produces; document which in a one-line comment).

- [ ] **Step 2: Run the full gate**

```bash
gofmt -l . && go vet ./... && go test ./... -race -count=1
```

Expected: no unformatted files, vet clean, all tests PASS.

- [ ] **Step 3: Commit**

```bash
git add cloud/*/[a-z]*_test.go
git commit -m "test: add representative httptest coverage for all 19 domain packages"
```

---

### Task 7: README and final gate

**Files:**
- Modify: `README.md`

**Interfaces:**
- Consumes: everything above.
- Produces: documented `cloud/` usage.

- [ ] **Step 1: Add a `cloud/` section to `README.md`**

Document: what it is (generated northbound client, 2,510 operations, 19 domain packages); the one extra dependency (`github.com/oapi-codegen/runtime`); construction example (`cloud.New(cloud.Options{Username: ..., Password: ...})` then `cl.Site().<op>(...)`); the envelope convention (200 + `errorCode != 0` is a failure — use `cloud.NewAPIError`); the `cloud.CollectAll` paging helper; regeneration (`make -C gen gen`, requires oapi-codegen v2.8.1); the verified auth behavior (401 + refresh once).

- [ ] **Step 2: Final gate**

```bash
gofmt -l . && go vet ./... && go test ./... -race -count=1
```

Expected: no unformatted files, vet clean, all tests PASS.

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: document cloud northbound client"
```

---

## Post-implementation (not part of the gate)

- Manual smoke test with real credentials: token fetch → one read operation (e.g. list sites). The mock-only tests cannot verify the full live auth contract; this is a recommended follow-up, per the spec.
