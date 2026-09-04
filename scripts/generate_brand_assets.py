#!/usr/bin/env python3
"""Validate the committed Ghost FTP brand assets.

Brand binaries are committed to the repository and are not regenerated during
production builds. This keeps builds deterministic and avoids maintaining a
custom PNG/ICO renderer solely for CI. The historical GhostFTP documentation
banner is intentionally no longer part of the brand contract.
"""

from __future__ import annotations

import argparse
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
ICON_PNG = ROOT / "build" / "icon.png"
ICON_ICO = ROOT / "build" / "icon.ico"
WEB_LOGO = ROOT / "GhostFTP WEB" / "assets" / "images" / "logo.svg"
WEB_MARK = ROOT / "GhostFTP WEB" / "assets" / "images" / "mark.svg"

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

    logo = require_file(WEB_LOGO).decode("utf-8", errors="strict")
    if "Ghost FTP" not in logo or "GhostFTP" in logo:
        raise ValueError("web logo does not contain the canonical Ghost FTP brand")

    mark = require_file(WEB_MARK).decode("utf-8", errors="strict")
    if "<svg" not in mark:
        raise ValueError("web mark is not an SVG asset")


def main() -> int:
    parser = argparse.ArgumentParser(description="Validate Ghost FTP brand assets")
    parser.add_argument(
        "--check",
        action="store_true",
        help="retained for build-script compatibility; validation is always performed",
    )
    parser.parse_args()

    try:
        validate()
    except (OSError, UnicodeError, ValueError) as exc:
        print(f"BRAND_ASSET_AUDIT=FAILED: {exc}", file=sys.stderr)
        return 1

    print("BRAND_ASSET_AUDIT=PASS")
    print("PUBLIC_BRAND=Ghost FTP")
    print("LEGACY_GhostFTP_DOC_HEADER=REMOVED")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
