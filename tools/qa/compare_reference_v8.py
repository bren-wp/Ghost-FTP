#!/usr/bin/env python3
"""Compare all 75 Ghost FTP v8 concepts to GENUINE installed-app captures.

Pixel comparisons count changes to any RGB or alpha channel (RGBA).

Usage:
 python tools/qa/compare_reference_v8.py \
   --reference-zip "GhostFTP-Premium-Brand-UI(8).zip" \
   --captures-dir "build/qa/captures" \
   --output "build/qa/ghostftp-pixel-diff.json"

Install Pillow to measure channel-level differences: python -m pip install Pillow
Capture filenames: captures/windows/files.png, captures/android/sites.png, etc.
This NEVER promotes a design concept to real application data or marks it accepted.
"""
from __future__ import annotations
import argparse
import hashlib
import json
from io import BytesIO
from pathlib import Path
import struct
import sys
import zipfile

ARCHIVE_SHA256 = "1cdaea3e60c0ec4b14c3e8fa515351ba0710ee9022060e7f107b8bb9fbf6e877"
PREFIX = "GhostFTP-Premium-Brand-UI/"
EXPECTED = {"windows": (30, (1290, 852)), "linux": (29, (1290, 852)),
            "android": (16, (480, 960))}
MAX_UNCOMPRESSED_BYTES = 45 * 1024 * 1024


def size_of_png(data: bytes) -> tuple[int, int]:
    if len(data) < 33 or data[:8] != b"\x89PNG\r\n\x1a\n" or data[12:16] != b"IHDR":
        raise ValueError("File lacks a valid PNG header")
    width, height = struct.unpack(">II", data[16:24])
    if not 1 <= width <= 8192 or not 1 <= height <= 8192:
        raise ValueError("Unsafe reference screenshot geometry")
    return width, height


def verify(archive: Path, mapping: Path) -> list[dict]:
    # Do not read an unbounded untrusted archive into memory just to hash it.
    if archive.stat().st_size > 64 * 1024 * 1024:
        raise ValueError("Reference ZIP exceeds 64 MiB compressed safety limit")
    if hashlib.sha256(archive.read_bytes()).hexdigest() != ARCHIVE_SHA256:
        raise ValueError("Reference ZIP SHA256 mismatch: v8 is the ONLY accepted reference")
    registry = json.loads(mapping.read_text(encoding="utf-8"))
    if registry.get("referenceArchive") != "GhostFTP-Premium-Brand-UI(8).zip":
        raise ValueError("Repository screen manifest points to another reference version")
    expected = []
    for platform, (count, size) in EXPECTED.items():
        screens = registry["platforms"][platform]
        if len(screens) != count:
            raise ValueError(f"Expected {count} {platform} screens; found {len(screens)}")
        for s in screens:
            source = f"screens/{platform}/{s['id']}.png"
            if s["source"] != source:
                raise ValueError(f"Screen ID/path mismatch: {source}")
            expected.append(dict(platform=platform, screen=s["id"], path=source, dimensions=size))
    if len({x["path"] for x in expected}) != 75:
        raise ValueError("Duplicate/missing visual reference entries")
    with zipfile.ZipFile(archive) as z:
        names = {info.filename[len(PREFIX):] for info in z.infolist()
                 if info.filename.startswith(PREFIX + "screens/") and info.filename.endswith(".png")}
        if names != {x["path"] for x in expected}:
            raise ValueError("ZIP and source 75-screen manifest have different inventories")
        if sum(i.file_size for i in z.infolist()) > MAX_UNCOMPRESSED_BYTES:
            raise ValueError("Uncompressed visual archive exceeds safety limit")
        for x in expected:
            if size_of_png(z.read(PREFIX + x["path"])) != x["dimensions"]:
                raise ValueError("Bad screenshot dimensions: " + x["path"])
    return expected


def pixel_difference(reference: bytes, capture: Path, size: tuple[int, int]) -> dict:
    try:
        from PIL import Image, ImageChops, ImageStat
    except ImportError as e:
        raise RuntimeError("Install Pillow to measure PNG pixel differences") from e
    # An RGB-only comparison falsely accepts screenshots where only opacity
    # changed, including a fully invisible icon vs a visible one. Compare
    # unpremultiplied RGBA channels, not just their three RGB projections.
    if capture.is_symlink():
        raise ValueError("real installed-app capture must not be a symlink")
    with Image.open(BytesIO(reference)) as opened:
        if opened.format != "PNG":
            raise ValueError("reference is not PNG")
        expected = opened.convert("RGBA")
    with Image.open(capture) as opened:
        if opened.format != "PNG":
            raise ValueError("capture is not PNG")
        actual = opened.convert("RGBA")
    if expected.size != size or actual.size != size:
        return dict(status="wrong_capture_size", capture_size=actual.size, exact=False)
    diff = ImageChops.difference(expected, actual)
    stats = ImageStat.Stat(diff)
    pixels = size[0] * size[1]
    # One changed channel (including alpha) must change the exact-pixel count.
    # Grayscale conversion rounds one-unit RGB/alpha deltas down to zero.
    red, green, blue, alpha = diff.split()
    changed_mask = ImageChops.lighter(
        ImageChops.lighter(red, green), ImageChops.lighter(blue, alpha)
    )
    changed = pixels - changed_mask.histogram()[0]
    return dict(status="compared", exact=changed == 0,
                mean_rgb_absolute_difference=round(sum(stats.mean[:3]) / 3, 4),
                mean_alpha_absolute_difference=round(stats.mean[3], 4),
                different_pixel_count=changed,
                different_pixel_fraction=round(changed / pixels, 6),
                capture_sha256=hashlib.sha256(capture.read_bytes()).hexdigest())


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--reference-zip", type=Path, required=True)
    parser.add_argument("--contract", type=Path, default=Path("docs/qa/premium-75-screen-contract.json"))
    parser.add_argument("--captures-dir", type=Path)
    parser.add_argument("--output", type=Path, default=Path("build/qa/ghostftp-pixel-diff.json"))
    parser.add_argument("--require-all-captures", action="store_true")
    parser.add_argument("--require-exact", action="store_true",
                        help="Fail if any real capture is missing, invalid, resized or differs by any RGBA channel")
    args = parser.parse_args()
    try:
        screens = verify(args.reference_zip, args.contract)
        records = []
        with zipfile.ZipFile(args.reference_zip) as z:
            for screen in screens:
                path = screen["path"]
                capture = (args.captures_dir / screen["platform"] / (screen["screen"] + ".png")) if args.captures_dir else None
                result = dict(platform=screen["platform"], screen=screen["screen"],
                              reference=path, reference_size=list(screen["dimensions"]),
                              visual_acceptance="pending")
                if capture is None or not capture.is_file():
                    result["status"] = "missing_real_capture"
                else:
                    result["capture_path"] = str(capture)
                    try:
                        result.update(pixel_difference(z.read(PREFIX + path), capture,
                                                       screen["dimensions"]))
                    except (OSError, ValueError, RuntimeError) as error:
                        # One broken screenshot must not hide the other 74
                        # comparisons. Preserve a per-screen failure record.
                        result["status"] = "invalid_capture"
                        result["error"] = str(error)
                records.append(result)
    except (KeyError, OSError, ValueError, RuntimeError, zipfile.BadZipFile) as error:
        print("Ghost FTP visual QA: " + str(error), file=sys.stderr)
        return 2

    counts = {p: dict(references=n,
                      compared=sum(x["platform"] == p and x["status"] == "compared" for x in records),
                      missing=sum(x["platform"] == p and x["status"] != "compared" for x in records))
              for p, (n, _) in EXPECTED.items()}
    report = dict(archive_sha256=ARCHIVE_SHA256,
                  reference_type="Illustrative concept, NOT real app data",
                  human_accepted_screens=0, counts=counts, screens=records,
                  acceptance="Only approve after installed screenshot alignment, legitimate state and functional QA")
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(report, indent=2, ensure_ascii=False) + "\n",
                           encoding="utf-8")
    print("Reviewed 75 source images; compared",
          sum(x["status"] == "compared" for x in records), "real captures ->", args.output)
    if args.require_exact:
        return int(any(x["status"] != "compared" or x.get("exact") is not True
                       for x in records))
    return int(args.require_all_captures and any(x["status"] != "compared" for x in records))


if __name__ == "__main__":
    raise SystemExit(main())
