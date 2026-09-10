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
        "docs/images/ghost-ftp-site-manager.png",
        "docs/images/ghost-ftp-settings.png",
        "docs/images/ghost-ftp-about.png",
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
            self.assertFalse(
                lowered.startswith(("http://", "https://", "//", "data:")),
                f"{rel} contains non-local image source {source!r}",
            )
            self.assertNotIn("?", source, f"{rel} image source must be a stable repository path: {source!r}")
            resolved = (doc.parent / source).resolve()
            try:
                resolved.relative_to(ROOT.resolve())
            except ValueError as exc:
                self.fail(f"{rel} image source escapes repository root: {source!r}: {exc}")
            self.assertTrue(resolved.is_file(), f"{rel} references missing image {source!r}")
        return sources

    def test_root_readme_uses_product_icon_and_authentic_screenshot_set(self) -> None:
        sources = self._assert_local_existing_images("README.md")
        missing = sorted(self.required_root_media - sources)
        self.assertEqual(missing, [], "README is missing required maintained product media: " + ", ".join(missing))

    def test_docs_index_uses_only_local_media(self) -> None:
        sources = self._assert_local_existing_images("docs/README.md")
        expected = {
            "../build/icon.png",
            "images/ghost-ftp-main-workspace.png",
            "images/ghost-ftp-site-manager.png",
            "images/ghost-ftp-settings.png",
            "images/ghost-ftp-about.png",
        }
        self.assertEqual(sorted(expected - sources), [], "docs index is missing maintained product media")

    def test_readme_copy_keeps_authentic_media_provenance_explicit(self) -> None:
        root = (ROOT / "README.md").read_text(encoding="utf-8")
        docs = (ROOT / "docs/README.md").read_text(encoding="utf-8")
        for text, label in ((root, "README.md"), (docs, "docs/README.md")):
            lowered = text.lower()
            self.assertIn("repository-local", lowered, f"{label} must state local media provenance")
            self.assertIn(
                "verified internal native x64 payload",
                lowered,
                f"{label} must bind screenshots to the verified internal native x64 evidence payload",
            )
            self.assertIn(
                "universal windows build",
                lowered,
                f"{label} must bind native screenshot evidence to the universal Windows build chain",
            )
            self.assertIn("mockup", lowered, f"{label} must reject mockups as production evidence")
            self.assertRegex(
                lowered,
                r"not (?:(?:an architecture-specific|a) )?public download",
                f"{label} must not present internal x64 evidence as a public artifact",
            )


if __name__ == "__main__":
    unittest.main()
