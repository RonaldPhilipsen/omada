# Omada Go Client — Design (Phase 1: Controller)

Status: APPROVED — written 2026-10-29; builder check completed 2026-10-29;
user approved Phase 1 scope (port of the Python library, controller-only)
2026-10-29. The full tplinkcloud northbound OpenAPI spec (4.7 MB, 2538 public
ops: 2403 under /openapi/v1, 99 under /openapi/v2, 8 under /openapi/v3)
remains explicitly out of scope for phase 1 —
note that the `/openapi/v3/` namespace is only 8 ops; the "v3" in the
`/v3/api-docs` URL is the OpenAPI format version, not an API version.

## Intent and why

A Go client package for the TP-Link Omada SDN platform, starting as a faithful
port of the Python library `tplink_omada_client`
(https://github.com/MarkGodwin/tplink-omada-api, v1.5.14, MIT, Mark Godwin).
The Python library is controller-only; the cloud (tplinkcloud.com) northbound
API will be added later as a separate package in the same module.

## User decisions (from conversation, 2026-10-29)

- Support **both** the local Omada controller API and the cloud northbound API.
- **Phase 1 = port the reference (Python) library only**; cloud comes later.
- Module path: `github.com/RonaldPhilipsen/omada`.
- Location: new folder `~/Documents/Source/omada`.

## Builder decisions (my calls — flag if wrong)

- Module layout: root package files (`go.mod`, README, LICENSE) plus a
  `controller/` package now, so `cloud/` can slot in later without churn.
- **Stdlib only** — no third-party dependencies.
- **Synchronous API with `context.Context`** as first arg (Python is async;
  Go idiom differs).
- **No CLI in v1** — library only (the Python repo ships an `omada` CLI;
  that is out of scope for now).
- Go-idiomatic names: drop the `Omada` prefix where the package name carries
  the context (e.g. `controller.AccessPoint`, `controller.SiteClient`,
  `controller.Client` for a network client). Original names kept in docs.
- Paged endpoints return **fully collected slices** (Python uses async
  generators; Go returns `[]T, error`).
- Small internal **version comparator** replacing Python's `awesomeversion`,
  good enough for the specific comparisons needed (5.1.0, 6, 6.2.0.0, 6.3):
  dot-separated numeric segments, missing segment = 0.
- `LICENSE` = MIT with attribution to the original author; README notes the
  port and provenance (tested upstream on Omada 5.5.7–6.2.x, 5.0.x not
  compatible; VLAN contract live-tested upstream on 5.15.8.12).
- Null/optional semantics: settings structs use **pointers** for "not set"
  (mirrors Python `None`).
- Models keep their raw JSON (`map[string]any`) internally — exposed as
  `Raw()` — because VLAN update payloads and read-back verification rebuild
  from raw data.

## Builder check decisions (2026-10-29, reviewer subagent + builder calls)

Builder review (fresh subagent, verified against `release/v1.5.14` of the
upstream repo) confirmed the checklist is faithful and buildable and the
stdlib-only constraint holds. Five open questions were settled as builder
calls:

1. **Concurrency**: `Client`/`SiteClient` are safe for concurrent use; a
   mutex serializes login/re-login (csrf token, lastLogon, login state are
   mutable). Tested with `-race`.
2. **Validation errors**: a single typed `*ValidationError` (wraps,
   `errors.As`-able) stands in for the upstream `ValueError` family (~40
   distinct: VLAN name/ID/target/conflict/guards, channel selection, AP
   channel availability). The sentinel list in Errors is unchanged.
3. **Defects**: upstream defects are fixed, not ported — (a) pagination is
   bounded by `totalRows` (upstream keys the loop off `currentSize` and loops
   forever on an empty page with `totalRows > 0`); (b) `VpnPolicies` fetches
   all pages, not just one; (c) re-login triggers only on auth-class errors
   (upstream `_check_login` swallows everything). Divergences documented in
   the README.
4. **Duck-typed unions**: methods accepting `mac | Device | nil` take `any`
   and type-switch. The client hierarchy ports as a flat `Client` model with
   `*WiredDetails`/`*WirelessDetails` pointer fields (nil = not present), not
   an interface tree.
5. **VLAN preview helpers**: `PreviewVlanCreate`/`PreviewVlanUpdate`/
   `PreviewVlanDelete` are in scope — public upstream, the documented
   "preview by default" workflow, and exercised by the ported test suite.

## Phase 1 scope: feature parity checklist

### Connection layer (`controller.Connection`, unexported type)

- URL normalization: prepend `https://` when no scheme.
- Cookie-based auth (session set during `login`) + `Csrf-Token`,
  `Omada-Request-Source: web-local`, `Referer`, `Origin` headers on
  authenticated requests.
- Go's `http.CookieJar` policy rejects cookies for IP-address hosts; provide
  an "unsafe" jar equivalent (accept cookies on IP domains) matching the
  Python `CookieJar(unsafe=True)` behavior for IP-based controllers.
- Optional TLS skip-verify (`InsecureSkipVerify` option); allow injecting a
  custom `*http.Client`.
- Auto-login on first request; re-login on `LoginFailed` (errorCode -30109)
  or `LoginSessionClosed` (HTTP 200 with non-JSON body) and retry once.
- Login session considered valid for 1 hour, then verified via
  `loginStatus`.
- `login` flow: GET `/api/info` (unauthenticated) → version must be >= 5.1.0
  else `ErrUnsupportedVersion` → POST `login` → store controller ID + CSRF
  token.
- Response unwrapping: non-200 → parse JSON body for `errorCode`/`msg`;
  200 + non-JSON → `LoginSessionClosed`; JSON with `errorCode != 0` →
  `RequestError{Code, Msg}` (-30109 → login-failed class); `result` key
  unwrapped.
- Pagination helpers (page size 100): old v2 API GET paging
  (`currentPage`/`currentPageSize`, `totalRows`), OpenAPI v1/v2 POST-body
  paging (`page`/`pageSize`), OpenAPI GET-query paging.

### Controller client (`controller.Client`)

- `Login(ctx) (string, error)` — returns controller ID.
- `ControllerInfo(ctx)` → `/api/info` (version, apiVer, configured, type,
  supportApp, omadacId, registeredRoot, omadacCategory, mspMode,
  omadaCloudUrl).
- `ControllerVersion(ctx)`.
- `ControllerName(ctx)` → `maintenance/uiInterface` (`controllerName`).
- `ControllerType(ctx)` → `anon/controllerType` (isSoftController,
  combinedGateway).
- `ControllerStatus(ctx)` → `maintenance/controllerStatus` (name, mac,
  upTime, controllerVersion, model with "Controller" fallback).
- `Sites(ctx) []Site` → `users/current` → `privilege.sites` (name, id/key).
- `SiteClient(ctx, nameOrSite)` — resolve name → id (else `ErrSiteNotFound`).
- `Reboot(ctx) (int seconds, error)` → `cmd/reboot` (`rebootTime`).
- `SetCertificate(ctx, pfxPath, keyPassword)` — multipart upload to
  `files/controller/certificate` (file + `data` JSON with cerName), then
  PATCH `controller/setting` with `{certificate: {cerId, cerName, cerType:
  "PFX", enable: true, keyPassword}}`.
- `CheckFirmwareUpdates(ctx, channel *int)`:
  - version < 6.3: channel selection is an error; GET
    `controller/notification/updateInfo`.
  - >= 6.3: GET `maintenance/software/channelUpdate`; pick
    `currentChannel` (or requested channel; missing → error); hardware
    controllers (not soft) additionally GET
    `maintenance/hardware/checkFirmware` for the current firmware version;
    normalized result has `hardware` and/or `software` sections with
    upgrade/currentVersion/latestVersion/releaseNotes/releaseUrl/
    downloadLink/denyUpgrade.
- `InstallControllerFirmware(ctx, targetVersion)` → `cmd/upgradeFirmware`.
- `GetControllerUpgradeStatus(ctx)` → `maintenance/hardware/upgradeStatus`
  (upgradeStatus, upgradeMsg, upgradeTime, downloadProgress, rebootTime).

### Site client (`controller.SiteClient`)

- **Clients**: `BlockClient`/`UnblockClient`/`ReconnectClient` (accept MAC
  string or client object; `cmd/clients/{mac}/block|unblock|reconnect`),
  `GetClient` (wireless vs wired detail types), `UpdateClient`
  (name, lockToAps, fixed IP address; empty settings = read-only),
  `ConnectedClients` (version >= 6.2.0.0 → OpenAPI v2 POST
  `clients` with `filters.active=true, hideHealthUnsupported, scope=1`;
  else old v2 GET `clients` with `filters.active=false` — note upstream
  quirk), `KnownClients` (`insight/clients`).
- **Devices**: `Devices() []Device`, `Device(mac)`, `Topology()`
  (gateway-rooted tree with successors, uplink info, WAN port summaries).
- **Access points**: `AccessPoints()`, `AccessPoint(mac|device)` →
  `eaps/{mac}` (lan port settings, radio settings/status per band
  2g/5g/5g2/6g, capabilities, wired uplink); `AccessPointPort(...)`;
  `UpdateAccessPointPort(...)` (PATCH `eaps/{mac}` `lanPortSettings`);
  `AccessPointChannels(mac)` → `eaps/{mac}/channelInfo`;
  `SetAccessPointRadioSettings(mac, band, settings)` — channel 0 = auto
  (clears freq), else map channel number → controller index+freq via
  channelInfo; PUT `eaps/{mac}/config/radios` with
  `{channelLimitType: 0, <bandKey>: radioPayload}`; read back after update.
- **Switches**: `Switches()`, `Switch(mac)`, `SwitchPorts(mac)`,
  `SwitchPort(mac, index|port)`; `SwitchPortOverrides(mac, port)` (port
  overrides if enabled, else merged from profile); `UpdateSwitchPort`
  (payload built from current port + settings: name, profileId, linkSpeed,
  duplex, tagIds, nativeNetworkId, networkTagsSetting +
  tagged/untagged ids, voice network, optional profile overrides incl.
  PoE/802.1x/LLDP/loopback/STP/isolation, version-gated EEE/flow control/
  VLAN-based loopback for 6.0+; old v2 URL below 6, OpenAPI v1 URL on 6+);
  `PortProfile(id)` / `PortProfiles()` (>= 6.2.0.0 → OpenAPI v2
  `lan-profiles` GET paging; else `setting/lan/profileSummary`).
- **Gateways**: `Gateways()`, `Gateway(mac|nil = first gateway)`,
  `GatewayPort(portId, ...)`, `SetGatewayWanPortConnectState(portId,
  connect, [ipv6])` → `cmd/gateways/{mac}/internetState|ipv6State`;
  `SetGatewayPortSettings` (PoE only; rejects PoE on non-PoE gateways;
  PATCH gateway with full `poeSettings` list mirroring the UI).
- **Misc**: `SetLedSetting(mac|device, setting)` → PATCH device resource
  path; `FirmwareDetails(mac)` → `devices/{mac}/firmware`;
  `StartFirmwareUpgrade(mac)` → `cmd/devices/{mac}/onlineUpgrade`.
- **LAN networks / VLANs** (conservative VLAN-only contract):
  `LanNetworks()` (paged `setting/lan/networks`);
  `CreateVlan(VlanCreate{name, vlanId})` — validate (name 1–128 UTF-8 bytes,
  printable, no protected names; vlan 2–4090, not 1; conflicts on id/name
  vs fresh inventory), payload `application=0, purpose=vlan, vlans=<id as
  string>, ...guards disabled`, POST `setting/lan/vlans`, verify by
  re-list + payload match;
  `UpdateVlan(target, VlanUpdate{name?, vlanId?}, byID)` — preserve
  writable fields, reject unknown controller fields, no-op detection, PATCH
  `setting/lan/networks/{id}`, verify;
  `DeleteVlan(target, byID)` — target must be non-primary VLAN-only
  application=0, DELETE, verify absence;
  read-back failures produce guidance: "write may have succeeded; inspect
  controller state before retrying"; no auto-retry of writes.
- **DHCP reservations**: `DhcpReservations()` (>= 6.2.0.0 OpenAPI v1 GET
  paging `setting/service/dhcp`; else old paging), `CreateDhcpReservation
  (mac, ip, netId, [description])`, `UpdateDhcpReservation(mac, [ip],
  [description], [enabled])` (PATCH `/.../dhcp/{mac-lower}`),
  `DeleteDhcpReservation(mac)`.
- **VPN**: `VpnPolicies()` (OpenAPI v2 list per category: server →
  `client-to-site-vpn-servers`, client → `client-to-site-vpn-clients`,
  site-to-site → `site-to-site-vpns`; categories without support are
  skipped silently), `SetVpnPolicyEnabled(id, enabled)` (PATCH OpenAPI v1
  `vpn/{id}/status`).

### Data models & enums

Port from `definitions.py`, `devices.py`, `clients.py`, `networks.py`,
`topology.py`, `vlans.py`, `vpn.py`. Enums (int-based, unknown → Unknown,
mirroring Python `_missing_`): DeviceStatus(+Category), PortType,
GatewayPortType/Mode, LinkStatus/Speed/Duplex, Eth802Dot1X,
NetworkTagsSetting, BandwidthControl, PoEMode, ConnectType, RadioId (with
settings/status key maps), WifiMode, LedSetting, ChannelWidth,
AuthenticationStatus, VpnType, VpnCategory (string enum).

Models: ControllerInfo/Type/Status, UpdateInfo (hardware/software/
normalized), HardwareUpgradeStatus, Device/ListDevice/DetailedDevice,
Switch(+Port/PortDetails/PortProfile/Caps/Uplink/Downlink),
AccessPoint(+LanPort/RadioSettings/RadioStatus/Channel/RadioChannels),
Gateway(+PortStatus/PortConfig), Link/Uplink/Downlink, NetworkClient
(Connected/Wired/WiredDetails/Wireless/WirelessDetails, IpSetting,
RateLimit), LanNetwork, DhcpReservation, Topology/TopologyNode/
TopologyWanPort, VpnPolicy, VlanCreate/VlanUpdate request types.

## Errors

`*RequestError{Code int, Msg string}` (wraps, `errors.As`-able); sentinel /
typed errors: `ErrLoginFailed`, `ErrLoginSessionClosed`,
`ErrUnsupportedControllerVersion`, `ErrSiteNotFound`,
`ErrConnectionFailed`, `ErrBadControllerURL`, `ErrInvalidDevice`.

## Tests

- Standard library `testing` + `httptest`; table-driven.
- Port the behaviors covered by the Python test suite (auth/login flows,
  re-login on -30109, non-JSON 200, version-gated endpoint selection,
  pagination across page sizes, switch port payload construction, AP radio
  channel mapping, gateway PoE payload, VLAN validation/conflicts/read-back,
  DHCP CRUD payloads, controller firmware channel logic).
- Fixture JSON in `controller/testdata/` (write fresh fixtures shaped like
  the API; the Python repo has only one fixture file).
- `go vet` + `gofmt` clean; `go test ./...` green is the completion bar.

## Non-goals (phase 1)

- Cloud/tplinkcloud.com northbound API (phase 2; module layout already
  accommodates a `cloud/` package; spec reference:
  `https://use1-omada-northbound.tplinkcloud.com/v3/api-docs` — public,
  1868 paths / ~2450 ops under `/openapi/v1|v2|v3/{omadacId}/...`,
  OAuth2 client-credentials auth).
- CLI, websocket event stream, features beyond the Python library.

## Open / next steps

1. Builder review of this document (fresh subagent) — done 2026-10-29.
2. User approval of this document — done 2026-10-29.
3. Next: implementation plan (superpowers:writing-plans), then build.
