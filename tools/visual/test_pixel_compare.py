#!/usr/bin/env python3
import tempfile
import unittest
from pathlib import Path

from PIL import Image
from pixel_compare import compare


class NativePixelComparisonTests(unittest.TestCase):
    def test_real_pixel_differences_are_not_hidden(self):
        with tempfile.TemporaryDirectory() as temp:
            a = Path(temp) / "reference.png"
            b = Path(temp) / "native.png"
            Image.new("RGB", (4, 3), (4, 10, 22)).save(a)
            Image.new("RGB", (4, 3), (4, 10, 22)).save(b)
            result = compare(a, b)
            self.assertEqual(result["different_pixels"], 0)
            self.assertTrue(result["pixel_exact"])
            changed = Image.open(b).copy()
            changed.putpixel((0, 0), (250, 10, 22))
            changed.save(b)
            result = compare(a, b)
            self.assertEqual(result["different_pixels"], 1)
            self.assertFalse(result["pixel_exact"])
            self.assertAlmostEqual(result["different_pixels_percent"], 100 / 12, places=5)

    def test_size_mismatch_is_rejected(self):
        with tempfile.TemporaryDirectory() as temp:
            a = Path(temp) / "reference.png"
            b = Path(temp) / "native.png"
            Image.new("RGB", (4, 3)).save(a)
            Image.new("RGB", (5, 3)).save(b)
            with self.assertRaises(ValueError):
                compare(a, b)


if __name__ == "__main__":
    unittest.main()
