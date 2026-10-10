#!/usr/bin/env python3
"""Fail-closed Linux WebKit screenshot QA: a decorated black window is NOT UI."""
import sys
from pathlib import Path

from PIL import Image, ImageFile

ImageFile.LOAD_TRUNCATED_IMAGES = False


def validate(filename: str) -> None:
    source = Path(filename)
    if not source.is_file() or source.stat().st_size < 4096:
        raise ValueError("Native screenshot is absent or suspiciously small.")
    with Image.open(source) as image:
        image.load()
        if image.format != "PNG" or image.size != (1290, 852):
            raise ValueError(f"Unexpected PNG format or reference size: {image.format} {image.size}")
        # Exclude window manager decorations; analyse only app content.
        content = image.convert("RGB").crop((75, 65, 1215, 785))
        pixels = list(content.resize((190, 120), Image.NEAREST).getdata())

    colors = {(r // 16, g // 16, b // 16) for r, g, b in pixels}
    # A blank WebKit surface can still produce >18 KiB PNG due to Openbox
    # gradients/title text. Require actual structure INSIDE the app rectangle.
    illuminated = sum(max(r, g, b) >= 42 for r, g, b in pixels)
    active_fraction = illuminated / len(pixels)
    edge_count = 0
    width = 190
    for pos, p in enumerate(pixels):
        x = pos % width
        if x < width - 1:
            q = pixels[pos + 1]
            if max(abs(p[k] - q[k]) for k in range(3)) >= 35:
                edge_count += 1
    edge_ratio = edge_count / (len(pixels) - 120)
    print(
        f"Linux app image QA: quantized colors={len(colors)}, "
        f"bright coverage={active_fraction:.4f}, horizontal edges={edge_ratio:.4f}"
    )
    if len(colors) < 10 or active_fraction < 0.015 or edge_ratio < 0.002:
        raise ValueError(
            "The X11 window was found, but the WebKit application content is "
            "blank/black or structureless. Refusing to publish it as visual proof."
        )


if __name__ == "__main__":
    try:
        if len(sys.argv) != 2:
            raise ValueError("Usage: check-linux-premium-png.py <genuine screenshot>")
        validate(sys.argv[1])
    except (OSError, ValueError) as error:
        print(f"Linux visual QA FAILED: {error}", file=sys.stderr)
        sys.exit(1)
