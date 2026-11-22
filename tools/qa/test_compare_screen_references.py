#!/usr/bin/env python3
"""Fail-closed tests for the 59-screen visual comparator (tiny synthetic fixtures)."""
from __future__ import annotations

import importlib.util
import io
import json
import sys
import tempfile
import unittest
import zipfile
from pathlib import Path
from unittest.mock import patch

from PIL import Image

SCRIPT = Path(__file__).with_name("compare-screen-references.py")
SPEC = importlib.util.spec_from_file_location("ghostftp_qa_compare", SCRIPT)
assert SPEC is not None and SPEC.loader is not None
comparator = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(comparator)


class VisualAcceptanceSafetyTests(unittest.TestCase):
    def run_comparison(self, *, modify=False, capture=True, require_exact=True):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            references = root / "reference.zip"
            captures = root / "captures"
            output = root / "report"
            (captures / "windows").mkdir(parents=True)
            image = Image.new("RGB", (5, 4), (9, 20, 32))
            raw = io.BytesIO()
            image.save(raw, format="PNG")
            with zipfile.ZipFile(references, "w") as bundle:
                bundle.writestr(
                    comparator.ZIP_ROOT + "windows/files.png", raw.getvalue()
                )
            if capture:
                if modify:
                    image.putpixel((0, 0), (10, 20, 32))  # just one channel +1
                image.save(captures / "windows" / "files.png")
            arguments = [
                str(SCRIPT), "--references", str(references),
                "--captures", str(captures), "--output", str(output),
                "--platform", "windows",
            ]
            if require_exact:
                arguments.append("--require-exact")
            # Test the actual CLI/main path with a one-screen tiny fixture.
            with (
                patch.object(comparator, "SCREEN_SETS", {"windows": ("files",)}),
                patch.object(comparator, "SIZES", {"windows": (5, 4)}),
                patch.object(sys, "argv", arguments),
            ):
                code = comparator.main()
            report = json.loads((output / "results.json").read_text(encoding="utf8"))
            return code, report

    def test_exact_pixels_pass_only_when_every_channel_matches(self):
        code, report = self.run_comparison()
        self.assertEqual(code, 0)
        self.assertTrue(report["screens"][0]["pixel_exact"])
        self.assertEqual(report["screens"][0]["different_pixels_exact"], 0)
        # A concept reference is NOT sufficient to claim true native QA.
        self.assertFalse(report["pixel_exactness_certified"])

    def test_single_channel_delta_below_review_threshold_fails_strict_mode(self):
        code, report = self.run_comparison(modify=True)
        self.assertEqual(code, 3)
        entry = report["screens"][0]
        self.assertEqual(entry["changed_pixels_over_12"], 0)
        self.assertEqual(entry["different_pixels_exact"], 1)
        self.assertFalse(entry["pixel_exact"])

    def test_missing_native_capture_never_passes(self):
        code, report = self.run_comparison(capture=False)
        self.assertEqual(code, 2)
        self.assertEqual(report["screens"][0]["status"], "missing")

    def test_comparison_only_mode_reports_differences_without_certifying_parity(self):
        code, report = self.run_comparison(modify=True, require_exact=False)
        self.assertEqual(code, 0)
        self.assertFalse(report["screens"][0]["pixel_exact"])
        self.assertFalse(report["pixel_exactness_certified"])
        self.assertFalse(report["require_exact"])


if __name__ == "__main__":
    unittest.main()
