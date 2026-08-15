#!/usr/bin/env python3
"""Create a small deterministic installer payload ZIP for ByFTP."""
from __future__ import annotations
import argparse
import hashlib
import json
import zipfile
from pathlib import Path

FIXED_TIME = (2026, 1, 1, 0, 0, 0)

def add(zf: zipfile.ZipFile, src: Path, arcname: str) -> dict[str, object]:
    data = src.read_bytes()
    info = zipfile.ZipInfo(arcname, FIXED_TIME)
    info.compress_type = zipfile.ZIP_DEFLATED
    info.external_attr = 0o755 << 16
    zf.writestr(info, data, compresslevel=9)
    return {"name": arcname, "size": len(data), "sha256": hashlib.sha256(data).hexdigest()}

def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--app", required=True, type=Path)
    ap.add_argument("--uninstaller", required=True, type=Path)
    ap.add_argument("--output", required=True, type=Path)
    args = ap.parse_args()
    args.output.parent.mkdir(parents=True, exist_ok=True)
    manifest: list[dict[str, object]] = []
    with zipfile.ZipFile(args.output, "w") as zf:
        manifest.append(add(zf, args.app, "ByFTP.exe"))
        manifest.append(add(zf, args.uninstaller, "Uninstall.exe"))
        info = zipfile.ZipInfo("manifest.json", FIXED_TIME)
        info.compress_type = zipfile.ZIP_DEFLATED
        info.external_attr = 0o644 << 16
        zf.writestr(info, json.dumps({"schema": 1, "files": manifest}, separators=(",", ":"), sort_keys=True).encode("utf-8"), compresslevel=9)
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
