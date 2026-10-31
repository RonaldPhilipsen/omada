# omada

Go client for the TP-Link Omada SDN controller API. This package is a
synchronous port of [`tplink-omada-client`](https://github.com/tplink-omada-client)
v1.5.14 covering the controller scope.

## What it is

The client speaks the Omada controller REST API (`/openapi/...` and legacy
`/api/...` namespaces) over HTTP with a permissive cookie jar and automatic
re-login on session expiry.

## Supported

Omada Software Controller **5.1.0** and later.

Method families:

- sites (controller, site, site-by-name)
- clients (list, get, block, unblock, reconnect, update settings, known)
- devices (list, get, topology, LED, reboot, firmware, reset)
- access points (list, get, ports, radios, channels)
- switches and port profiles
- gateways and PoE
- LAN networks, VLANs and DHCP reservations
- VPN policies (server / client / site-to-site) and status updates
- firmware updates and controller updates

## Quick start

```go
client, err := controller.New("https://omada.example.com", "user", "pass")
if err != nil { ... }

sites, err := client.Sites(ctx)
if err != nil { ... }

siteClient, err := client.SiteClient(ctx, sites[0].ID())
if err != nil { ... }

// Create a VLAN-only network.
network, err := siteClient.CreateVLAN(ctx, "office", 10)
if err != nil { ... }

// Block a client by MAC.
if err := siteClient.BlockClient(ctx, "AA:BB:CC:DD:EE:FF"); err != nil { ... }
```

## Concurrency

`Client` and `SiteClient` are safe for concurrent use. Authentication
transitions are serialized.

## Errors

- `*RequestError` — the controller answered with a non-zero error code; the
  controller code is available via `Code` and `errors.Is` matches
  `ErrLoginFailed` for code `-30109`.
- `*ValidationError` — invalid caller input.

Sentinels: `ErrLoginFailed`, `ErrLoginSessionClosed`, `ErrUnsupportedControllerVersion`,
`ErrSiteNotFound`, `ErrConnectionFailed`, `ErrBadControllerURL`, `ErrInvalidDevice`.

## Divergences from the Python library

- Three defect fixes: bounded pagination, `VpnPolicies` all-pages, re-login
  scope, plus the VLAN read-back contract message.
- Structural notes: a single `Device` struct, a flat `Client` struct, and the
  `ValueError` family folded into `*ValidationError`.

## Phase 2

The northbound cloud API (`tplinkcloud.com`) is intentionally out of scope.
