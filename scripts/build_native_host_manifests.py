#!/usr/bin/env python3
"""Generate deterministic Ghost FTP Native Messaging host manifests."""

from __future__ import annotations

import argparse
import json
from pathlib import Path
import re
import sys

HOST_NAME = "com.ghostftp.bridge"
HOST_DESCRIPTION = "Ghost FTP local browser bridge"
FIREFOX_EXTENSION_ID = "ghostftp-connection-helper@ghostftp.com"
CHROMIUM_ID = re.compile(r"^[a-p]{32}$")
CONTROL = re.compile(r"[\x00\r\n]")


def normalize_host_path(value: str) -> str:
    if CONTROL.search(value):
        raise ValueError("native host path contains control characters")
    path = Path(value).expanduser()
    if not path.is_absolute():
        raise ValueError("native host path must be absolute")
    text = str(path)
    if not text or len(text) > 32767:
        raise ValueError("native host path is invalid")
    return text


def normalize_chromium_ids(values: list[str]) -> list[str]:
    result: list[str] = []
    seen: set[str] = set()
    for raw in values:
        value = raw.strip().lower()
        if not CHROMIUM_ID.fullmatch(value):
            raise ValueError(f"invalid Chromium extension ID: {raw!r}")
        if value not in seen:
            seen.add(value)
            result.append(value)
    result.sort()
    return result


def firefox_manifest(host_path: str) -> dict[str, object]:
    return {
        "name": HOST_NAME,
        "description": HOST_DESCRIPTION,
        "path": normalize_host_path(host_path),
        "type": "stdio",
        "allowed_extensions": [FIREFOX_EXTENSION_ID],
    }


def chromium_manifest(host_path: str, extension_ids: list[str]) -> dict[str, object]:
    ids = normalize_chromium_ids(extension_ids)
    if not ids:
        raise ValueError("at least one explicit Chromium extension ID is required")
    return {
        "name": HOST_NAME,
        "description": HOST_DESCRIPTION,
        "path": normalize_host_path(host_path),
        "type": "stdio",
        "allowed_origins": [f"chrome-extension://{extension_id}/" for extension_id in ids],
    }


def write_json(path: Path, value: dict[str, object]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    encoded = json.dumps(value, indent=2, ensure_ascii=False, sort_keys=False) + "\n"
    path.write_text(encoded, encoding="utf-8", newline="\n")


def build(output_dir: Path, host_path: str, chromium_ids: list[str], require_chromium: bool) -> list[Path]:
    host = normalize_host_path(host_path)
    output_dir.mkdir(parents=True, exist_ok=True)

    firefox_path = output_dir / f"{HOST_NAME}.firefox.json"
    write_json(firefox_path, firefox_manifest(host))
    outputs = [firefox_path]

    chromium_path = output_dir / f"{HOST_NAME}.chromium.json"
    ids = normalize_chromium_ids(chromium_ids)
    if ids:
        write_json(chromium_path, chromium_manifest(host, ids))
        outputs.append(chromium_path)
    else:
        chromium_path.unlink(missing_ok=True)
        if require_chromium:
            raise ValueError("Chromium host registration requested without an official extension ID")

    return outputs


def main() -> int:
    parser = argparse.ArgumentParser(description="Build Ghost FTP Native Messaging host manifests")
    parser.add_argument("--host-path", required=True, help="absolute installed native-host executable path")
    parser.add_argument("--output-dir", type=Path, required=True)
    parser.add_argument(
        "--chromium-id",
        action="append",
        default=[],
        help="explicit Chrome/Edge/Opera extension ID; may be provided multiple times",
    )
    parser.add_argument(
        "--require-chromium",
        action="store_true",
        help="fail closed when no explicit Chromium extension ID is supplied",
    )
    args = parser.parse_args()

    try:
        outputs = build(args.output_dir, args.host_path, args.chromium_id, args.require_chromium)
    except (OSError, UnicodeError, ValueError) as exc:
        print(f"NATIVE_HOST_MANIFEST_BUILD=FAILED: {exc}", file=sys.stderr)
        return 1

    print("NATIVE_HOST_MANIFEST_BUILD=PASS")
    print(f"NATIVE_HOST_NAME={HOST_NAME}")
    for output in outputs:
        print(f"NATIVE_HOST_MANIFEST={output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
