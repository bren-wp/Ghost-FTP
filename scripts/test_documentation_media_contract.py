#!/usr/bin/env python3
"""Regression checks for version-bound authentic documentation media."""

from __future__ import annotations

import hashlib
import json
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
MEDIA = ROOT / "docs" / "images" / "0.0.6"
EXPECTED_CAPTURE_SHA = "9adace20030a300c39eb320a97482eb50dfdb9d8"
EXPECTED_RUN_ID = 34863585111
EXPECTED_IMAGES = {
    "ghost-ftp-main-workspace.png",
    "ghost-ftp-site-manager.png",
    "ghost-ftp-bookmarks.png",
    "ghost-ftp-settings.png",
    "ghost-ftp-about.png",
    "ghost-ftp-linux-main-workspace.png",
    "ghost-ftp-linux-bookmarks.png",
    "ghost-ftp-linux-settings.png",
    "ghost-ftp-android-files.png",
    "ghost-ftp-android-navigation.png",
    "ghost-ftp-android-sites.png",
    "ghost-ftp-android-bookmarks.png",
    "ghost-ftp-android-transfers.png",
    "ghost-ftp-android-settings.png",
    "ghost-ftp-android-about.png",
}


class DocumentationMediaContractTest(unittest.TestCase):
    def test_verified_006_media_matches_provenance(self) -> None:
        provenance_path = MEDIA / "UI-SCREENSHOT-PROVENANCE.json"
        self.assertTrue(provenance_path.is_file())
        provenance = json.loads(provenance_path.read_text(encoding="utf-8"))

        self.assertEqual(provenance.get("capture_source_sha"), EXPECTED_CAPTURE_SHA)
        self.assertEqual(provenance.get("workflow_run_id"), EXPECTED_RUN_ID)
        self.assertEqual(provenance.get("evidence"), "authentic-runtime-capture")

        images = provenance.get("images") or []
        names = {Path(item["path"]).name for item in images}
        self.assertEqual(len(images), 15)
        self.assertEqual(names, EXPECTED_IMAGES)
        self.assertEqual({p.name for p in MEDIA.glob("*.png")}, EXPECTED_IMAGES)

        for item in images:
            name = Path(item["path"]).name
            payload = (MEDIA / name).read_bytes()
            self.assertEqual(len(payload), item["bytes"], name)
            self.assertEqual(hashlib.sha256(payload).hexdigest(), item["sha256"], name)

        digest_lines = (MEDIA / "SHA256.txt").read_text(encoding="utf-8").splitlines()
        self.assertEqual(len(digest_lines), 15)
        expected_lines = sorted(
            f"{item['sha256']}  {Path(item['path']).name}" for item in images
        )
        self.assertEqual(sorted(digest_lines), expected_lines)

    def test_readme_uses_version_bound_runtime_media(self) -> None:
        readme = (ROOT / "README.md").read_text(encoding="utf-8")
        for marker in (
            "docs/images/0.0.6/ghost-ftp-main-workspace.png",
            "docs/images/0.0.6/ghost-ftp-site-manager.png",
            "docs/images/0.0.6/ghost-ftp-settings.png",
            "docs/images/0.0.6/ghost-ftp-linux-main-workspace.png",
            "docs/images/0.0.6/ghost-ftp-android-files.png",
            "docs/images/0.0.6/ghost-ftp-about.png",
            EXPECTED_CAPTURE_SHA,
            str(EXPECTED_RUN_ID),
        ):
            self.assertIn(marker, readme)

        for retired in (
            "](docs/images/ghost-ftp-main-workspace.png)",
            'src="docs/images/ghost-ftp-site-manager.png"',
            'src="docs/images/ghost-ftp-settings.png"',
            'src="docs/images/ghost-ftp-about.png"',
        ):
            self.assertNotIn(retired, readme)

    def test_reference_ui_uses_version_bound_media_and_all_browsers(self) -> None:
        reference = (ROOT / "docs" / "REFERENCE-UI.md").read_text(encoding="utf-8")
        for marker in (
            "images/0.0.6/ghost-ftp-main-workspace.png",
            "images/0.0.6/ghost-ftp-linux-main-workspace.png",
            "images/0.0.6/ghost-ftp-android-files.png",
            "Chrome, Edge, Firefox and Opera",
            EXPECTED_CAPTURE_SHA,
            str(EXPECTED_RUN_ID),
        ):
            self.assertIn(marker, reference)

    def test_one_time_importer_is_not_part_of_product_tree(self) -> None:
        self.assertFalse((ROOT / ".github" / "workflows" / "import-0.0.6-ui-media.yml").exists())


if __name__ == "__main__":
    unittest.main()
