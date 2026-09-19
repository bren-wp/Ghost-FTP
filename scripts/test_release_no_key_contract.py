#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]

class ReleaseNoKeyContractTests(unittest.TestCase):
    def read(self, rel: str) -> str:
        return (ROOT / rel).read_text(encoding="utf-8")

    def test_no_key_release_has_only_active_platforms(self):
        workflow = self.read(".github/workflows/release-no-key.yml")
        for marker in (
            "Ghost-FTP-${version}-Setup.exe",
            "Ghost-FTP-${version}-Portable.exe",
            "Ghost-FTP-${version}-Android.apk",
            "PUBLIC_PLATFORM_ARTIFACTS=13",
            "PUBLIC_RELEASE_FILES=16",
        ):
            self.assertIn(marker, workflow)
        for retired in ("macos:", "MACOS_", "macOS", "macos/"):
            self.assertNotIn(retired, workflow)

    def test_active_source_platforms_are_explicit(self):
        workflow = self.read(".github/workflows/release-no-key.yml")
        self.assertIn("ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID", workflow)

if __name__ == "__main__":
    unittest.main()
