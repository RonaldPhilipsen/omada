#!/usr/bin/env python3
"""Verify the tag-to-package mapping claims every public operation exactly once."""
import json
import sys

OPERATION_METHODS = ("get", "post", "put", "patch", "delete")


def main():
    if len(sys.argv) != 3:
        print(f"usage: {sys.argv[0]} <patched-spec.json> <tags.json>", file=sys.stderr)
        return 2
    with open(sys.argv[1], "r", encoding="utf-8") as f:
        spec = json.load(f)
    with open(sys.argv[2], "r", encoding="utf-8") as f:
        packages = json.load(f)

    public_tags = set()
    nops = 0
    bad = []
    for path, path_item in spec.get("paths", {}).items():
        if path.startswith("/internal/"):
            continue
        for method in OPERATION_METHODS:
            operation = path_item.get(method)
            if not isinstance(operation, dict):
                continue
            nops += 1
            tags = set(operation.get("tags") or [])
            public_tags |= tags
            claimers = [pkg for pkg, pkg_tags in packages.items() if tags & set(pkg_tags)]
            if len(claimers) != 1:
                bad.append((method.upper(), path, sorted(tags), claimers))

    if bad:
        for method, path, tags, claimers in bad:
            print(f"error: {method} {path} tags={tags} claimed by {claimers}", file=sys.stderr)
        return 1

    print(f"OK: {nops} public operations, {len(public_tags)} tags, {len(packages)} packages")
    return 0


if __name__ == "__main__":
    sys.exit(main())
