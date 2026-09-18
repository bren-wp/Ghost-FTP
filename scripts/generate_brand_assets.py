#!/usr/bin/env python3
"""Generate and validate deterministic Ghost FTP desktop brand assets.

The canonical Ghost FTP mark follows the approved dark reference UI: a compact
ghost silhouette carrying two transfer arrows, rendered with a cyan -> electric
blue -> violet gradient on a transparent/dark rounded-square application tile.
The renderer is dependency-free so Windows builds can reproduce PNG/ICO assets
without network access or bundled font/image tooling.
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

TRANSPARENT = (0, 0, 0, 0)
CHARCOAL = (8, 14, 20, 255)
CYAN = (0, 229, 255, 255)
BLUE = (59, 130, 246, 255)
VIOLET = (139, 92, 246, 255)
MIST = (229, 231, 235, 255)


def _mix(a: tuple[int, int, int, int], b: tuple[int, int, int, int], t: float) -> tuple[int, int, int, int]:
    t = min(1.0, max(0.0, t))
    return tuple(round(a[i] * (1.0 - t) + b[i] * t) for i in range(4))


def _gradient(x: float, y: float) -> tuple[int, int, int, int]:
    t = min(1.0, max(0.0, (0.62 * x + 0.38 * y - 42.0) / 205.0))
    if t < 0.52:
        return _mix(CYAN, BLUE, t / 0.52)
    return _mix(BLUE, VIOLET, (t - 0.52) / 0.48)


def _inside_rounded_rect(x: float, y: float, left: float, top: float, right: float, bottom: float, radius: float) -> bool:
    if x < left or x >= right or y < top or y >= bottom:
        return False
    cx = min(max(x, left + radius), right - radius)
    cy = min(max(y, top + radius), bottom - radius)
    dx = x - cx
    dy = y - cy
    return dx * dx + dy * dy <= radius * radius


def _inside_ellipse(x: float, y: float, cx: float, cy: float, rx: float, ry: float) -> bool:
    if rx <= 0 or ry <= 0:
        return False
    dx = (x - cx) / rx
    dy = (y - cy) / ry
    return dx * dx + dy * dy <= 1.0


def _inside_rotated_ellipse(x: float, y: float, cx: float, cy: float, rx: float, ry: float, angle: float) -> bool:
    s, c = math.sin(angle), math.cos(angle)
    dx, dy = x - cx, y - cy
    px = dx * c + dy * s
    py = -dx * s + dy * c
    return (px / rx) ** 2 + (py / ry) ** 2 <= 1.0


def _inside_triangle(x: float, y: float, a: tuple[float, float], b: tuple[float, float], c: tuple[float, float]) -> bool:
    def sign(p1, p2, p3):
        return (p1[0] - p3[0]) * (p2[1] - p3[1]) - (p2[0] - p3[0]) * (p1[1] - p3[1])
    p = (x, y)
    d1, d2, d3 = sign(p, a, b), sign(p, b, c), sign(p, c, a)
    neg = d1 < 0 or d2 < 0 or d3 < 0
    pos = d1 > 0 or d2 > 0 or d3 > 0
    return not (neg and pos)


def _inside_arrow(x: float, y: float, y0: float, length: float, thickness: float) -> bool:
    left = 54.0
    body_right = left + length - 28.0
    tip = left + length
    if left <= x <= body_right and y0 - thickness / 2 <= y <= y0 + thickness / 2:
        return True
    return _inside_triangle(
        x,
        y,
        (body_right - 2.0, y0 - thickness * 1.35),
        (tip, y0),
        (body_right - 2.0, y0 + thickness * 1.35),
    )


def _inside_ghost(x: float, y: float) -> bool:
    # Sleek forward-leaning ghost: rounded crown, tapered lower body and a
    # streaming tail that visually merges with the transfer direction.
    head = _inside_rotated_ellipse(x, y, 143.0, 101.0, 54.0, 52.0, -0.18)
    shoulder = _inside_rotated_ellipse(x, y, 136.0, 132.0, 69.0, 52.0, -0.18)
    body = _inside_triangle(x, y, (83.0, 118.0), (193.0, 94.0), (171.0, 197.0))
    tail = _inside_triangle(x, y, (91.0, 128.0), (171.0, 197.0), (61.0, 176.0))
    return head or shoulder or body or tail


def _sample_reference_pixel(x: float, y: float) -> tuple[int, int, int, int]:
    # Rounded app tile with a thin electric-blue rim.
    if not _inside_rounded_rect(x, y, 8.0, 8.0, 248.0, 248.0, 42.0):
        return TRANSPARENT

    inner = _inside_rounded_rect(x, y, 13.0, 13.0, 243.0, 243.0, 38.0)
    tile = CHARCOAL if inner else _mix(BLUE, VIOLET, 0.42)

    ghost = _inside_ghost(x, y)
    if ghost:
        tile = _gradient(x, y)

    # Eyes are negative-space dark ovals.
    if ghost and (
        _inside_rotated_ellipse(x, y, 151.0, 92.0, 7.0, 12.0, 0.18)
        or _inside_rotated_ellipse(x, y, 176.0, 99.0, 6.5, 11.0, 0.18)
    ):
        return CHARCOAL

    # Two transfer arrows cut through the lower half of the ghost and extend
    # into the streaming tail. Their bright cyan/blue fill stays legible at 16px.
    if _inside_arrow(x, y, 139.0, 102.0, 10.0):
        return _mix(CYAN, BLUE, min(1.0, max(0.0, (x - 54.0) / 102.0)))
    if _inside_arrow(x, y, 164.0, 82.0, 8.0):
        return _mix(BLUE, VIOLET, min(1.0, max(0.0, (x - 54.0) / 82.0)))

    # Small motion dots mirror the approved mark without adding noisy detail.
    for cx, cy, r, color in (
        (49.0, 139.0, 3.0, CYAN),
        (42.0, 164.0, 2.8, BLUE),
        (55.0, 184.0, 2.5, VIOLET),
    ):
        if _inside_ellipse(x, y, cx, cy, r, r):
            return color

    return tile


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
            struct.pack("<BBBBHHII", width, height, 0, 0, 1, 32, len(image), offset)
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
    parser.add_argument("--materialize", action="store_true", help="write deterministic cyan/blue/violet PNG/ICO assets before validation")
    parser.add_argument("--check", action="store_true", help="validate the current assets without rewriting them")
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
    print("CANONICAL_LOGO=GHOST_TRANSFER_CYAN_BLUE_VIOLET")
    print("ACTIVE_BRAND_ASSETS=WINDOWS,LINUX,MACOS,ANDROID")
    print("RETIRED_WEB_PWA_ASSETS=BLOCKED")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
