# Omada Cloud Northbound Client — Design (Phase 2)

Status: DRAFT — written 2026-10-31. Follows the approved Phase 1 design
(`2026-10-29-omada-go-client-design.md`), which reserved the `cloud/` package
for exactly this work.

## Intent and why

A Go client package for the TP-Link Omada **cloud northbound API**
(`use1-omada-northbound.tplinkcloud.com`), generated from the public OpenAPI
spec the service publishes at `/v3/api-docs`. Phase 1 (the local controller
client in `controller/`) is complete; this is the Phase 2 scope the original
design doc named as a non-goal.

The published spec is the single source of truth. It is complete and standard
(OpenAPI 3.0.1, "Omada Open API v0.1"), so the client is **generated**, not
hand-written.

## User decisions

- Implement the **entire public northbound spec** — all operations under
  `/openapi/v1|v2|v3/{omadacId}/...`.
- Use an **off-the-shelf Go generator** (not a custom one).
- **oapi-codegen is fine as a dependency** — the generated client may import
  `github.com/oapi-codegen/runtime`. The `controller/` package stays
  stdlib-only; this dependency is scoped to `cloud/`.
- **Group the code logically** — one package per domain, not one giant file.

## What the spec contains (verified 2026-10-31)

Fetched from `https://use1-omada-northbound.tplinkcloud.com/v3/api-docs`
(4.7 MB, OpenAPI 3.0.1):

- 1,936 paths / 2,538 operations.
- Namespaces: `/openapi/v1/{omadacId}` (1,826 paths), `/openapi/v2/{omadacId}`
  (76), `/openapi/v3/{omadacId}` (8), plus 26 `/internal` paths (28 ops) that
  are server-internal and **excluded**.
- **2,510 public operations** are in scope.
- 3,484 component schemas; 167 tags.
- No security scheme is declared in the spec; real auth is OAuth2-style
  client credentials: `POST {base}/v1/token` with a `{"username","password"}`
  JSON body returning an access token (`access_token`/`expires_in`). The
  endpoint exists (probed; failure mode is a redirect, surfaced as an error).
  The token URL is configurable so other regions work.
- **Auth failure modes (verified against the live service, 2026-10-31):**
  - Missing token → `HTTP 401` with envelope
    `{"errorCode": -44116, "msg": "Open API Authorized failed..."}`.
  - Business errors (unknown omadacId, bad request data, ...) →
    `HTTP 200` with a non-zero `errorCode` envelope (same 200+envelope
    convention as the local controller API).
  - The spec declares only `200` responses for all operations, so 401s are
    "unexpected status" to the generated clients — the transport hook (below)
    owns them.
- **Spec defect (known):** 16 operations are missing path-parameter
  declarations (e.g. `{mac}` in `.../health/incident/{mac}`). oapi-codegen
  refuses the spec as-is. A deterministic patch script (checked into the
  repo) adds the missing `in: path` params (`required: true`, `string`).
  Nothing else in the spec is modified.

## Goals

- A `cloud/` module tree with a generated client covering all 2,510 public
  operations, grouped into logical domain packages.
- A thin hand-written `cloud` root package: construction, base URL, token
  provisioning (fetch, cache, refresh), and auth header injection.
- Reproducible generation: pinned generator version + checked-in spec patch
  script + per-package configs; generated code is checked in (no build-time
  codegen needed to compile the module).
- `go vet` + `gofmt` clean; `go test ./... -race -count=1` green.

## Non-goals

- The local controller API (done, in `controller/`) and any changes to it.
- Hand-writing operation wrappers — operations come from the generator.
- Webhooks, event streams, or anything not in the spec.
- Region auto-detection (caller passes the base URL; default is use1).
- Retries with backoff beyond a single 401 token-refresh retry.
- A CLI.

## Constraints

- Go stdlib + `github.com/oapi-codegen/runtime` (generated code) only. No
  other third-party runtime dependencies anywhere in the module.
- Generated code must compile and vet clean as generated (gofmt applied).
- No import cycles: `cloud` root → domain packages; domain packages are
  independent of each other (shared types are duplicated per package, which
  is expected and fine).
- No duplication between `controller/` and `cloud/`: no cross-imports in
  either direction, no shared types, no copy-pasted code. The two trees cover
  the same domains against two different services with different wire
  formats, so similarly-named types are intentional and never merged.
- The spec patch script must be pure and deterministic (same input → same
  output), and must fail loudly if the upstream spec changes shape in a way
  the patch doesn't handle.

## Architecture

### Layout

```
cloud/
  cloud.go            # Client, New, token provider, auth RequestEditor
  token.go            # client-credentials token flow (fetch/cache/refresh)
  paging.go           # small generic all-pages helper
  site/               # generated: Site + templates + customers
  client/             # generated: clients, insight-clients, vouchers
  device/             # generated: devices, firmware, backup
  ap/                 # generated: access points, WLAN optimization
  switch/             # generated: switches, QoS, MLAG, OUI VLAN
  gateway/            # generated: gateways, SD-WAN, OSPF, VRRP, IPTV,
                      #            cert profiles, SNMP
  lan/                # generated: wired networks, routing, NAT, DNS,
                      #            DHCP reservations, multicast
  wireless/           # generated: wireless networks, profiles
  vpn/                # generated: VPN, SSL VPN, Wireguard
  security/           # generated: ACLs, app control, IDS/IPS, filtering,
                      #            firewall, bandwidth, MAC/IP binding
  insight/            # generated: dashboards, stats, reports, health
  user/               # generated: users, roles, SSO, auth, controller
                      #            settings, app management
  msp/                # generated: MSP settings, users, logs, licenses
  stack/              # generated: stacking + OLT/GPON
  voip/               # generated: VoIP, Bluetooth
  sim/                # generated: SIM
  log/                # generated: logs, audit, CLI, webhooks
  topology/           # generated: topology
  remote/             # generated: remote access, SSH, UPnP
gen/
  patch_spec.py       # deterministic spec patch (missing path params)
  tags.json           # tag -> package mapping (the grouping table)
  configs/<pkg>.yaml  # one oapi-codegen config per domain package
  Makefile            # make gen: patch spec, generate all packages
```

### Tag → package grouping

19 packages, 167 tags, every one of the 2,510 public operations assigned to
exactly one package (verified by script; the Makefile target re-verifies and
fails if an operation is unassigned or double-assigned):

| package | ops | tags |
|---|---|---|
| lan | 254 | Wired Network(+Template), Routing(+T), NAT(+T), Disable Nat, DNS(+T), mDNS(+T), Lan Multicast(+T), DHCP Reservation |
| security | 233 | Application Control, ACL(+T), URL Filtering(+T), IDS/IPS(+T), Wireless IDS/IPS(+T), Global Security, Threat Management, Attack Defense(+T), Firewall(+T), Access Control(+T), MAC Filtering(+T), Bandwidth Control(+T), Session Limit(+T), IP-MAC Binding(+T), Dhcp Snooping, Arp Detection, shield |
| wireless | 213 | Wireless Network(+Template), Profiles(+Template) |
| stack | 186 | Stack, OLT, OLT ONU Register/Management, OLT Service Port(+Profile), OLT Pon/ONT Port, OLT Traffic/Service Profile, OLT T-cont/Gem Port/Gem Mapping/Line Profile/DBA |
| switch | 168 | Switch(+Template), Switch QoS, Mlag, OUI Based VLAN(+Template) |
| insight | 153 | Insight, Dashboard, Global Dashboard Overview, Statistic, Report v2, Network Report, Network Analyze, Health, Incidents(+Template), History Data Retention |
| user | 140 | User and Role, Local User, Cloud User, Authentication(+T), SSO, Form Auth Data, License, uisetting, System Settings, Controller Settings, APP Controller Settings, app |
| device | 140 | Device, Intelli Recover Device, Device Management Setting, Firmware, Backup and Restore, Backup schedule, Schedule(+Template) |
| site | 138 | Site, Site Template(+Configuration), Site Configuration, MSP Site, Customer, Batch OpenAPI, MSP Batch OpenAPI, Data Export, Quick Action, Cluster, Osw Port Layout, Msp Osw Port Layout |
| gateway | 124 | Gateway(+Template), Gateway QoS(+Template), SD-WAN, OSPF, VRRP, EoGRE Tunnel, IPTV(+Template), CertProfiles(+Template), SNMP(+Template) |
| voip | 112 | VoIP(+Template), Bluetooth(+Template) |
| ap | 112 | Ap, WLAN Optimization, Rrm, Wi-Fi Calling Traffic |
| vpn | 111 | VPN, SSL VPN, Wireguard VPN |
| client | 108 | Client, Client Insight, Authorized Client, Intelli Recover Client, Voucher, Hotspot Operators |
| msp | 92 | Msp setting, MSP User and Role, Msp Log, MSP SSO, Msp License, MSP Device, MSP History Data Retention, Msp Webhook Setting |
| log | 58 | Log(+Template), Audit Log(+Template), CLI(+Template), Webhook Setting |
| sim | 45 | SIM(+Template) |
| topology | 21 | Topology |
| remote | 19 | Remote Access, SSH(+Template), UPnP(+Template), Service(Template) |

Template variants (site templates) stay in the same package as their live
counterparts; generated operation names keep the spec's operationIds, so
template vs live operations remain distinguishable.

### Root `cloud` package (hand-written)

- `cloud.Client`: base URL (default `https://use1-omada-northbound.tplinkcloud.com`,
  overridable), an `*http.Client`, and a token provider.
- Token provider: `POST {base}/v1/token` with JSON `{"username","password"}`;
  caches the token until ~60 s before `expires_in`; a `RequestEditor`
  injects `Authorization: Bearer <token>`; on an HTTP 401 response the
  client refreshes once and retries.
- Accessors: one method per domain package returning the generated client
  pre-wired with the base URL and the auth `RequestEditor` (e.g.
  `Site() site.Client`, `Clients() client.Client`). Exact accessor naming is
  a builder call; keep it unambiguous.
- `paging.go`: a small generic all-pages helper for `page`/`pageSize`-style
  endpoints (the spec's dominant pagination shape): takes a fetch function,
  collects all pages bounded by `totalRows` where present (same
  bounded-collector discipline as `controller/conn/pagination.go`).
- Envelope handling: every northbound response body is the
  `{"errorCode", "msg", "result"}` envelope (generated as a typed struct per
  package). `cloud` provides `APIError{Code int; Msg string}` implementing
  `error` plus a tiny helper (`cloud.NewAPIError(code, msg)`) so callers can
  uniformly detect business errors: a `200` response with
  `errorCode != 0` is an API failure, not success. Callers check it after
  each operation; the client does not wrap every generated method.

### Generation workflow

- `gen/patch_spec.py`: reads the raw spec, adds the missing path params
  (16 ops as of 2026-10-31; the script is data-driven, not hardcoded to 16),
  writes the patched spec. Fails if it encounters an unexpected defect class.
- `gen/tags.json`: the tag→package table above.
- `gen/configs/<pkg>.yaml`: per-package oapi-codegen config —
  `generate: {client: true, models: true}`,
  `output-options: {include-tags: <tags of package>,
  resolve-type-name-collisions: true}`, `package: <pkg>`,
  `output: cloud/<pkg>/<pkg>.gen.go`.
- `make gen`: patch spec → verify every public op maps to exactly one package
  → generate all 19 packages → gofmt.
- Generator pinned: **oapi-codegen v2.8.1** (verified: generates all 2,510
  ops, compiles, vets clean; output ~530k lines total).
- Generated files are checked in. Regeneration is a maintainer action.

### Tests

- Root `cloud` package: token flow (fetch, cache hit, expiry refresh, 401
  retry-once, token-endpoint failure), base URL handling, RequestEditor
  injection — against `httptest` mocks.
- Each domain package: representative `httptest` tests (a few operations per
  package: GET with path+query params, POST with body, response decode, 4xx
  error path). Not every operation — the generator's output is uniform;
  coverage targets URL construction, param encoding, and decoding shape.
- `gen/`: a test that the tag→package mapping assigns all 2,510 public ops
  exactly once (runs against the patched spec).
- Gate: `gofmt -l .` empty, `go vet ./...` clean,
  `go test ./... -race -count=1` green.
- Caveat: the mock-only test plan cannot verify the full auth contract
  end-to-end. A manual smoke test with real credentials (token fetch → one
  read operation) is a recommended follow-up, not part of the automated gate.

## Builder calls (my decisions — flag if wrong)

1. Generated code is checked in (not generated at build time): the module
   compiles without `make` or network access.
2. One generated file per package (`<pkg>.gen.go`); no hand-written code
   inside domain packages (only the root `cloud` package is hand-written).
3. Type duplication across domain packages is accepted (self-contained
   packages; oapi-codegen prunes each package to the schemas it references).
4. The 26 `/internal` paths are excluded outright.
5. MSP operations get their own package even though some touch sites and
   customers (their paths live under `/msp/{mspId}/...` — a separate
   namespace with separate auth scoping).
6. The `gen/` directory (patch script, tags.json, configs, Makefile) lives at
   the repo root next to `cloud/`.
7. The spec is fetched fresh by the maintainer at regeneration time (the raw
   spec is **not** checked in — it's 4.7 MB and changes upstream); the
   patched spec is a build artifact in `gen/` (gitignored).

## What's left to the builder

- Exact accessor method names on `cloud.Client` for the 19 domain clients.
- Per-package representative test selection (which operations to test).
- The precise 401-retry semantics inside the generated clients (oapi-codegen
  RequestEditors run per-request; the retry hook lives in the shared
  `*http.Client` transport).
- Makefile ergonomics (parallel generation, single-package targets).
