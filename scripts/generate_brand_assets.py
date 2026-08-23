#!/usr/bin/env python3
"""Generate and verify the official ByFTP icon and English documentation header.

Generated PNG compression can differ across zlib builds even when the decoded
image is identical. Verification therefore validates PNG structure, CRCs,
dimensions and every RGBA pixel instead of requiring one zlib byte stream.
"""

from __future__ import annotations

import argparse
import binascii
import struct
import zlib
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
ICON_PNG = ROOT / "build" / "icon.png"
ICON_ICO = ROOT / "build" / "icon.ico"
HEADER_PNG = ROOT / "docs" / "images" / "byftp-header.png"

BG = (10, 18, 32, 255)
PANEL = (23, 39, 62, 255)
TEXT = (238, 245, 255, 255)
MUTED = (157, 174, 196, 255)
A1 = (66, 210, 196, 255)
A2 = (90, 142, 255, 255)

FONT = {
    "A": ["01110", "10001", "10001", "11111", "10001", "10001", "10001"],
    "B": ["11110", "10001", "10001", "11110", "10001", "10001", "11110"],
    "C": ["01111", "10000", "10000", "10000", "10000", "10000", "01111"],
    "D": ["11110", "10001", "10001", "10001", "10001", "10001", "11110"],
    "E": ["11111", "10000", "10000", "11110", "10000", "10000", "11111"],
    "F": ["11111", "10000", "10000", "11110", "10000", "10000", "10000"],
    "I": ["11111", "00100", "00100", "00100", "00100", "00100", "11111"],
    "L": ["10000", "10000", "10000", "10000", "10000", "10000", "11111"],
    "N": ["10001", "11001", "10101", "10011", "10001", "10001", "10001"],
    "P": ["11110", "10001", "10001", "11110", "10000", "10000", "10000"],
    "R": ["11110", "10001", "10001", "11110", "10100", "10010", "10001"],
    "S": ["01111", "10000", "10000", "01110", "00001", "00001", "11110"],
    "T": ["11111", "00100", "00100", "00100", "00100", "00100", "00100"],
    "U": ["10001", "10001", "10001", "10001", "10001", "10001", "01110"],
    "Y": ["10001", "10001", "01010", "00100", "00100", "00100", "00100"],
    " ": ["00000"] * 7,
}

PNG_SIGNATURE = b"\x89PNG\r\n\x1a\n"
ICO_SIZES = (16, 24, 32, 48, 64, 128, 256)


def canvas(w: int, h: int) -> bytearray:
    pixels = bytearray(w * h * 4)
    for y in range(h):
        t = y / max(1, h - 1)
        color = tuple(
            round(BG[i] * (1 - t) + (19, 32, 52, 255)[i] * t)
            for i in range(4)
        )
        pixels[y * w * 4 : (y + 1) * w * 4] = bytes(color) * w
    return pixels


def rect(p: bytearray, w: int, h: int, x0: int, y0: int, x1: int, y1: int, color: tuple[int, int, int, int]) -> None:
    x0 = max(0, x0)
    y0 = max(0, y0)
    x1 = min(w, x1)
    y1 = min(h, y1)
    row = bytes(color) * max(0, x1 - x0)
    for y in range(y0, y1):
        p[(y * w + x0) * 4 : (y * w + x1) * 4] = row


def rr(p: bytearray, w: int, h: int, x0: int, y0: int, x1: int, y1: int, radius: int, color: tuple[int, int, int, int]) -> None:
    for y in range(y0, y1):
        for x in range(x0, x1):
            dx = max(x0 + radius - x, 0, x - (x1 - radius - 1))
            dy = max(y0 + radius - y, 0, y - (y1 - radius - 1))
            if dx * dx + dy * dy <= radius * radius:
                rect(p, w, h, x, y, x + 1, y + 1, color)


def line(p: bytearray, w: int, h: int, x0: int, y0: int, x1: int, y1: int, thickness: int, color: tuple[int, int, int, int]) -> None:
    dx = abs(x1 - x0)
    sx = 1 if x0 < x1 else -1
    dy = -abs(y1 - y0)
    sy = 1 if y0 < y1 else -1
    error = dx + dy
    while True:
        half = max(1, thickness) // 2
        rect(p, w, h, x0 - half, y0 - half, x0 + half + 1, y0 + half + 1, color)
        if x0 == x1 and y0 == y1:
            break
        twice = 2 * error
        if twice >= dy:
            error += dy
            x0 += sx
        if twice <= dx:
            error += dx
            y0 += sy


def arrow(p: bytearray, w: int, h: int, x0: int, y: int, x1: int, color: tuple[int, int, int, int]) -> None:
    line(p, w, h, x0, y, x1, y, max(2, w // 180), color)
    delta = max(5, abs(x1 - x0) // 5)
    side = -1 if x1 > x0 else 1
    line(p, w, h, x1, y, x1 + side * delta, y - delta, max(2, w // 180), color)
    line(p, w, h, x1, y, x1 + side * delta, y + delta, max(2, w // 180), color)


def icon(size: int) -> tuple[int, int, bytearray]:
    p = canvas(size, size)
    margin = max(2, size // 16)
    rr(p, size, size, margin, margin, size - margin, size - margin, max(3, size // 7), PANEL)
    panel_w = max(5, size // 4)
    panel_h = max(8, size // 2)
    y = (size - panel_h) // 2
    left_x = size // 8
    right_x = size - size // 8 - panel_w
    rr(p, size, size, left_x, y, left_x + panel_w, y + panel_h, max(2, size // 32), TEXT)
    rr(p, size, size, right_x, y, right_x + panel_w, y + panel_h, max(2, size // 32), TEXT)
    for yy in (y + panel_h // 4, y + panel_h // 2, y + 3 * panel_h // 4):
        rect(p, size, size, left_x + panel_w // 5, yy, left_x + 4 * panel_w // 5, yy + max(1, size // 40), BG)
        rect(p, size, size, right_x + panel_w // 5, yy, right_x + 4 * panel_w // 5, yy + max(1, size // 40), BG)
    arrow(p, size, size, left_x + panel_w + size // 20, size // 2 - size // 10, right_x - size // 20, A1)
    arrow(p, size, size, right_x - size // 20, size // 2 + size // 10, left_x + panel_w + size // 20, A2)
    return size, size, p


def text(p: bytearray, w: int, h: int, value: str, x: int, y: int, scale: int, color: tuple[int, int, int, int]) -> None:
    cursor = x
    for ch in value.upper():
        for gy, row in enumerate(FONT.get(ch, FONT[" "])):
            for gx, bit in enumerate(row):
                if bit == "1":
                    rect(
                        p,
                        w,
                        h,
                        cursor + gx * scale,
                        y + gy * scale,
                        cursor + (gx + 1) * scale,
                        y + (gy + 1) * scale,
                        color,
                    )
        cursor += 6 * scale


def header() -> tuple[int, int, bytearray]:
    w, h = 1200, 320
    p = canvas(w, h)
    rr(p, w, h, 40, 40, 280, 280, 42, PANEL)
    iw, ih, ip = icon(200)
    for y in range(ih):
        p[((60 + y) * w + 60) * 4 : ((60 + y) * w + 60 + iw) * 4] = ip[y * iw * 4 : (y + 1) * iw * 4]
    text(p, w, h, "BYFTP", 335, 72, 17, TEXT)
    text(p, w, h, "SECURE FILE TRANSFER", 338, 215, 5, MUTED)
    rect(p, w, h, 338, 270, 1095, 276, A1)
    return w, h, p


def png(w: int, h: int, p: bytes | bytearray) -> bytes:
    def chunk(kind: bytes, data: bytes) -> bytes:
        body = kind + data
        return (
            struct.pack(">I", len(data))
            + body
            + struct.pack(">I", binascii.crc32(body) & 0xFFFFFFFF)
        )

    raw = bytearray()
    for y in range(h):
        raw.append(0)
        raw.extend(p[y * w * 4 : (y + 1) * w * 4])
    return (
        PNG_SIGNATURE
        + chunk(b"IHDR", struct.pack(">IIBBBBB", w, h, 8, 6, 0, 0, 0))
        + chunk(b"IDAT", zlib.compress(bytes(raw), 9))
        + chunk(b"IEND", b"")
    )


def ico() -> bytes:
    images = []
    for size in ICO_SIZES:
        w, h, p = icon(size)
        images.append(png(w, h, p))

    offset = 6 + 16 * len(images)
    entries = []
    for size, data in zip(ICO_SIZES, images):
        encoded_size = 0 if size == 256 else size
        entries.append(
            struct.pack(
                "<BBBBHHII",
                encoded_size,
                encoded_size,
                0,
                0,
                1,
                32,
                len(data),
                offset,
            )
        )
        offset += len(data)
    return struct.pack("<HHH", 0, 1, len(images)) + b"".join(entries) + b"".join(images)


def expected() -> dict[Path, bytes]:
    w, h, p = icon(512)
    hw, hh, hp = header()
    return {
        ICON_PNG: png(w, h, p),
        ICON_ICO: ico(),
        HEADER_PNG: png(hw, hh, hp),
    }


def decode_png(data: bytes) -> tuple[int, int, bytes]:
    """Decode the strict RGBA/filter-0 PNG format emitted by this generator."""
    if not data.startswith(PNG_SIGNATURE):
        raise ValueError("invalid PNG signature")

    pos = len(PNG_SIGNATURE)
    ihdr: bytes | None = None
    idat: list[bytes] = []
    saw_iend = False
    chunk_order: list[bytes] = []

    while pos < len(data):
        if len(data) - pos < 12:
            raise ValueError("truncated PNG chunk")
        length = struct.unpack_from(">I", data, pos)[0]
        end = pos + 12 + length
        if end > len(data):
            raise ValueError("truncated PNG chunk data")
        kind = data[pos + 4 : pos + 8]
        payload = data[pos + 8 : pos + 8 + length]
        stored_crc = struct.unpack_from(">I", data, pos + 8 + length)[0]
        actual_crc = binascii.crc32(kind + payload) & 0xFFFFFFFF
        if stored_crc != actual_crc:
            raise ValueError("PNG CRC mismatch")
        if kind not in {b"IHDR", b"IDAT", b"IEND"}:
            raise ValueError(f"unexpected PNG chunk {kind!r}")
        chunk_order.append(kind)
        if kind == b"IHDR":
            if ihdr is not None or idat or saw_iend:
                raise ValueError("invalid IHDR placement")
            ihdr = payload
        elif kind == b"IDAT":
            if ihdr is None or saw_iend:
                raise ValueError("invalid IDAT placement")
            idat.append(payload)
        else:
            if payload or ihdr is None or not idat or saw_iend:
                raise ValueError("invalid IEND")
            saw_iend = True
            if end != len(data):
                raise ValueError("trailing PNG data")
        pos = end

    if not saw_iend or ihdr is None or not idat:
        raise ValueError("incomplete PNG")
    if chunk_order[0] != b"IHDR" or chunk_order[-1] != b"IEND":
        raise ValueError("invalid PNG chunk order")
    if len(ihdr) != 13:
        raise ValueError("invalid IHDR length")

    w, h, bit_depth, color_type, compression, filtering, interlace = struct.unpack(">IIBBBBB", ihdr)
    if w <= 0 or h <= 0:
        raise ValueError("invalid PNG dimensions")
    if (bit_depth, color_type, compression, filtering, interlace) != (8, 6, 0, 0, 0):
        raise ValueError("unsupported PNG format")

    raw = zlib.decompress(b"".join(idat))
    stride = w * 4
    if len(raw) != h * (stride + 1):
        raise ValueError("invalid decoded PNG size")

    pixels = bytearray(w * h * 4)
    for y in range(h):
        row_start = y * (stride + 1)
        if raw[row_start] != 0:
            raise ValueError("unsupported PNG row filter")
        pixels[y * stride : (y + 1) * stride] = raw[row_start + 1 : row_start + 1 + stride]
    return w, h, bytes(pixels)


def decode_ico(data: bytes) -> tuple[tuple[int, int, bytes], ...]:
    """Validate the ICO directory and decode every embedded PNG image."""
    if len(data) < 6:
        raise ValueError("truncated ICO header")
    reserved, image_type, count = struct.unpack_from("<HHH", data, 0)
    if reserved != 0 or image_type != 1 or count != len(ICO_SIZES):
        raise ValueError("invalid ICO header")
    directory_end = 6 + count * 16
    if len(data) < directory_end:
        raise ValueError("truncated ICO directory")

    decoded = []
    previous_end = directory_end
    for index, expected_size in enumerate(ICO_SIZES):
        off = 6 + index * 16
        width, height, palette, reserved_byte, planes, bpp, size, image_offset = struct.unpack_from(
            "<BBBBHHII", data, off
        )
        encoded_size = 0 if expected_size == 256 else expected_size
        if (width, height, palette, reserved_byte, planes, bpp) != (
            encoded_size,
            encoded_size,
            0,
            0,
            1,
            32,
        ):
            raise ValueError("invalid ICO directory entry")
        if image_offset != previous_end or size <= 0 or image_offset + size > len(data):
            raise ValueError("invalid ICO image range")
        image = decode_png(data[image_offset : image_offset + size])
        if image[0] != expected_size or image[1] != expected_size:
            raise ValueError("ICO image dimensions do not match directory")
        decoded.append(image)
        previous_end = image_offset + size

    if previous_end != len(data):
        raise ValueError("trailing ICO data")
    return tuple(decoded)


def asset_matches(path: Path, generated: bytes) -> bool:
    if not path.is_file():
        return False
    actual = path.read_bytes()
    try:
        if path.suffix.lower() == ".png":
            return decode_png(actual) == decode_png(generated)
        if path.suffix.lower() == ".ico":
            return decode_ico(actual) == decode_ico(generated)
    except (ValueError, zlib.error, struct.error):
        return False
    return actual == generated


def main() -> int:
    parser = argparse.ArgumentParser(description="Generate reproducible ByFTP brand assets")
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    exp = expected()

    if args.check:
        bad = [
            str(path.relative_to(ROOT))
            for path, generated in exp.items()
            if not asset_matches(path, generated)
        ]
        if bad:
            raise SystemExit("Brand assets are out of date or invalid: " + ", ".join(bad))
        print("BRAND_ASSETS=PASS")
        return 0

    for path, generated in exp.items():
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(generated)
        print("Updated:", path.relative_to(ROOT))
    print("BRAND_ASSETS=PASS")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
