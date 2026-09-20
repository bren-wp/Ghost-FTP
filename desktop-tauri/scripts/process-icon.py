"""Crop the lighthouse source down to its rounded-square art, dropping the
surrounding black border and replacing the corners outside the rounded
shape with transparency. Output goes to src-tauri/icons/source.png at
1024x1024; `npm run tauri icon` then fans it out to every platform size."""

import sys
from pathlib import Path
from PIL import Image, ImageDraw, ImageFilter


def crop_navy(src: Image.Image) -> Image.Image:
    """Find the bounding box of non-pure-black pixels and crop to it."""
    px = src.load()
    w, h = src.size

    def navy(x, y):
        p = px[x, y]
        r, g, b = p[0], p[1], p[2]
        # Pure black border is exactly (0,0,0). The navy interior is roughly
        # (18, 23, 38). Sum > 30 cleanly separates them.
        return (r + g + b) > 30

    def find_first(cols):
        for c in cols:
            for r in range(h):
                if navy(c, r):
                    return c
        return 0

    def find_first_row(rows):
        for r in rows:
            for c in range(w):
                if navy(c, r):
                    return r
        return 0

    left = find_first(range(w))
    right = find_first(range(w - 1, -1, -1))
    top = find_first_row(range(h))
    bottom = find_first_row(range(h - 1, -1, -1))
    return src.crop((left, top, right + 1, bottom + 1))


def apply_rounded_alpha(img: Image.Image, radius_ratio: float = 0.22) -> Image.Image:
    """Make pixels outside the rounded-square mask transparent."""
    size_w, size_h = img.size
    mask = Image.new("L", (size_w, size_h), 0)
    draw = ImageDraw.Draw(mask)
    radius = int(min(size_w, size_h) * radius_ratio)
    draw.rounded_rectangle(
        (0, 0, size_w - 1, size_h - 1), radius=radius, fill=255
    )
    # Slight blur smooths the rounded edge so it doesn't look stairstepped
    # when downscaled to 16x16 / 32x32.
    mask = mask.filter(ImageFilter.GaussianBlur(0.5))
    img = img.convert("RGBA")
    img.putalpha(mask)
    return img


def main():
    src_path = Path(sys.argv[1])
    dest_path = Path(sys.argv[2])
    target_size = int(sys.argv[3]) if len(sys.argv) > 3 else 1024

    src = Image.open(src_path).convert("RGBA")
    cropped = crop_navy(src)
    resized = cropped.resize((target_size, target_size), Image.LANCZOS)
    final = apply_rounded_alpha(resized)
    dest_path.parent.mkdir(parents=True, exist_ok=True)
    final.save(dest_path, "PNG")
    print(f"wrote {dest_path} ({target_size}x{target_size})")


if __name__ == "__main__":
    main()
