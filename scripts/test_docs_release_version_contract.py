#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]

class ReleaseDocumentationContractTests(unittest.TestCase):
    def read(self, rel: str) -> str:
        return (ROOT / rel).read_text(encoding="utf-8")

    def test_release_docs_match_active_platform_shape(self):
        version = self.read("VERSION").strip()
        self.assertEqual(version, "0.0.8")
        verification = self.read("docs/RELEASE-VERIFICATION.md")
        self.assertIn("13 platform artifacts", verification)
        self.assertIn("16 public files", verification)
        self.assertIn("ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID", verification)
        self.assertNotIn("Ghost-FTP-0.0.8-macOS", verification)

    def test_readme_uses_authentic_windows_linux_android_evidence(self):
        readme = self.read("README.md")
        for marker in (
            "docs/images/0.0.8/ghost-ftp-main-workspace.png",
            "docs/images/0.0.8/ghost-ftp-linux-main-workspace.png",
            "docs/images/0.0.8/ghost-ftp-android-files.png",
        ):
            self.assertIn(marker, readme)

    def test_retired_macos_source_is_absent(self):
        self.assertFalse((ROOT / "macos").exists())
        self.assertFalse((ROOT / ".github/workflows/macos-app.yml").exists())
        self.assertFalse((ROOT / ".github/workflows/macos-production.yml").exists())

if __name__ == "__main__":
    unittest.main()
