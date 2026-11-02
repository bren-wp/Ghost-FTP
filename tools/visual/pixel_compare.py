#!/usr/bin/env python3
"""Measure every actual pixel. Never treat a visual concept as native capture."""
import argparse
import hashlib
import json
import sys
from pathlib import Path

from PIL import Image, ImageChops, ImageStat


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def compare(reference, actual):
    with Image.open(reference) as source, Image.open(actual) as rendered:
        expected = source.convert("RGBA")
        observed = rendered.convert("RGBA")
        if expected.size != observed.size:
            raise ValueError(
                f"Native render size {observed.size} does not match reference {expected.size}"
            )
        width, height = expected.size
        diff = ImageChops.difference(expected, observed)
        different = sum(pixel != (0, 0, 0, 0) for pixel in diff.getdata())
        mean_rgb = sum(ImageStat.Stat(diff.convert("RGB")).mean) / 3
        return {
            "reference": str(reference),
            "actual": str(actual),
            "reference_sha256": digest(reference),
            "actual_sha256": digest(actual),
            "width": width,
            "height": height,
            "total_pixels": width * height,
            "different_pixels": different,
            "different_pixels_percent": round(100 * different / (width * height), 6),
            "mean_absolute_rgb_channel_difference": round(mean_rgb, 6),
            "pixel_exact": different == 0,
            "note": (
                "Raw pixel differences include authentic file lists, locale, "
                "fonts and runtime state; this is not a functional failure percentage."
            ),
        }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--reference", type=Path, required=True)
    parser.add_argument("--actual", type=Path, required=True)
    parser.add_argument("--json-output", type=Path)
    parser.add_argument(
        "--require-exact", action="store_true",
        help="Fail if any pixel differs; never enabled for unapproved concept renders",
    )
    args = parser.parse_args()
    try:
        results = compare(args.reference, args.actual)
        report = json.dumps(results, ensure_ascii=False, indent=2) + "\n"
        if args.json_output:
            args.json_output.parent.mkdir(parents=True, exist_ok=True)
            args.json_output.write_text(report, encoding="utf-8")
        else:
            print(report, end="")
        if args.require_exact and not results["pixel_exact"]:
            print("Pixel parity unverified: native image differs.", file=sys.stderr)
            return 1
        return 0
    except (OSError, ValueError) as error:
        print(f"Pixel comparison refused: {error}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
