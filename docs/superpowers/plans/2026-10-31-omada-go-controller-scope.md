# Controller OpenAPI scope completion — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. Execution method: native (this session), per user's "continue".

**Goal:** Bring the Go controller client to parity with the upstream `tplink-omada-client` v1.5.14 controller scope by adding the remaining thin OpenAPI wrappers: `GetSwitchPort`, `GetFirmwareDetails`, `GetGatewayPort`, `SetGatewayWanPortConnectState`, `Controller.InstallControllerFirmware`, `Controller.GetControllerUpgradeStatus` — each with TDD tests.

**Architecture:** Each method is a thin wrapper over `conn.requestEndpoint` (v2 API namespace) following existing file conventions; empty required inputs return `*ValidationError`; errors propagate as `*RequestError`. `GetGatewayPort` composites `Gateway()` + `Gateway.PortConfigs()`. Tests use `newTestClient`/`writeJSON` in `controller/*_test.go`.

**Tech Stack:** Go stdlib only (module github.com/RonaldPhilipsen/omada, package controller), table-driven `httptest` tests per existing conventions.

**Spec:** Upstream reference `/tmp/opencode/tplink-omada-api` (tag release/v1.5.14), `src/tplink_omada_client/omadasiteclient.py` methods `get_switch_port`, `get_firmware_details`, `get_gateway_port`, `set_gateway_wan_port_connect_state`, `install_controller_firmware`, `get_controller_upgrade_status`; repo design `docs/superpowers/specs/2026-10-29-omada-go-client-design.md`.

## Global Constraints

- Stdlib only; no third-party dependencies.
- Every method takes `ctx context.Context` first.
- Empty required string input → `&ValidationError{Msg: "..."}`.
- Controller-level endpoints use site `""`; site-scoped endpoints use `s.site`.
- URL shapes: site-scoped `/{id}/api/v2/sites/site-1/{endpoint}`; controller-level `/{id}/api/v2/{endpoint}`.
- Tests: `newTestClient(t, handler)` + `writeJSON(t, w, v)`; assertions on exact URL, method, body, and return values.
- Gate: `gofmt -l .` empty, `go vet ./...` clean, `go test ./... -race -count=1` green.

## Review Focus

1. Empty device ID → ValidationError before any HTTP call (owned by each site-scoped task).
2. Port index 0 / nonexistent port → error (GetSwitchPort miss task; GetGatewayPort miss task).
3. Controller-level vs site-scoped URL shape (InstallControllerFirmware/GetControllerUpgradeStatus tasks).
4. Request body contents for SetGatewayWanPortConnectState (`{"portId": n, "operation": 1|0}`) and InstallControllerFirmware (`{"targetVersion": v}`) (owned by those tasks).

---

### Task 1: SiteClient.GetSwitchPort

**Files:**
- Modify: `controller/site_switch.go`
- Test: `controller/site_switch_test.go`

**Interfaces:**
- Consumes: `s.c.requestEndpoint(ctx, "GET", "v2", "switches/"+deviceID+"/ports/"+strconv.Itoa(portIndex), s.site, nil, nil)`; `NewSwitchPortDetails(m)`.
- Produces: `func (s *SiteClient) GetSwitchPort(ctx context.Context, deviceID string, portIndex int) (*SwitchPortDetails, error)`; added to `TestGetSwitchPort`, `TestGetSwitchPortEmptyID`, `TestGetSwitchPortMiss`.

- [ ] **Step 1: Write the failing test** `TestGetSwitchPort` (handler: GET `/{id}/api/v2/sites/site-1/switches/AA:BB:CC:DD:EE:03/ports/3` → result port map; assert `Index()==3`, `PoEMode()`, no error), `TestGetSwitchPortEmptyID` (assert `*ValidationError`), `TestGetSwitchPortMiss` (handler returns empty result → assert error).
- [ ] **Step 2: Run test to verify it fails** → `go test ./controller/ -run 'TestGetSwitchPort' -count=1 -v` → compile error "undefined: GetSwitchPort".
- [ ] **Step 3: Implement `GetSwitchPort`** in `controller/site_switch.go` after `SwitchPorts`; empty deviceID → `&ValidationError{Msg: "device ID must not be empty"}`; `len(m)==0` → `fmt.Errorf("%w: port %d not found", ErrInvalidDevice, portIndex)`.
- [ ] **Step 4: Run test to verify it passes** → `go test ./controller/ -run 'TestGetSwitchPort' -count=1 -v` → PASS.
- [ ] **Step 5: Commit** → `git add controller/site_switch.go controller/site_switch_test.go && git commit -m "feat(controller): add GetSwitchPort"`.

### Task 2: SiteClient.GetFirmwareDetails

**Files:**
- Modify: `controller/site_device.go`
- Test: `controller/site_device_test.go`

**Interfaces:**
- Consumes: `s.c.requestEndpoint(ctx, "GET", "v2", "devices/"+deviceID+"/firmware", s.site, nil, nil)`; `NewFirmwareUpdate(m)`.
- Produces: `func (s *SiteClient) GetFirmwareDetails(ctx context.Context, deviceID string) (*FirmwareUpdate, error)`; added to `TestGetFirmwareDetails`, `TestGetFirmwareDetailsEmptyID`.

- [ ] **Step 1: Write the failing test** → handler GET `/{id}/api/v2/sites/site-1/devices/AA:BB:CC:DD:EE:03/firmware` → result firmware map; assert fields populated, no error; empty ID → `*ValidationError`.
- [ ] **Step 2: Run test to verify it fails** → undefined: GetFirmwareDetails.
- [ ] **Step 3: Implement `GetFirmwareDetails`** in `controller/site_device.go` after `GetFirmwareUpdates`; empty deviceID → `&ValidationError{Msg: "device ID must not be empty"}`.
- [ ] **Step 4: Run test to verify it passes** → PASS.
- [ ] **Step 5: Commit** → `git add controller/site_device.go controller/site_device_test.go && git commit -m "feat(controller): add GetFirmwareDetails"`.

### Task 3: SiteClient.GetGatewayPort

**Files:**
- Modify: `controller/site_gateway.go`
- Test: `controller/site_gateway_test.go`

**Interfaces:**
- Consumes: `s.Gateway(ctx, deviceID)`; `Gateway.PortConfigs()`; `GatewayPortConfig.PortNumber()`.
- Produces: `func (s *SiteClient) GetGatewayPort(ctx context.Context, deviceID string, portIndex int) (*GatewayPortConfig, error)`; added to `TestGetGatewayPort`, `TestGetGatewayPortEmptyID`, `TestGetGatewayPortMiss`.

- [ ] **Step 1: Write the failing test** → gateway handler GET `/{id}/api/v2/sites/site-1/gateways/AA:BB:CC:DD:EE:03` → gateway with `port_configs` list incl. port 1; assert `PortNumber()==1`, `Name()`; port 99 → `*ValidationError` "gateway port not found"; empty ID → `*ValidationError`.
- [ ] **Step 2: Run test to verify it fails** → undefined: GetGatewayPort.
- [ ] **Step 3: Implement `GetGatewayPort`** in `controller/site_gateway.go` after `Gateway`; empty deviceID → ValidationError; gateway lookup error → propagate; loop `gw.PortConfigs()` match `PortNumber()==portIndex`; none → `&ValidationError{Msg: "gateway port not found"}`; return pointer to found config (`NewGatewayPortConfig(cfg)`).
- [ ] **Step 4: Run test to verify it passes** → PASS.
- [ ] **Step 5: Commit** → `git add controller/site_gateway.go controller/site_gateway_test.go && git commit -m "feat(controller): add GetGatewayPort"`.

### Task 4: SiteClient.SetGatewayWanPortConnectState

**Files:**
- Modify: `controller/site_gateway.go`
- Test: `controller/site_gateway_test.go`

**Interfaces:**
- Consumes: `s.c.requestEndpoint(ctx, "POST", "v2", "gateways/"+deviceID+"/internetState", s.site, nil, body)` where `body = map[string]any{"portId": portIndex, "operation": 1 if connect else 0}`; `NewGatewayPortStatus(m)`.
- Produces: `func (s *SiteClient) SetGatewayWanPortConnectState(ctx context.Context, deviceID string, portIndex int, connect bool) (*GatewayPortStatus, error)`; added to `TestSetGatewayWanPortConnectState`, `TestSetGatewayWanPortConnectStateEmptyID`, plus body assertion.

- [ ] **Step 1: Write the failing test** → handler POST `/{id}/api/v2/sites/site-1/gateways/AA:BB:CC:DD:EE:03/internetState`; capture body via `json.NewDecoder(r.Body).Decode(&body)`; assert `body["portId"]==1`, `body["operation"]==1` (connect true) / `0` (connect false second case); result → `InternetState()==true`; empty ID → `*ValidationError`.
- [ ] **Step 2: Run test to verify it fails** → undefined: SetGatewayWanPortConnectState.
- [ ] **Step 3: Implement `SetGatewayWanPortConnectState`** in `controller/site_gateway.go` after `GetGatewayPort`; empty deviceID → ValidationError; operation int mapping; return `NewGatewayPortStatus(m)`.
- [ ] **Step 4: Run test to verify it passes** → PASS.
- [ ] **Step 5: Commit** → `git add controller/site_gateway.go controller/site_gateway_test.go && git commit -m "feat(controller): add SetGatewayWanPortConnectState"`.

### Task 5: Controller.InstallControllerFirmware + GetControllerUpgradeStatus

**Files:**
- Modify: `controller/controller.go`
- Test: `controller/controller_test.go`

**Interfaces:**
- Consumes: `cl.c.requestEndpoint(ctx, "POST", "v2", "cmd/upgradeFirmware", "", nil, body)` body `{"targetVersion": targetVersion}`; `cl.c.requestEndpoint(ctx, "GET", "v2", "maintenance/hardware/upgradeStatus", "", nil, nil)`; `NewHardwareUpgradeStatus(m)`.
- Produces: `func (cl *Controller) InstallControllerFirmware(ctx context.Context, targetVersion string) error`; `func (cl *Controller) GetControllerUpgradeStatus(ctx context.Context) (*HardwareUpgradeStatus, error)`; added to `TestInstallControllerFirmware`, `TestGetControllerUpgradeStatus`, empty-version validation.

- [ ] **Step 1: Write the failing test** → controller-level handlers POST `/{id}/api/v2/cmd/upgradeFirmware` (assert body targetVersion) and GET `/{id}/api/v2/maintenance/hardware/upgradeStatus` → `UpgradeStatus()`; empty targetVersion → `*ValidationError`.
- [ ] **Step 2: Run test to verify it fails** → undefined: InstallControllerFirmware / GetControllerUpgradeStatus.
- [ ] **Step 3: Implement both** in `controller/controller.go` near `Reboot`/`Shutdown`; empty targetVersion → `&ValidationError{Msg: "target version must not be empty"}`.
- [ ] **Step 4: Run test to verify it passes** → PASS.
- [ ] **Step 5: Commit** → `git add controller/controller.go controller/controller_test.go && git commit -m "feat(controller): add InstallControllerFirmware and GetControllerUpgradeStatus"`.

### Task 6: Full gate + commit Task 18 equivalent

**Files:** none new.

- [ ] **Step 1: Run full gate** → `gofmt -l .` (empty), `go vet ./...` (clean), `go test ./... -race -count=1` (green).
- [ ] **Step 2: Fix anything surfacing** (iterate until green).
- [ ] **Step 3: Final verify** → `go test ./... -race -count=1` green; `git log --oneline` shows one commit per feature.
- [ ] **Step 4: Done** → no further commit (working tree clean except untracked `mise.toml`).

## Self-Review

1. Spec coverage: all six remaining upstream controller wrappers mapped to tasks 1–5; none missing from the plan.
2. Step scan: each step writes exactly one test or one method body; no TBD lines.
3. Type consistency: `*SwitchPortDetails`, `*FirmwareUpdate`, `*GatewayPortConfig`, `*GatewayPortStatus`, `*HardwareUpgradeStatus` constructors already exist (`NewSwitchPortDetails`, `NewFirmwareUpdate`, `NewGatewayPortConfig`, `NewGatewayPortStatus`, `NewHardwareUpgradeStatus`).
4. Review focus: all four failure modes have owning tasks and tests.
5. Proportion: plan ~ fits the six-method scope; bodies deferred to signature+tests.

Plan complete and saved to `docs/superpowers/plans/2026-10-31-omada-go-controller-scope.md`. Execution method is native (this session). I'll start with Task 1.
