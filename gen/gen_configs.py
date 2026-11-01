#!/usr/bin/env python3
"""Write per-package oapi-codegen configs from tags.json.

For each package in tags.json, emits configs/<pkg>.yaml with a client+models
generation config scoped to the package's tags.
"""
import json
from pathlib import Path

HERE = Path(__file__).resolve().parent

GO_PACKAGE_NAMES = {"switch": "switchpkg"}


def go_package_name(pkg):
    return GO_PACKAGE_NAMES.get(pkg, pkg)


def yaml_scalar(value):
    """Return a safe YAML plain scalar for a tag string."""
    text = str(value)
    if not text or text != text.strip():
        return json.dumps(text)
    if text[0] in "-?:,[]{}#&*!|>'\"%@`" or ": " in text or " #" in text:
        return json.dumps(text)
    return text


def render_config(pkg, tags):
    lines = [
        f"package: {go_package_name(pkg)}",
        "generate:",
        "  client: true",
        "  models: true",
        f"output: ../cloud/{pkg}/{pkg}.gen.go",
        "output-options:",
        "  include-tags:",
    ]
    for tag in tags:
        lines.append(f"    - {yaml_scalar(tag)}")
    lines.append("  resolve-type-name-collisions: true")
    return "\n".join(lines) + "\n"


def main():
    tags = json.loads((HERE / "tags.json").read_text())
    configs = HERE / "configs"
    configs.mkdir(exist_ok=True)
    for pkg, pkg_tags in tags.items():
        path = configs / f"{pkg}.yaml"
        path.write_text(render_config(pkg, pkg_tags))
        print(f"wrote {path}")


if __name__ == "__main__":
    main()
