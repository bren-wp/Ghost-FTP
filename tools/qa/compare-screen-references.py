#!/usr/bin/env python3
"""Compare authentic Ghost FTP application captures to the supplied 75 PNG concepts.

Never treat illustrative data/server names as a runtime requirement. A pixel
comparison is meaningful only after OS, scale, viewport, app state and
redaction rules are matched. This tool DOES NOT certify functionality/security.
Requires Pillow. No external network access, fake screenshots or OCR.
"""
from __future__ import annotations

import argparse
import csv
import io
import json
from pathlib import Path
import zipfile

from PIL import Image, ImageChops, ImageEnhance

SCREEN_SETS = {
    "windows": (
        "brand", "files", "sites", "new-connection", "quick-connect",
        "transfers", "failed-transfers", "transfer-schedule", "sync",
        "directory-diff", "duplicates", "disk-usage", "settings",
        "settings-transfer", "settings-security", "settings-appearance",
        "terminal", "command-palette", "snippets", "agent", "import",
        "host-key", "overwrite", "properties", "permissions", "shortcuts",
        "help", "updates", "search", "delete-confirm",
    ),
    "linux": (
        "files", "sites", "new-connection", "quick-connect", "transfers",
        "failed-transfers", "transfer-schedule", "sync", "directory-diff",
        "duplicates", "disk-usage", "settings", "settings-transfer",
        "settings-security", "settings-appearance", "terminal",
        "command-palette", "snippets", "agent", "import", "host-key",
        "overwrite", "properties", "permissions", "shortcuts", "help",
        "updates", "search", "delete-confirm",
    ),
    "android": (
        "files", "sites", "connect", "transfers", "sync", "settings",
        "about", "new-folder", "rename", "delete", "upload", "download",
        "host-key", "activity", "disconnected", "transfer-detail",
    ),
}
SIZES = {"windows": (1290, 852), "linux": (1290, 852), "android": (480, 960)}
ZIP_ROOT = "GhostFTP-Premium-Brand-UI/screens/"


def image_is_actual_png(blob: bytes, name: str) -> Image.Image:
    if not blob.startswith(b"\\x89PNG\\r\\n\\x1a\\n"):
        raise ValueError(f"{name}: not an authentic PNG file")
    with Image.open(io.BytesIO(blob)) as opened:
        opened.verify()
    image = Image.open(io.BytesIO(blob))
    return image.convert("RGB")


def compare(reference: Image.Image, actual: Image.Image, output: Path) -> dict:
    delta = ImageChops.difference(reference, actual)
    histogram = delta.histogram()
    channel_sum = sum((index % 256) * count for index, count in enumerate(histogram))
    pixel_count = reference.width * reference.height
    mae = channel_sum / (pixel_count * 3)
    changed = sum(
        1 for pixel in delta.getdata()
        if max(pixel) > 12
    )
    bounds = delta.getbbox()
    output.mkdir(parents=True, exist_ok=True)
    ImageEnhance.Contrast(delta).enhance(3).save(output / "diff.png")
    reference.save(output / "reference.png")
    # Never alter or distribute original captured files with credentials.
    # Save only the enhanced delta and the non-secret concept reference.
    return {
        "mean_abs_error_0_255": round(mae, 4),
        "changed_pixels_over_12": changed,
        "changed_ratio": round(changed / pixel_count, 6),
        "diff_bounds": bounds,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--references", required=True, type=Path,
                        help="Original GhostFTP-Premium-Brand-UI ZIP as supplied")
    parser.add_argument("--captures", required=True, type=Path,
                        help="Real runtime captures at <platform>/<screen>.png")
    parser.add_argument("--output", required=True, type=Path,
                        help="Separate QA report directory; may contain screenshot differences")
    parser.add_argument("--platform", choices=("windows", "linux", "android", "all"),
                        default="all")
    args = parser.parse_args()
    if not args.references.is_file() or not args.captures.is_dir():
        parser.error("Reference ZIP and actual-capture directory must exist")
    if args.output.resolve() == args.captures.resolve():
        parser.error("Reports cannot overwrite the real screenshots")
    platforms = SCREEN_SETS.keys() if args.platform == "all" else [args.platform]
    reports = []
    args.output.mkdir(parents=True, exist_ok=True)
    with zipfile.ZipFile(args.references) as bundle:
        for platform in platforms:
            for name in SCREEN_SETS[platform]:
                relative = f"{platform}/{name}.png"
                entry = f"{ZIP_ROOT}{relative}"
                target = args.captures / relative
                row = {"platform": platform, "screen": name,
                       "reference": entry, "capture": relative,
                       "status": "missing", "reference_size": list(SIZES[platform])}
                reference = image_is_actual_png(bundle.read(entry), entry)
                if reference.size != SIZES[platform]:
                    raise ValueError(f"Unexpected reference size: {entry} {reference.size}")
                if not target.is_file():
                    reports.append(row)
                    continue
                try:
                    actual = image_is_actual_png(target.read_bytes(), str(target))
                    row["actual_size"] = list(actual.size)
                    if actual.size != reference.size:
                        row["status"] = "wrong_size"
                    else:
                        row.update(compare(reference, actual, args.output / platform / name))
                        row["status"] = "compared"
                except Exception as exc:
                    row["status"] = "invalid_capture"
                    row["reason"] = str(exc)
                reports.append(row)
    (args.output / "results.json").write_text(json.dumps({
        "reference_kind": "illustrative design concepts, NOT actual app evidence",
        "pixel_exactness_certified": False,
        "criteria": "Fixed viewport, DPI, language, theme and real UI state required",
        "screens": reports,
    }, indent=2, ensure_ascii=False) + "\n", encoding="utf8")
    with (args.output / "results.csv").open("w", newline="", encoding="utf8") as output:
        writer = csv.DictWriter(output, fieldnames=[
            "platform", "screen", "status", "mean_abs_error_0_255", "changed_ratio",
            "changed_pixels_over_12", "reference_size", "actual_size", "diff_bounds",
            "reason",
        ], extrasaction="ignore")
        writer.writeheader()
        for row in reports:
            writer.writerow(row)
    coverage = sum(x["status"] == "compared" for x in reports)
    missing = sum(x["status"] == "missing" for x in reports)
    print(f"Ghost FTP visual audit: {coverage}/{len(reports)} comparable screenshots; "
          f"{missing} missing. Report: {args.output / 'results.json'}")
    # Incomplete/invalid evidence is never a passing visual-acceptance gate.
    return 0 if coverage == len(reports) else 2


if __name__ == "__main__":
    raise SystemExit(main())
