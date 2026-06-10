#!/usr/bin/env python3
"""Generate Ansible inventory from Terraform outputs (vm stack)."""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
from pathlib import Path


def load_tf_outputs(terraform_dir: Path) -> dict:
    result = subprocess.run(
        ["terraform", "-chdir", str(terraform_dir), "output", "-json"],
        check=True,
        capture_output=True,
        text=True,
    )
    raw = json.loads(result.stdout)
    return {key: value.get("value") for key, value in raw.items()}


def render_inventory(outputs: dict) -> str:
    public_ips = outputs.get("public_ips") or []
    private_ips = outputs.get("private_ips") or []
    vm_names = outputs.get("vm_names") or []
    admin_user = outputs.get("admin_username") or "azureadmin"

    if not public_ips:
        raise SystemExit("No public_ips in terraform outputs — run terraform apply first.")

    lines = ["[webservers]"]
    for i, public_ip in enumerate(public_ips):
        name = vm_names[i] if i < len(vm_names) else f"app-{i}"
        private_ip = private_ips[i] if i < len(private_ips) else ""
        lines.append(
            f"{name} ansible_host={public_ip} ansible_user={admin_user} private_ip={private_ip}"
        )

    lines.extend(
        [
            "",
            "[webservers:vars]",
            "ansible_python_interpreter=/usr/bin/python3",
        ]
    )
    return "\n".join(lines) + "\n"


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "terraform_dir",
        nargs="?",
        default="infra/terraform/vm",
        help="Path to terraform module (default: infra/terraform/vm)",
    )
    parser.add_argument(
        "-o",
        "--output",
        type=Path,
        default=Path("infra/ansible/inventory/hosts.ini"),
        help="Inventory output file",
    )
    args = parser.parse_args()

    terraform_dir = Path(args.terraform_dir)
    if not terraform_dir.is_dir():
        print(f"terraform dir not found: {terraform_dir}", file=sys.stderr)
        raise SystemExit(1)

    inventory = render_inventory(load_tf_outputs(terraform_dir))
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(inventory, encoding="utf-8")
    print(inventory, end="")


if __name__ == "__main__":
    main()
