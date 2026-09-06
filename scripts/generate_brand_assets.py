#!/usr/bin/env python3
"""Validate the committed Ghost FTP desktop brand assets.

Brand binaries are committed to the repository and are not regenerated during
production builds. This keeps Windows/Linux builds deterministic and avoids a
runtime or build-time dependency on a custom image renderer. Retired Web/PWA
assets are deliberately outside the maintained desktop brand contract.
"""

from __future__ import annotations

import argparse
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
ICON_PNG = ROOT / "build" / "icon.png"
ICON_ICO = ROOT / "build" / "icon.ico"

PNG_SIGNATURE = b"\x89PNG\r\n\x1a\n"
ICO_SIGNATURE = b"\x00\x00\x01\x00"
MIN_BINARY_SIZE = 256


def require_file(path: Path, minimum_size: int = 1) -> bytes:
    if not path.is_file():
        raise ValueError(f"missing brand asset: {path.relative_to(ROOT)}")
    data = path.read_bytes()
    if len(data) < minimum_size:
        raise ValueError(f"brand asset is unexpectedly small: {path.relative_to(ROOT)}")
    return data


def validate() -> None:
    png = require_file(ICON_PNG, MIN_BINARY_SIZE)
    if not png.startswith(PNG_SIGNATURE):
        raise ValueError("build/icon.png is not a valid PNG asset")

    ico = require_file(ICON_ICO, MIN_BINARY_SIZE)
    if not ico.startswith(ICO_SIGNATURE):
        raise ValueError("build/icon.ico is not a valid Windows icon asset")

    # Fail closed if the retired application surface is accidentally restored.
    if (ROOT / "GhostFTP WEB").exists():
        raise ValueError("retired Web/PWA application surface is present")


def main() -> int:
    parser = argparse.ArgumentParser(description="Validate Ghost FTP desktop brand assets")
    parser.add_argument(
        "--check",
        action="store_true",
        help="accepted for build-script compatibility; validation is always performed",
    )
    parser.parse_args()

    try:
        validate()
    except (OSError, UnicodeError, ValueError) as exc:
        print(f"BRAND_ASSET_AUDIT=FAILED: {exc}", file=sys.stderr)
        return 1

    print("BRAND_ASSET_AUDIT=PASS")
    print("PUBLIC_BRAND=Ghost FTP")
    print("ACTIVE_BRAND_ASSETS=WINDOWS_LINUX")
    print("RETIRED_WEB_PWA_ASSETS=BLOCKED")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
