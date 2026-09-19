#!/usr/bin/env python3
from __future__ import annotations

import re
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]

class ReadmeMediaContractTests(unittest.TestCase):
    required_root_media = {
        "build/icon.png",
        "docs/images/ghost-ftp-main-workspace.png",
        "docs/images/ghost-ftp-linux-main-workspace.png",
        "docs/images/ghost-ftp-android-files.png",
    }

    def _image_sources(self, text: str) -> set[str]:
        html = set(re.findall(r'<img\s+[^>]*src=["\']([^"\']+)["\']', text, flags=re.I))
        markdown = set(re.findall(r'!\[[^\]]*\]\(([^)\s]+)(?:\s+"[^"]*")?\)', text))
        return {value.strip() for value in html | markdown if value.strip()}

    def _assert_local_existing_images(self, rel: str) -> set[str]:
        doc = ROOT / rel
        text = doc.read_text(encoding="utf-8")
        sources = self._image_sources(text)
        self.assertTrue(sources, f"{rel} must render at least one local Ghost FTP image")
        for source in sorted(sources):
            lowered = source.lower()
            self.assertFalse(lowered.startswith(("http://", "https://", "//", "data:")), source)
            self.assertNotIn("?", source, source)
            resolved = (doc.parent / source).resolve()
            resolved.relative_to(ROOT.resolve())
            self.assertTrue(resolved.is_file(), f"{rel} references missing image {source!r}")
        return sources

    def test_root_readme_uses_real_icon_and_current_cross_platform_runtime_media(self) -> None:
        sources = self._assert_local_existing_images("README.md")
        self.assertEqual(sorted(self.required_root_media - sources), [])
        readme = (ROOT / "README.md").read_text(encoding="utf-8").lower()
        self.assertIn("authentic application screenshots", readme)
        self.assertIn("product evidence, not generated mockups", readme)
        self.assertIn("windows · linux · android", readme)

    def test_current_readme_does_not_use_generated_or_remote_images(self) -> None:
        sources = self._assert_local_existing_images("README.md")
        for source in sources:
            self.assertNotIn("generated", source.lower())
            self.assertFalse(source.lower().startswith(("http://", "https://")))

    def test_historical_008_evidence_remains_version_bound(self) -> None:
        historical = ROOT / "docs" / "images" / "0.0.8"
        self.assertTrue((historical / "UI-SCREENSHOT-PROVENANCE.json").is_file())
        self.assertTrue((historical / "SHA256.txt").is_file())

    def test_ui_screenshot_workflow_is_read_only_and_cross_platform(self) -> None:
        workflow = (ROOT / ".github/workflows/ui-screenshots.yml").read_text(encoding="utf-8")
        lower = workflow.lower()
        for marker in (
            "ghostftp-authentic-ui-windows",
            "ghostftp-authentic-ui-linux",
            "ghostftp-authentic-ui-android",
            "ghostftp-authentic-ui-verified-bundle",
        ):
            self.assertIn(marker, workflow)
        self.assertNotIn("contents: write", lower)
        self.assertNotIn("git push", lower)
        self.assertNotIn("github-actions[bot]", lower)

if __name__ == "__main__":
    unittest.main()
