#!/usr/bin/env python3
"""Patch the Omada northbound OpenAPI spec: add missing path parameters."""
import json
import re
import sys

OPERATION_METHODS = ("get", "post", "put", "patch", "delete")
PLACEHOLDER_RE = re.compile(r"\{([^/{}]+)\}")


def declared_path_params(path_item, operation):
    names = set()
    for params in (path_item.get("parameters") or [], operation.get("parameters") or []):
        for p in params:
            if isinstance(p, dict) and p.get("in") == "path" and p.get("name"):
                names.add(p["name"])
    return names


def main():
    if len(sys.argv) != 3:
        print(f"usage: {sys.argv[0]} <input.json> <output.json>", file=sys.stderr)
        return 2
    with open(sys.argv[1], "r", encoding="utf-8") as f:
        spec = json.load(f)

    added = 0
    for path, path_item in spec.get("paths", {}).items():
        if not (path.startswith("/openapi/") or path.startswith("/internal/")):
            continue
        for method in OPERATION_METHODS:
            operation = path_item.get(method)
            if not isinstance(operation, dict):
                continue
            if not operation.get("operationId"):
                print(f"error: no operationId on {method.upper()} {path}", file=sys.stderr)
                return 1
            declared = declared_path_params(path_item, operation)
            for name in PLACEHOLDER_RE.findall(path):
                if name in declared:
                    continue
                operation.setdefault("parameters", []).append(
                    {
                        "name": name,
                        "in": "path",
                        "required": True,
                        "description": name,
                        "schema": {"type": "string"},
                    }
                )
                added += 1

    with open(sys.argv[2], "w", encoding="utf-8") as f:
        json.dump(spec, f, sort_keys=True)
        f.write("\n")
    print(f"patched params: {added}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
