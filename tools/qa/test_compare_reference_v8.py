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

    def test_wrong_size_is_not_compared_or_accepted(self) -> None:
        reference = Image.new("RGB", (2, 3))
        actual = Image.new("RGB", (3, 2))
        result = self.compare(reference, actual)
        self.assertEqual("wrong_capture_size", result["status"])
        self.assertFalse(result["exact"])


if __name__ == "__main__":
    unittest.main()
