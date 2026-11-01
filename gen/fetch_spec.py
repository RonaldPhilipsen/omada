#!/usr/bin/env python3
"""Fetch the Omada northbound OpenAPI spec and write the raw JSON bytes."""
import sys
import urllib.request

DEFAULT_URL = "https://use1-omada-northbound.tplinkcloud.com/v3/api-docs"


def main():
    url = sys.argv[1] if len(sys.argv) > 1 else DEFAULT_URL
    out = sys.argv[2] if len(sys.argv) > 2 else "api-docs.json"
    req = urllib.request.Request(url, headers={"User-Agent": "omada-gen/1.0"})
    with urllib.request.urlopen(req) as resp:
        if resp.status != 200:
            print(f"error: expected HTTP 200, got {resp.status}", file=sys.stderr)
            return 1
        data = resp.read()
    with open(out, "wb") as f:
        f.write(data)
    print(f"wrote {len(data)} bytes to {out}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
