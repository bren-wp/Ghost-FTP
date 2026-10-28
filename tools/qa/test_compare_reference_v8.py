"""Regression tests for exact RGB changed-pixel accounting in v8 QA."""
from __future__ import annotations

from io import BytesIO
from pathlib import Path
from tempfile import TemporaryDirectory
import sys
import unittest

from PIL import Image

sys.path.insert(0, str(Path(__file__).resolve().parent))
from compare_reference_v8 import pixel_difference


def png_bytes(image: Image.Image) -> bytes:
    stream = BytesIO()
    image.save(stream, format="PNG")
    return stream.getvalue()


class PixelDifferenceTest(unittest.TestCase):
    def compare(self, reference: Image.Image, actual: Image.Image) -> dict:
        with TemporaryDirectory() as directory:
            path = Path(directory) / "actual.png"
            actual.save(path, format="PNG")
            return pixel_difference(png_bytes(reference), path, reference.size)

    def test_single_channel_one_unit_difference_counts_as_one_changed_pixel(self) -> None:
        baseline = Image.new("RGB", (2, 1), (0, 0, 0))
        for color in ((1, 0, 0), (0, 1, 0), (0, 0, 1)):
            with self.subTest(channel=color):
                actual = baseline.copy()
                actual.putpixel((1, 0), color)
                result = self.compare(baseline, actual)
                self.assertEqual("compared", result["status"])
                self.assertFalse(result["exact"])
                self.assertEqual(1, result["different_pixel_count"])
                self.assertEqual(0.5, result["different_pixel_fraction"])

    def test_identical_rgb_does_not_count_any_pixel(self) -> None:
        image = Image.new("RGB", (2, 3), (17, 95, 203))
        result = self.compare(image, image.copy())
        self.assertTrue(result["exact"])
        self.assertEqual(0, result["different_pixel_count"])
        self.assertEqual(0, result["different_pixel_fraction"])

    def test_alpha_only_one_unit_difference_counts_as_real_pixel_change(self) -> None:
        baseline = Image.new("RGBA", (2, 1), (1, 2, 3, 255))
        actual = baseline.copy()
        actual.putpixel((1, 0), (1, 2, 3, 254))
        result = self.compare(baseline, actual)
        self.assertEqual("compared", result["status"])
        self.assertFalse(result["exact"])
        self.assertEqual(1, result["different_pixel_count"])
        self.assertEqual(0.5, result["different_pixel_fraction"])
        self.assertEqual(0, result["mean_rgb_absolute_difference"])
        self.assertEqual(0.5, result["mean_alpha_absolute_difference"])

    def test_fully_transparent_pixel_color_change_is_not_false_exact(self) -> None:
        # RGBA byte-for-byte parity is different from composited appearance;
        # report the actual channel delta, do not claim exact screenshot data.
        baseline = Image.new("RGBA", (2, 1), (0, 0, 0, 0))
        actual = baseline.copy()
        actual.putpixel((0, 0), (1, 0, 0, 0))
        result = self.compare(baseline, actual)
        self.assertFalse(result["exact"])
        self.assertEqual(1, result["different_pixel_count"])

    def test_opaque_rgb_reference_and_equivalent_rgba_capture_are_exact(self) -> None:
        reference = Image.new("RGB", (3, 2), (17, 19, 23))
        actual = Image.new("RGBA", (3, 2), (17, 19, 23, 255))
        result = self.compare(reference, actual)
        self.assertTrue(result["exact"])
        self.assertEqual(0, result["different_pixel_count"])

    def test_capture_symlink_is_rejected_even_when_target_is_valid_png(self) -> None:
        with TemporaryDirectory() as directory:
            root = Path(directory)
            original = root / "original.png"
            original.write_bytes(png_bytes(Image.new("RGBA", (2, 1))))
            link = root / "actual.png"
            try:
                link.symlink_to(original)
            except (OSError, NotImplementedError) as error:
                self.skipTest(f"Symlinks unavailable: {error}")
            with self.assertRaisesRegex(ValueError, "must not be a symlink"):
                pixel_difference(original.read_bytes(), link, (2, 1))

    def test_capture_non_png_is_rejected_instead_of_compared_as_png(self) -> None:
        with TemporaryDirectory() as directory:
            reference = Image.new("RGB", (2, 1), (7, 8, 9))
            target = Path(directory) / "renamed.png"
            reference.save(target, format="BMP")
            with self.assertRaisesRegex(ValueError, "capture is not PNG"):
                pixel_difference(png_bytes(reference), target, (2, 1))

    def test_wrong_size_is_not_compared_or_accepted(self) -> None:
        reference = Image.new("RGB", (2, 3))
        actual = Image.new("RGB", (3, 2))
        result = self.compare(reference, actual)
        self.assertEqual("wrong_capture_size", result["status"])
        self.assertFalse(result["exact"])


if __name__ == "__main__":
    unittest.main()
