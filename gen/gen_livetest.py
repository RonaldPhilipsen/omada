#!/usr/bin/env python3
"""Generate cloud/livetest/live_table.gen.go from the checked-in generated
clients.

Scans every cloud/*/*.gen.go for GET operations (via the oapi-codegen doc
comment "Corresponds with GET <path> (the `<opId>` operationId).") and emits
a table of closures that call the real generated methods. The live test
harness (cloud/livetest, build tag `live`) walks the table, resolves path
parameters from a discovered resource scope, and reports per-endpoint status
without ever printing response bodies.

Usage: python3 gen/gen_livetest.py
"""
import re
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
CLOUD = REPO / "cloud"
OUT = CLOUD / "livetest" / "live_table.gen.go"

# package dir name -> (import name, scope field)
# Import names match the aliases used in cloud/cloud.go.
PKG = {
    "ap": ("ap", "ap"),
    "client": ("client", "clients"),
    "device": ("device", "devices"),
    "gateway": ("gateway", "gateways"),
    "insight": ("insight", "insight"),
    "lan": ("lan", "lans"),
    "log": ("omadalog", "omadalog"),
    "msp": ("msp", "msp"),
    "remote": ("remote", "remote"),
    "security": ("security", "security"),
    "sim": ("sim", "sims"),
    "site": ("site", "site"),
    "stack": ("stack", "stacks"),
    "switch": ("switchpkg", "switches"),
    "topology": ("topology", "topology"),
    "user": ("user", "users"),
    "voip": ("voip", "voip"),
    "vpn": ("vpn", "vpn"),
    "wireless": ("wireless", "wireless"),
}

# path parameter name -> generated call argument.
# Unknown parameters map to s.missing (never set), so operations that need
# them are skipped at runtime.
PARAM_ARG = {
    "omadacId": "s.omadacID",
    "siteId": "s.siteID",
    "mspId": "s.mspID",
    "customerId": "s.customerID",
    "deviceMac": "s.deviceMac",
    "apMac": "s.apID",
    "switchMac": "s.switchID",
    "gatewayMac": "s.gatewayID",
    "networkId": "s.networkID",
    "ssidId": "s.ssidID",
    "vpnId": "s.vpnID",
    "clientMac": "s.clientMac",
    "mac": "s.deviceMac",
    "vlan": '"1"',
    "deviceType": '"ap"',
    "format": '"json"',
    "cliType": '"ap"',
}

OP_RE = re.compile(
    r"Corresponds with (GET) ([^\s(]+) \(the `(\w+)` operationId\)\.\s*\n"
    r"func \(c \*Client\) (\w+)\(([^)]*)\) \(\*http\.Response, error\) \{"
)
PARAMS_STRUCT_RE = re.compile(r"^type (\w+Params) struct \{(.*?)\n\}", re.M | re.S)
PAGE_RE = re.compile(r"^\tPage (\*?)int32", re.M)
PAGESIZE_RE = re.compile(r"^\tPageSize (\*?)int32", re.M)


def params_init(pkg_import, params_struct, body):
    """Emit the params-struct argument for a call, or None when absent."""
    if params_struct is None:
        return None
    page = PAGE_RE.search(body) if body else None
    psize = PAGESIZE_RE.search(body) if body else None
    if page and psize:
        if page.group(1) == "*" and psize.group(1) == "*":
            return f"&{pkg_import}.{params_struct}{{Page: &livePage, PageSize: &livePageSize}}"
        if page.group(1) == "" and psize.group(1) == "":
            return f"&{pkg_import}.{params_struct}{{Page: 1, PageSize: 10}}"
        # Mixed pointer/value page fields are not seen in the current spec;
        # fall through to a zero value and let the server apply defaults.
    return f"&{pkg_import}.{params_struct}{{}}"


def main():
    entries = []
    for f in sorted(CLOUD.glob("*/*.gen.go")):
        pkg_dir = f.parent.name
        if pkg_dir == "livetest":
            continue
        pkg_import, scope_field = PKG[pkg_dir]
        src = f.read_text()
        structs = {m.group(1): m.group(2) for m in PARAMS_STRUCT_RE.finditer(src)}
        for m in OP_RE.finditer(src):
            _method, op_path, _opid, fname, sig = m.groups()
            sig_args = [a.strip() for a in sig.split(",")]
            params_struct = None
            for arg in sig_args:
                if arg.startswith("params "):
                    params_struct = arg.split()[1].lstrip("*")
            need = re.findall(r"\{(\w+)\}", op_path)
            if len(need) != sum(
                1 for a in sig_args
                if not a.startswith(("ctx ", "params ", "reqEditors"))
            ):
                print(
                    f"WARNING: {pkg_dir}.{fname}: {len(need)} path params in path, "
                    f"signature has a different number of string args; skipping",
                    file=sys.stderr,
                )
                continue
            call_args = []
            p = 0
            for arg in sig_args:
                if arg.startswith(("ctx ", "reqEditors")):
                    continue
                if arg.startswith("params "):
                    init = params_init(pkg_import, params_struct, structs.get(params_struct))
                    if init is not None:
                        call_args.append(init)
                    continue
                call_args.append(PARAM_ARG.get(need[p], "s.missing"))
                p += 1
            need_list = ", ".join(f'"{p}"' for p in need)
            entries.append(
                (
                    pkg_dir,
                    f"    {{\n"
                    f'        name: "{pkg_dir}.{fname}",\n'
                    f"        need: []string{{{need_list}}},\n"
                    f"        call: func(ctx context.Context, s *scope) (*http.Response, error) {{\n"
                    f"            return s.{scope_field}.{fname}(ctx, {', '.join(call_args)})\n"
                    f"        }},\n"
                    f"    }},"
                )
            )

    entries.sort(key=lambda e: e[0] + e[1])
    used_pkgs = sorted({e[0] for e in entries})
    import_lines = []
    for pkg_dir in used_pkgs:
        imp, _f = PKG[pkg_dir]
        if imp == pkg_dir:
            import_lines.append(f'\t"github.com/RonaldPhilipsen/omada/cloud/{pkg_dir}"')
        else:
            import_lines.append(f'\t{imp} "github.com/RonaldPhilipsen/omada/cloud/{pkg_dir}"')

    header = (
        "//go:build live\n\n"
        "// Code generated by gen/gen_livetest.py from the checked-in\n"
        "// oapi-codegen clients (cloud/*/*.gen.go). DO NOT EDIT.\n"
        "//\n"
        "// One entry per GET operation: the closure calls the generated\n"
        "// method; need lists the path parameters the operation requires.\n"
        "// The harness skips an entry when any needed parameter cannot be\n"
        "// resolved from the discovered resource scope.\n"
    )
    content = f"""{header}
package livetest

import (
\t"context"
\t"net/http"

{chr(10).join(import_lines)}
)

// livePage and livePageSize are shared by generated params initializers for
// operations whose page fields are pointer types.
var (
\tlivePage     = int32(1)
\tlivePageSize = int32(10)
)

var liveOps = []opEntry{{
{chr(10).join(e[1] for e in entries)}
}}
"""
    OUT.parent.mkdir(parents=True, exist_ok=True)
    OUT.write_text(content)
    print(f"wrote {OUT.relative_to(REPO)}: {len(entries)} GET operations")


if __name__ == "__main__":
    sys.exit(main())
