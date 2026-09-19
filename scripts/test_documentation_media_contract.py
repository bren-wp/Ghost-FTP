#!/usr/bin/env python3
"""Regression checks for authentic Ghost FTP documentation media."""
from __future__ import annotations

import hashlib
import json
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
HISTORICAL = ROOT / "docs" / "images" / "0.0.8"
CURRENT_IMAGES = (
    ROOT / "docs" / "images" / "ghost-ftp-main-workspace.png",
    ROOT / "docs" / "images" / "ghost-ftp-linux-main-workspace.png",
    ROOT / "docs" / "images" / "ghost-ftp-android-files.png",
)
EXPECTED_CAPTURE_SHA = "54032022c926b1ac07a4964785b20882e02ca582"
EXPECTED_RUN_ID = 35329686590

class DocumentationMediaContractTest(unittest.TestCase):
    def test_historical_008_evidence_keeps_exact_provenance(self) -> None:
        provenance_path = HISTORICAL / "UI-SCREENSHOT-PROVENANCE.json"
        self.assertTrue(provenance_path.is_file())
        provenance = json.loads(provenance_path.read_text(encoding="utf-8"))
        self.assertEqual(provenance.get("capture_source_sha"), EXPECTED_CAPTURE_SHA)
        self.assertEqual(provenance.get("workflow_run_id"), EXPECTED_RUN_ID)
        self.assertEqual(provenance.get("evidence"), "authentic-runtime-capture")

        images = provenance.get("images") or []
        self.assertEqual(len(images), 18)
        for item in images:
            name = Path(item["path"]).name
            payload = (HISTORICAL / name).read_bytes()
            self.assertEqual(len(payload), item["bytes"], name)
            self.assertEqual(hashlib.sha256(payload).hexdigest(), item["sha256"], name)

        digest_lines = (HISTORICAL / "SHA256.txt").read_text(encoding="utf-8").splitlines()
        self.assertEqual(len(digest_lines), 18)

    def test_current_readme_uses_current_repository_runtime_media(self) -> None:
        for path in CURRENT_IMAGES:
            self.assertTrue(path.is_file(), path)
            self.assertGreater(path.stat().st_size, 0, path)

        readme = (ROOT / "README.md").read_text(encoding="utf-8")
        for marker in (
            "docs/images/ghost-ftp-main-workspace.png",
            "docs/images/ghost-ftp-linux-main-workspace.png",
            "docs/images/ghost-ftp-android-files.png",
        ):
            self.assertIn(marker, readme)
        self.assertIn("product evidence, not generated mockups", readme)

    def test_reference_ui_rejects_mockups_as_runtime_evidence(self) -> None:
        reference = (ROOT / "docs" / "REFERENCE-UI.md").read_text(encoding="utf-8")
        self.assertIn("Only authentic runtime screenshots are product evidence.", reference)
        self.assertIn("Generated images and visual references are design targets, not proof of execution.", reference)
        for marker in ("Windows", "Linux", "Android"):
            self.assertIn(marker, reference)

    def test_one_time_importer_is_not_part_of_product_tree(self) -> None:
        self.assertFalse((ROOT / ".github" / "workflows" / "import-0.0.6-ui-media.yml").exists())

if __name__ == "__main__":
    unittest.main()
