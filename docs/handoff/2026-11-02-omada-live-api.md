# Handoff: Omada cloud client — live read-only API test + CI

Date: 2026-11-02
Branch: `omada-cloud` (worktree `.worktrees/omada-cloud`, based on `main` @ `3539c74`)
Status: **UNBLOCKED and green.** The token endpoint works on the LAN address
`https://192.168.0.2:443` and the public host `https://omada.rphilipsen.nl:443`.
The full live read-only suite passes on both: `total=1045 pass=487 biz=160
skip=398 fail=0`. A client bug (context cancelled before the response body was
read) and a rate-limit burst were fixed; CI env secrets are set. See the
"Resolution (2026-11-02)" section at the bottom.

## The task (from the user)

1. Test locally that **all read-only (GET) endpoints** of the Omada northbound
   API work against the user's real controller, and make them work.
2. Create a **CI job** that runs against the Omada API, using the GitHub
   **Test** environment secrets `OMADA_TEST_CLIENT_ID` /
   `OMADA_TEST_CLIENT_SECRET` (a read-only Open API application).

Hard constraint from the user: **only query/expose data that would be okay to
leak** — no IP addresses, locations, MACs, hostnames, or any response content
in logs. The harness is built log-safe by construction (see invariants below);
keep it that way.

## Credentials / endpoints (do NOT write these into committed files)

- Client ID / secret: in the 1Password item the user created (search
  `omada`), mirrored in the GitHub Test environment as
  `OMADA_TEST_CLIENT_ID` / `OMADA_TEST_CLIENT_SECRET`.
- OmadacID (org id): `803d74060572a8ec2ecebe22591f1a89`
  (confirmed by the controller's `GET /api/info`).
- Controller (local UI): `https://omada.rphilipsen.nl/#/dashboard` (port 443;
  the API also answers on `:8043`). Serves the **stock self-signed cert**
  (`CN=localhost`, no SAN, issuer `TP-LINK CA P1`) — hostname verification
  fails for every modern client.
- **Cloud endpoint (the real Open API home, per the user):**
  `https://euw1-omada-cloud.tplinkcloud.com/` — this is the "Interface
  Access Address" shown on the controller's
  Settings → Platform Integration → Open API page.

## What is already built (commit `0103984`, pushed to `origin/omada-cloud`)

| File | What it is |
|---|---|
| `cloud/livetest/livetest.go` | Live harness (build tag `live`): builds the `cloud.Client`, runs token preflight, discovers site → devices (AP/switch/gateway by `type`) → LAN/SSID/VPN/client/MSP-customer best-effort, walks the table, classifies PASS / BIZ (200 + non-zero `errorCode`) / SKIP (undiscoverable path param) / FAIL (transport, 401/403/404/405/5xx). Fails the run on FAIL only. |
| `cloud/livetest/live_table.gen.go` | Generated table of **all 1,045 GET operations**, each a closure calling the real generated method with page params (`page=1&pageSize=10`) where the params struct has them. |
| `cloud/livetest/livetest_test.go` | `TestLiveReadOnly` — skips fast when `OMADA_BASE_URL` / `OMADA_OMADAC_ID` / `OMADA_CLIENT_ID` / `OMADA_CLIENT_SECRET` are unset. |
| `gen/gen_livetest.py` + `gen/Makefile` (`livetest` target) | Regenerates the table from the checked-in `cloud/*/*.gen.go` (no network needed). Run `make -C gen livetest` after spec regeneration. |
| `cloud/cloud.go` | `Options.InsecureTLS` (opt-out of cert verification for API **and** token requests) and `Client.PreflightToken(ctx)` (surfaces token-endpoint failures as one line). |
| `cloud/cloud_test.go` | `TestInsecureTLSOption` unit test (hostname-mismatched self-signed server). |
| `.github/workflows/live-api.yml` | Job `read-only`: `environment: Test`, nightly cron `30 6 * * *` + `workflow_dispatch`, runs `go test -tags live ./cloud/livetest -v -count=1` with `OMADA_TEST_*` secrets mapped to `OMADA_*` env. Currently hardcodes `OMADA_INSECURE_TLS: "true"` with a comment to remove it once a proper cert is served. |
| `.pre-commit-config.yaml` | Local hook `omada-live-read-only` (`always_run: true`, `pass_filenames: false`) running the same test; skips fast without the env. pre-commit is installed via `uv tool install pre-commit`; hook installed at `/home/ronald245/Documents/Source/omada/.git/hooks/pre-commit` (shared across worktrees). |
| `README.md` | "Live read-only test" section documenting env vars, log safety, CI, pre-commit, regeneration. |

Gate status at commit time: `gofmt -l .` clean, `go vet ./...` and
`go vet -tags live ./...` clean, `go test ./... -race -count=1` all green.

Local run command (once the token endpoint is reachable):

```bash
cd .worktrees/omada-cloud
OMADA_BASE_URL=<endpoint> \
OMADA_OMADAC_ID=803d74060572a8ec2ecebe22591f1a89 \
OMADA_CLIENT_ID=<from 1Password> \
OMADA_CLIENT_SECRET=<from 1Password> \
[OMADA_INSECURE_TLS=true] \
go test -tags live ./cloud/livetest -v -count=1
```

## What is blocked, and the evidence

The controller is a **cloud-based controller (CBC)**: `GET
https://omada.rphilipsen.nl:8043/api/info` →
`controllerVer 6.3.0.111`, `registeredRoot: true`,
`omadaCloudUrl: https://omada.tplinkcloud.com`, `type: 17`,
`mspMode: false`. (6.3.0.111, Oct 2026, is the newest 6.3 build.)

Probes (all from this machine, 2026-11-01/02):

1. **Local controller** (`omada.rphilipsen.nl:8043`, `-k`):
   - `GET /api/info` → 200, correct omadacId.
   - `GET /openapi/v1/{omadacId}/sites` unauthenticated → clean 401 (API
     surface exists, auth enforced).
   - `POST /openapi/authorize/token?grant_type=client_credentials` with the
     documented JSON body (`omadacId`, `client_id`, `client_secret`) →
     **HTTP 500** (Tomcat error page), for **any** body — including
     deliberately bogus credentials. Without `grant_type` the same endpoint
     returns a clean envelope 401 (`errorCode -44116`), so the endpoint is
     alive and only the grant path crashes.
   - `POST /openapi/authorize/login` (auth-code flow entry) → also 500.
   - Tried: form-urlencoded, params duplicated in query+body (known Omada
     quirk), `grant_type` in the JSON body, charset-suffixed content type,
     Accept/User-Agent headers — all 500.
   - User checked the controller logs: **no error entries**.
   - Interpretation: on a CBC the local Open API surface is not the
     supported client-credentials home; the grant path is dead locally.
2. **Cloud endpoint** (`euw1-omada-cloud.tplinkcloud.com`, valid public
   cert, no `-k` needed):
   - `POST /openapi/authorize/token?grant_type=client_credentials` with the
     same correct body → **HTTP 403** with a TP-Link-branded "Error!" HTML
   page (not a JSON envelope). This is the current wall.

## Next steps (in order)

1. **Find the exact working token URL on the cloud endpoint.**
   The 403 HTML page suggests WAF/path rejection, not an auth answer.
   - Ask the user for the **verbatim "Interface Access Address"** string
     from the controller's Settings → Platform Integration → Open API page
     (they reported `https://euw1-omada-cloud.tplinkcloud.com/` — confirm
     whether it includes a path, and check the region: the public hosted
     spec lives on `use1-omada-northbound.tplinkcloud.com`; the EU
     equivalent host pattern would be
     `euw1-omada-northbound.tplinkcloud.com` — probe that host's
     `/openapi/authorize/token` too).
   - Probe variants: `Accept: application/json` header; GET the base URL to
     see what the host serves; check whether the 403 page differs per
     path (i.e. is it path-specific or host-wide).
2. **Once a token is obtained** (`{"errorCode":0, ... "result":
   {"accessToken": ...}}`), run the live test locally with
   `OMADA_BASE_URL` set to the working endpoint (drop
   `OMADA_INSECURE_TLS` if it has a proper cert). Expect a mix of PASS /
   BIZ / SKIP; **investigate every FAIL** (that is the "make them work"
   part) and fix client bugs (paths, params, envelope handling) in
   `cloud/` — never by weakening the log-safety invariant.
   BIZ lines (200 + non-zero errorCode) are expected for undeployed
   features on a read-only org; review them, don't chase them.
3. **Update the CI job**: put the working endpoint in the GitHub Test
   environment as `OMADA_TEST_BASE_URL` (the user must add it, plus
   `OMADA_TEST_OMADAC_ID` — only the two `OMADA_TEST_CLIENT_*` secrets
   exist there today). Remove `OMADA_INSECURE_TLS: "true"` from
   `live-api.yml` if the endpoint has a valid cert. Trigger
   `workflow_dispatch` once to verify green.
4. **Open the PR** `omada-cloud` → `main` (repo flow is branch → PR; see
   `release.yaml` / `spec-regen.yml` for style).

## Invariants (do not break)

- **Log safety:** the live test and its CI logs must never contain response
  bodies, resource IDs, MACs, IPs, hostnames or any controller data — only
  operation names, HTTP status codes, envelope `errorCode`/`msg`. This is
  why `transportDetail()` suppresses URLs and `classifyResponse()` never
  echoes bodies. New code in `cloud/livetest` must preserve this.
- **Read-only:** the application credentials are read-only; the test only
  calls GET operations (the table is generated from GETs only). Do not add
  write calls.
- **CI trigger choice:** nightly + `workflow_dispatch` + PRs that touch the
  live test itself (`cloud/livetest/**` or `live-api.yml`), all on the
  self-hosted `omada-runner` (user-approved 2026-11-02). Fork PRs can't
  read environment secrets, so the test skips fast there.
- **Repo conventions:** pin action SHAs in workflows with `# vX` comments;
  generated `*.gen.go` files are checked in; the gate is
  `gofmt -l .` + `go vet ./...` + `go test ./... -race -count=1`;
  `gen/spec.sha256` fingerprints the upstream spec (the `spec-regen.yml`
  workflow handles drift).
- **Never commit secrets.** This handoff deliberately names no secret
  values.

## Environment notes

- The dev machine had a full network outage mid-session (no IPv4 address,
  IPv6 only on the RFC 3849 documentation prefix `2001:db8::/32`); it
  recovered during the session. If DNS/network fails again, nothing live
  works — check `ip -4 addr` and `getent hosts github.com` first.
- `op` CLI: the 1Password app runs, but delegated sessions failed during the
  outage (`RequestDelegatedSession` errors); retry after network is up.
  `op item list` has no `--all` flag on this build (2.41.1).
- Go module cache already has `oapi-codegen/runtime@v1.8.0`; offline builds
  work.
- `pre-commit` was installed with `uv tool install pre-commit` (pipx is
  broken on this machine: `No module named 'pipx'`).
- Untracked pre-existing files in the main worktree (not part of this work):
  `docs/superpowers/plans/2026-11-01-omada-controller-scope-narrowing.md`,
  `docs/superpowers/specs/2026-11-01-omada-controller-scope-narrowing-design.md`.
  Leave them alone.
- Other worktrees exist: `~/.local/share/opencode/worktree/544db6/omada-controller-scope`
  (branch `omada-controller-scope`).

## Key references

- Upstream (hosted) OpenAPI spec: `https://use1-omada-northbound.tplinkcloud.com/v3/api-docs`
  (fetched copy in `/tmp/opencode/api-docs.json` — transient). It documents
  the 3,452 public operations but **not** the token endpoint.
- Token contract (org-level client credentials): `POST
  /openapi/authorize/token?grant_type=client_credentials` with JSON
  `{"omadacId", "client_id", "client_secret"}` →
  `{"errorCode":0,"result":{"accessToken","expiresIn",...}}`;
  `Authorization: AccessToken=<token>` on every call. Confirmed working on
  on-prem 6.2 by the terraform-provider-omada project and community reports;
  the 500/403 behavior here is CBC-specific.
- Community context: TP-Link stated (Sep 2026) that Open API access for
  cloud-based controllers "is not supported ... already on our roadmap" —
  the user's controller evidently now exposes it via the regional cloud
  endpoint, which is exactly why pinning the right URL matters.

## Resolution (2026-11-02)

The block was resolved without any server-side change. Summary of what
happened and what changed in the branch:

1. **The token endpoint works on the controller's own address.** The user
   supplied the verbatim "Interface Access Address"
   (`https://192.168.0.2:443`) and Omada ID
   (`803d74060572a8ec2ecebe22591f1a89`).
   `POST /openapi/authorize/token?grant_type=client_credentials` returns
   `errorCode 0` + `accessToken` there — and also on the public host
   `https://omada.rphilipsen.nl:443`. The `500`/`403` seen earlier was the
   `:8043` port and the cloud host, not the `:443` addresses. No client
   change was needed for the token flow.

2. **Fixed a client bug: "read response body" failures.** 4 of 1045 GETs
   (`client.GetClientActivity`, `security.GetGridArpDetects`,
   `security.GetGridDhcpSnoops`, `security.GetRules`) failed because
   `Run`/`discover` cancelled the per-call `context` **before** the response
   body was read. Cancelling a request context tears down the transport
   connection, which aborts the read of an in-flight (larger) body with
   `context canceled`. Fix in `cloud/livetest/livetest.go`: drain and close
   the body first, then cancel; `discover` now uses a `callAndRead` helper
   that fully reads the body inside the call. `classifyResponse` also
   surfaces the real read error (log-safe).

3. **Added `OMADA_LIVE_DELAY` throttling.** The controller rate-limits fast
   bursts (envelope `errorCode -7132`); an unthrottled run produced ~136
   `-7132` BIZ lines. A per-op delay (CI sets `100ms`) keeps the run under
   the limit so the report reflects real endpoint behavior. New optional
   `Config.PerOpDelay`, read from `OMADA_LIVE_DELAY`.

4. **CI wired up.** The GitHub `Test` environment now defines
   `OMADA_TEST_BASE_URL`, `OMADA_TEST_CLOUD_URL`, `OMADA_TEST_OMADAC_ID`,
   plus the pre-existing `OMADA_TEST_CLIENT_ID` /
   `OMADA_TEST_CLIENT_SECRET`. `live-api.yml` sets `OMADA_LIVE_DELAY: 100ms`
   and keeps `OMADA_INSECURE_TLS: "true"` (stock self-signed cert on both
   addresses).
5. **Self-hosted runner + dual-endpoint run (user request, 2026-11-02).**
   The job runs on the self-hosted `omada-runner` (on the controller's
   LAN) and probes the controller sequentially through both access paths:
   on-premises (`OMADA_TEST_BASE_URL` = `https://192.168.0.2:443`, the
   Interface Access Address) then cloud/public
   (`OMADA_TEST_CLOUD_URL` = `https://omada.rphilipsen.nl:443`); the
   second run only starts if the first passes. The workflow also triggers
   on PRs touching `cloud/livetest/**` or the workflow file. Note: the
   TP-Link `tplinkcloud.com` cloud hosts still 403 the token endpoint —
   both addresses above are the controller's own.

Result (both LAN and public host, with `OMADA_LIVE_DELAY=100ms`):

```
live read-only: total=1045 pass=487 biz=160 skip=398 fail=0
```

The 160 BIZ lines are legitimate business errors (undeployed features /
"no permission" on a read-only org) and are expected. The 398 SKIPs are
endpoints whose path parameters the org has no resource to supply.
