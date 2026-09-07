#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


class ReleaseDocumentationContractTests(unittest.TestCase):
    def test_release_verification_tracks_version_file(self):
        version = (ROOT / "VERSION").read_text(encoding="utf-8").strip()
        text = (ROOT / "docs/RELEASE-VERIFICATION.md").read_text(encoding="utf-8")
        required = [
            f"current maintained release is **{version} Stable**",
            f"## Expected {version} release identity",
            f"VERSION={version}",
            f"TAG=ghostftp-v{version}",
            f"TITLE=Ghost FTP {version}",
            f"Ghost-FTP-{version}-Setup-x64.exe",
            f"ghcr.io/bren-wp/ghost-ftp:{version}",
        ]
        for marker in required:
            self.assertIn(marker, text)

    def test_release_docs_describe_canonical_branch_dispatch(self):
        verification = (ROOT / "docs/RELEASE-VERIFICATION.md").read_text(encoding="utf-8")
        releases = (ROOT / "docs/GITHUB-RELEASES.md").read_text(encoding="utf-8")
        for text in (verification, releases):
            self.assertIn("release/ghostftp-vX.Y.Z", text)
            self.assertIn("workflow_dispatch", text)
            self.assertIn("VERSION", text)
        self.assertIn("A push to `main`, including a change to `VERSION`, must never publish a release directly.", verification)
        self.assertIn("A push to `main`, including a commit that changes `VERSION`, must not publish a release directly.", releases)


if __name__ == "__main__":
    unittest.main()
