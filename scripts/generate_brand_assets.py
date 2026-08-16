#!/usr/bin/env python3
"""Generira i provjerava službene ByFTP slikovne resurse bez vanjskih ovisnosti."""

from __future__ import annotations

import argparse
import binascii
import io
import struct
import tempfile
import zlib
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
ICON_PNG = ROOT / "build" / "icon.png"
ICON_ICO = ROOT / "build" / "icon.ico"
HEADER_PNG = ROOT / "docs" / "slike" / "byftp-zaglavlje.png"

# Boje su namjerno definirane ovdje kako bi se svi službeni resursi generirali
# iz jednog, reproducibilnog izvora.
BG_TOP = (10, 18, 32, 255)
BG_BOTTOM = (19, 32, 52, 255)
PANEL = (23, 39, 62, 255)
ACCENT = (66, 210, 196, 255)
ACCENT_2 = (90, 142, 255, 255)
TEXT = (238, 245, 255, 255)
MUTED = (157, 174, 196, 255)

FONT_5X7 = {
    "A": ["01110", "10001", "10001", "11111", "10001", "10001", "10001"],
    "B": ["11110", "10001", "10001", "11110", "10001", "10001", "11110"],
    "D": ["11110", "10001", "10001", "10001", "10001", "10001", "11110"],
    "E": ["11111", "10000", "10000", "11110", "10000", "10000", "11111"],
    "F": ["11111", "10000", "10000", "11110", "10000", "10000", "10000"],
    "G": ["01111", "10000", "10000", "10111", "10001", "10001", "01111"],
    "I": ["11111", "00100", "00100", "00100", "00100", "00100", "11111"],
    "J": ["00111", "00010", "00010", "00010", "10010", "10010", "01100"],
    "K": ["10001", "10010", "10100", "11000", "10100", "10010", "10001"],
    "L": ["10000", "10000", "10000", "10000", "10000", "10000", "11111"],
    "M": ["10001", "11011", "10101", "10101", "10001", "10001", "10001"],
    "N": ["10001", "11001", "10101", "10011", "10001", "10001", "10001"],
    "O": ["01110", "10001", "10001", "10001", "10001", "10001", "01110"],
    "P": ["11110", "10001", "10001", "11110", "10000", "10000", "10000"],
    "R": ["11110", "10001", "10001", "11110", "10100", "10010", "10001"],
    "S": ["01111", "10000", "10000", "01110", "00001", "00001", "11110"],
    "T": ["11111", "00100", "00100", "00100", "00100", "00100", "00100"],
    "U": ["10001", "10001", "10001", "10001", "10001", "10001", "01110"],
    "V": ["10001", "10001", "10001", "10001", "10001", "01010", "00100"],
    "Y": ["10001", "10001", "01010", "00100", "00100", "00100", "00100"],
    " ": ["00000"] * 7,
}


def canvas(width: int, height: int) -> bytearray:
    px = bytearray(width * height * 4)
    for y in range(height):
        t = y / max(1, height - 1)
        c = tuple(round(BG_TOP[i] * (1 - t) + BG_BOTTOM[i] * t) for i in range(4))
        row = bytes(c) * width
        px[y * width * 4 : (y + 1) * width * 4] = row
    return px


def put(px: bytearray, w: int, h: int, x: int, y: int, color: tuple[int, int, int, int]) -> None:
    if 0 <= x < w and 0 <= y < h:
        i = (y * w + x) * 4
        px[i : i + 4] = bytes(color)


def rect(px: bytearray, w: int, h: int, x0: int, y0: int, x1: int, y1: int, color: tuple[int, int, int, int]) -> None:
    x0, x1 = max(0, min(x0, x1)), min(w, max(x0, x1))
    y0, y1 = max(0, min(y0, y1)), min(h, max(y0, y1))
    row = bytes(color) * max(0, x1 - x0)
    for y in range(y0, y1):
        start = (y * w + x0) * 4
        px[start : start + len(row)] = row


def rounded_rect(px: bytearray, w: int, h: int, x0: int, y0: int, x1: int, y1: int, radius: int, color: tuple[int, int, int, int]) -> None:
    r = max(1, radius)
    for y in range(y0, y1):
        for x in range(x0, x1):
            dx = 0
            dy = 0
            if x < x0 + r:
                dx = x0 + r - x
            elif x >= x1 - r:
                dx = x - (x1 - r - 1)
            if y < y0 + r:
                dy = y0 + r - y
            elif y >= y1 - r:
                dy = y - (y1 - r - 1)
            if dx * dx + dy * dy <= r * r:
                put(px, w, h, x, y, color)


def line(px: bytearray, w: int, h: int, x0: int, y0: int, x1: int, y1: int, thickness: int, color: tuple[int, int, int, int]) -> None:
    dx = abs(x1 - x0)
    sx = 1 if x0 < x1 else -1
    dy = -abs(y1 - y0)
    sy = 1 if y0 < y1 else -1
    err = dx + dy
    while True:
        half = max(0, thickness // 2)
        rect(px, w, h, x0 - half, y0 - half, x0 + half + 1, y0 + half + 1, color)
        if x0 == x1 and y0 == y1:
            break
        e2 = 2 * err
        if e2 >= dy:
            err += dy
            x0 += sx
        if e2 <= dx:
            err += dx
            y0 += sy


def draw_arrow(px: bytearray, w: int, h: int, x0: int, y: int, x1: int, thickness: int, color: tuple[int, int, int, int]) -> None:
    line(px, w, h, x0, y, x1, y, thickness, color)
    d = max(4, abs(x1 - x0) // 5)
    if x1 > x0:
        line(px, w, h, x1, y, x1 - d, y - d, thickness, color)
        line(px, w, h, x1, y, x1 - d, y + d, thickness, color)
    else:
        line(px, w, h, x1, y, x1 + d, y - d, thickness, color)
        line(px, w, h, x1, y, x1 + d, y + d, thickness, color)


def draw_icon(size: int) -> tuple[int, int, bytearray]:
    px = canvas(size, size)
    margin = max(2, size // 16)
    rounded_rect(px, size, size, margin, margin, size - margin, size - margin, max(3, size // 7), PANEL)

    # Dva panela predstavljaju lokalno računalo i poslužitelj.
    pane_w = max(5, size // 4)
    pane_h = max(8, size // 2)
    y0 = (size - pane_h) // 2
    left_x = size // 8
    right_x = size - size // 8 - pane_w
    rounded_rect(px, size, size, left_x, y0, left_x + pane_w, y0 + pane_h, max(2, size // 32), TEXT)
    rounded_rect(px, size, size, right_x, y0, right_x + pane_w, y0 + pane_h, max(2, size // 32), TEXT)

    # Diskretne "datoteke" unutar panela.
    bar_h = max(1, size // 40)
    for yy in (y0 + pane_h // 4, y0 + pane_h // 2, y0 + 3 * pane_h // 4):
        rect(px, size, size, left_x + pane_w // 5, yy, left_x + 4 * pane_w // 5, yy + bar_h, BG_BOTTOM)
        rect(px, size, size, right_x + pane_w // 5, yy, right_x + 4 * pane_w // 5, yy + bar_h, BG_BOTTOM)

    mid0 = left_x + pane_w + max(1, size // 20)
    mid1 = right_x - max(1, size // 20)
    draw_arrow(px, size, size, mid0, size // 2 - size // 10, mid1, max(2, size // 38), ACCENT)
    draw_arrow(px, size, size, mid1, size // 2 + size // 10, mid0, max(2, size // 38), ACCENT_2)
    return size, size, px


def draw_text(px: bytearray, w: int, h: int, text: str, x: int, y: int, scale: int, color: tuple[int, int, int, int]) -> None:
    cursor = x
    for ch in text.upper():
        glyph = FONT_5X7.get(ch, FONT_5X7[" "])
        for gy, row in enumerate(glyph):
            for gx, bit in enumerate(row):
                if bit == "1":
                    rect(px, w, h, cursor + gx * scale, y + gy * scale, cursor + (gx + 1) * scale, y + (gy + 1) * scale, color)
        cursor += 6 * scale


def draw_header() -> tuple[int, int, bytearray]:
    w, h = 1200, 320
    px = canvas(w, h)
    # Brand kartica s ikonom na lijevoj strani.
    rounded_rect(px, w, h, 40, 40, 280, 280, 42, PANEL)
    iw, ih, icon = draw_icon(200)
    # Kopiranje ikone bez alpha blendinga jer je već neprozirna.
    for y in range(ih):
        src = y * iw * 4
        dst = ((60 + y) * w + 60) * 4
        px[dst : dst + iw * 4] = icon[src : src + iw * 4]
    draw_text(px, w, h, "BYFTP", 335, 72, 17, TEXT)
    draw_text(px, w, h, "SIGURAN PRIJENOS DATOTEKA", 338, 215, 5, MUTED)
    rect(px, w, h, 338, 270, 1095, 276, ACCENT)
    return w, h, px


def png_bytes(width: int, height: int, rgba: bytes) -> bytes:
    def chunk(kind: bytes, data: bytes) -> bytes:
        body = kind + data
        return struct.pack(">I", len(data)) + body + struct.pack(">I", binascii.crc32(body) & 0xFFFFFFFF)

    scan = bytearray()
    stride = width * 4
    for y in range(height):
        scan.append(0)
        scan.extend(rgba[y * stride : (y + 1) * stride])
    return b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", struct.pack(">IIBBBBB", width, height, 8, 6, 0, 0, 0)) + chunk(b"IDAT", zlib.compress(bytes(scan), 9)) + chunk(b"IEND", b"")

def icon_png(size: int) -> bytes:
    w, h, px = draw_icon(size)
    return png_bytes(w, h, bytes(px))


def ico_bytes() -> bytes:
    sizes = [16, 24, 32, 48, 64, 128, 256]
    images = [icon_png(s) for s in sizes]
    header = struct.pack("<HHH", 0, 1, len(images))
    offset = 6 + 16 * len(images)
    entries = bytearray()
    for size, data in zip(sizes, images):
        dim = 0 if size == 256 else size
        entries.extend(struct.pack("<BBBBHHII", dim, dim, 0, 0, 1, 32, len(data), offset))
        offset += len(data)
    return header + bytes(entries) + b"".join(images)


def header_png() -> bytes:
    w, h, px = draw_header()
    return png_bytes(w, h, bytes(px))


def expected_assets() -> dict[Path, bytes]:
    return {
        ICON_PNG: icon_png(512),
        ICON_ICO: ico_bytes(),
        HEADER_PNG: header_png(),
    }


def parse_png(data: bytes) -> tuple[bytes, bytes]:
    if not data.startswith(b"\x89PNG\r\n\x1a\n"):
        raise ValueError("PNG potpis nije ispravan")
    pos = 8
    saw_iend = False
    ihdr = b""
    idat = bytearray()
    while pos + 12 <= len(data):
        length = struct.unpack(">I", data[pos : pos + 4])[0]
        end = pos + 12 + length
        if end > len(data):
            raise ValueError(f"PNG chunk prelazi kraj datoteke (pozicija={pos}, duljina={length}, ukupno={len(data)})")
        kind = data[pos + 4 : pos + 8]
        payload = data[pos + 8 : pos + 8 + length]
        crc = struct.unpack(">I", data[pos + 8 + length : end])[0]
        if (binascii.crc32(kind + payload) & 0xFFFFFFFF) != crc:
            raise ValueError(f"PNG CRC nije ispravan za {kind!r}")
        if kind == b"IHDR":
            ihdr = payload
        elif kind == b"IDAT":
            idat.extend(payload)
        elif kind == b"IEND":
            saw_iend = True
            if end != len(data):
                raise ValueError("PNG sadrži podatke nakon IEND chunka")
            break
        pos = end
    if not saw_iend or not ihdr or not idat:
        raise ValueError("PNG nema obavezne IHDR/IDAT/IEND chunkove")
    try:
        pixels = zlib.decompress(bytes(idat))
    except zlib.error as exc:
        raise ValueError(f"PNG IDAT nije moguće dekomprimirati: {exc}") from exc
    return ihdr, pixels


def verify_png(data: bytes) -> None:
    parse_png(data)


def raw_png_semantic(width: int, height: int, rgba: bytes) -> tuple[bytes, bytes]:
    ihdr = struct.pack(">IIBBBBB", width, height, 8, 6, 0, 0, 0)
    scan = bytearray()
    stride = width * 4
    for y in range(height):
        scan.append(0)
        scan.extend(rgba[y * stride : (y + 1) * stride])
    return ihdr, bytes(scan)


def expected_png_semantic(path: Path) -> tuple[bytes, bytes]:
    if path == ICON_PNG:
        w, h, px = draw_icon(512)
        return raw_png_semantic(w, h, bytes(px))
    if path == HEADER_PNG:
        w, h, px = draw_header()
        return raw_png_semantic(w, h, bytes(px))
    raise ValueError(f"nepoznat PNG resurs: {path}")


def parse_ico_semantic(data: bytes) -> list[tuple[tuple[int, ...], tuple[bytes, bytes]]]:
    if len(data) < 6:
        raise ValueError("ICO je prekratak")
    reserved, kind, count = struct.unpack_from("<HHH", data, 0)
    if reserved != 0 or kind != 1 or count < 1 or count > 64:
        raise ValueError("ICO zaglavlje nije ispravno")
    result = []
    for i in range(count):
        off = 6 + i * 16
        if off + 16 > len(data):
            raise ValueError("ICO tablica je skraćena")
        width, height, colors, reserved2, planes, bpp, size, image_off = struct.unpack_from("<BBBBHHII", data, off)
        end = image_off + size
        if end > len(data):
            raise ValueError("ICO zapis izlazi iz datoteke")
        image = data[image_off:end]
        if not image.startswith(b"\x89PNG\r\n\x1a\n"):
            raise ValueError("ByFTP ICO očekuje PNG-backed slike")
        meta = (width, height, colors, reserved2, planes, bpp)
        result.append((meta, parse_png(image)))
    return result


def expected_ico_semantic() -> list[tuple[tuple[int, ...], tuple[bytes, bytes]]]:
    result = []
    for size in [16, 24, 32, 48, 64, 128, 256]:
        dim = 0 if size == 256 else size
        w, h, px = draw_icon(size)
        result.append(((dim, dim, 0, 0, 1, 32), raw_png_semantic(w, h, bytes(px))))
    return result


def assets_equivalent(path: Path, current: bytes) -> bool:
    # Uspoređujemo stvarni sadržaj/piksele, a ne zlib kompresijski layout koji
    # može varirati između verzija Pythona/zliba. Očekivani pikseli nastaju
    # izravno iz renderera, bez ponovnog parsiranja tek generiranog PNG-a.
    if path.suffix.lower() == ".png":
        return parse_png(current) == expected_png_semantic(path)
    if path.suffix.lower() == ".ico":
        return parse_ico_semantic(current) == expected_ico_semantic()
    return current == expected_assets()[path]

def write_assets() -> None:
    for path, data in expected_assets().items():
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(data)
        if path.suffix.lower() == ".png":
            verify_png(data)
        print(f"Ažurirano: {path.relative_to(ROOT)}")


def check_assets() -> None:
    for path, data in expected_assets().items():
        if not path.is_file():
            raise SystemExit(f"Nedostaje slikovni resurs: {path.relative_to(ROOT)}")
        current = path.read_bytes()
        try:
            equivalent = assets_equivalent(path, current)
        except ValueError as exc:
            raise SystemExit(f"Slikovni resurs nije valjan ({path.relative_to(ROOT)}): {exc}") from exc
        if not equivalent:
            raise SystemExit(f"Slikovni resurs nije sadržajno sinkroniziran: {path.relative_to(ROOT)}")
    print("SLIKOVNI_RESURSI=ISPRAVNI")


def main() -> int:
    parser = argparse.ArgumentParser(description="Generiranje i provjera ByFTP slikovnih resursa")
    parser.add_argument("--check", action="store_true", help="samo provjeri jesu li generirani resursi aktualni i ispravni")
    args = parser.parse_args()
    if args.check:
        check_assets()
    else:
        write_assets()
        check_assets()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
