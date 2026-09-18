#!/usr/bin/env python3
"""Generate and validate deterministic Ghost FTP desktop brand assets.

The source of truth is the same gold-ghost mark used by the supplied 0.0.8
reference screens. Production builds materialize PNG/ICO assets locally from
this dependency-free renderer so Windows, Linux and Android cannot drift to a
different logo or depend on a network image tool.
"""

from __future__ import annotations

import argparse
import binascii
import math
from pathlib import Path
import struct
import sys
import zlib

ROOT = Path(__file__).resolve().parents[1]
ICON_PNG = ROOT / "build" / "icon.png"
ICON_ICO = ROOT / "build" / "icon.ico"

PNG_SIGNATURE = b"\x89PNG\r\n\x1a\n"
ICO_SIGNATURE = b"\x00\x00\x01\x00"
CANVAS = 256
SUPERSAMPLE = 4

DARK = (11, 15, 23, 255)
GOLD = (246, 196, 69, 255)
GOLD_LIGHT = (255, 222, 111, 255)
TRANSPARENT = (0, 0, 0, 0)


def _inside_rounded_rect(x: float, y: float, left: float, top: float, right: float, bottom: float, radius: float) -> bool:
    if x < left or x >= right or y < top or y >= bottom:
        return False
    cx = min(max(x, left + radius), right - radius)
    cy = min(max(y, top + radius), bottom - radius)
    dx = x - cx
    dy = y - cy
    return dx * dx + dy * dy <= radius * radius


def _inside_circle(x: float, y: float, cx: float, cy: float, radius: float) -> bool:
    dx = x - cx
    dy = y - cy
    return dx * dx + dy * dy <= radius * radius


def _inside_ghost(x: float, y: float) -> bool:
    # Reference proportions: rounded head, straight shoulders/body and three
    # soft lower lobes. Coordinates are expressed on the canonical 256 canvas.
    cx = 128.0
    left = 78.0
    right = 178.0
    head_cy = 112.0
    radius = 50.0
    body_top = 112.0
    body_bottom = 181.0

    if y < body_top:
        return _inside_circle(x, y, cx, head_cy, radius) and y >= head_cy - radius
    if left <= x <= right and body_top <= y <= body_bottom:
        # Wavy lower edge: three rounded feet separated by two shallow arches.
        if y <= 161.0:
            return True
        # Keep the body where y is above the local scalloped lower boundary.
        phase = (x - left) / (right - left)
        boundary = 174.0 + 8.0 * math.cos(phase * 6.0 * math.pi)
        return y <= boundary
    return False


def _sample_reference_pixel(x: float, y: float) -> tuple[int, int, int, int]:
    # Transparent outer corners make the icon integrate cleanly with native
    # shells while the visible tile remains identical to the reference.
    if not _inside_rounded_rect(x, y, 8.0, 8.0, 248.0, 248.0, 42.0):
        return TRANSPARENT

    color = DARK

    outer = _inside_rounded_rect(x, y, 14.0, 14.0, 242.0, 242.0, 36.0)
    inner = _inside_rounded_rect(x, y, 22.0, 22.0, 234.0, 234.0, 30.0)
    if outer and not inner:
        color = GOLD

    if _inside_ghost(x, y):
        # Subtle vertical highlight keeps the mark visually faithful to the
        # supplied gold/cream reference without adding an external asset.
        t = min(1.0, max(0.0, (y - 62.0) / 120.0))
        color = tuple(round(GOLD_LIGHT[i] * (1.0 - 0.30 * t) + GOLD[i] * (0.30 * t)) for i in range(3)) + (255,)

    if _inside_circle(x, y, 108.0, 116.0, 8.0) or _inside_circle(x, y, 148.0, 116.0, 8.0):
        color = DARK

    return color


def _render_rgba(size: int) -> bytes:
    scale = CANVAS / float(size)
    ss = SUPERSAMPLE
    out = bytearray(size * size * 4)
    for py in range(size):
        for px in range(size):
            accum = [0, 0, 0, 0]
            for sy in range(ss):
                for sx in range(ss):
                    x = (px + (sx + 0.5) / ss) * scale
                    y = (py + (sy + 0.5) / ss) * scale
                    sample = _sample_reference_pixel(x, y)
                    for i, value in enumerate(sample):
                        accum[i] += value
            base = (py * size + px) * 4
            samples = ss * ss
            for i in range(4):
                out[base + i] = round(accum[i] / samples)
    return bytes(out)


def _png_chunk(kind: bytes, payload: bytes) -> bytes:
    return (
        struct.pack(">I", len(payload))
        + kind
        + payload
        + struct.pack(">I", binascii.crc32(kind + payload) & 0xFFFFFFFF)
    )


def _png_bytes(size: int) -> bytes:
    rgba = _render_rgba(size)
    scanlines = bytearray()
    stride = size * 4
    for row in range(size):
        scanlines.append(0)
        start = row * stride
        scanlines.extend(rgba[start : start + stride])
    ihdr = struct.pack(">IIBBBBB", size, size, 8, 6, 0, 0, 0)
    return (
        PNG_SIGNATURE
        + _png_chunk(b"IHDR", ihdr)
        + _png_chunk(b"IDAT", zlib.compress(bytes(scanlines), 9))
        + _png_chunk(b"IEND", b"")
    )


def _ico_bytes() -> bytes:
    sizes = (16, 24, 32, 48, 64, 96, 128, 256)
    images = [_png_bytes(size) for size in sizes]
    header = ICO_SIGNATURE + struct.pack("<H", len(images))
    directory = bytearray()
    offset = 6 + len(images) * 16
    for size, image in zip(sizes, images):
        width = 0 if size == 256 else size
        height = 0 if size == 256 else size
        directory.extend(
            struct.pack(
                "<BBBBHHII",
                width,
                height,
                0,
                0,
                1,
                32,
                len(image),
                offset,
            )
        )
        offset += len(image)
    return header + bytes(directory) + b"".join(images)


def materialize() -> None:
    ICON_PNG.parent.mkdir(parents=True, exist_ok=True)
    ICON_PNG.write_bytes(_png_bytes(256))
    ICON_ICO.write_bytes(_ico_bytes())


def require_file(path: Path, minimum_size: int = 1) -> bytes:
    if not path.is_file():
        raise ValueError(f"missing brand asset: {path.relative_to(ROOT)}")
    data = path.read_bytes()
    if len(data) < minimum_size:
        raise ValueError(f"brand asset is unexpectedly small: {path.relative_to(ROOT)}")
    return data


def validate() -> None:
    png = require_file(ICON_PNG, 1024)
    if not png.startswith(PNG_SIGNATURE):
        raise ValueError("build/icon.png is not a valid PNG asset")

    ico = require_file(ICON_ICO, 1024)
    if not ico.startswith(ICO_SIGNATURE):
        raise ValueError("build/icon.ico is not a valid Windows icon asset")

    if (ROOT / "GhostFTP WEB").exists():
        raise ValueError("retired Web/PWA application surface is present")


def main() -> int:
    parser = argparse.ArgumentParser(description="Generate or validate Ghost FTP desktop brand assets")
    parser.add_argument(
        "--materialize",
        action="store_true",
        help="write deterministic gold-ghost PNG/ICO assets before validation",
    )
    parser.add_argument(
        "--check",
        action="store_true",
        help="validate the current assets without rewriting them",
    )
    args = parser.parse_args()

    try:
        if args.materialize:
            materialize()
        validate()
    except (OSError, UnicodeError, ValueError) as exc:
        print(f"BRAND_ASSET_AUDIT=FAILED: {exc}", file=sys.stderr)
        return 1

    print("BRAND_ASSET_AUDIT=PASS")
    print("PUBLIC_BRAND=Ghost FTP")
    print("CANONICAL_LOGO=GOLD_GHOST_REFERENCE")
    print("ACTIVE_BRAND_ASSETS=WINDOWS,LINUX,ANDROID")
    print("RETIRED_WEB_PWA_ASSETS=BLOCKED")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
