#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


class ReleaseDocumentationContractTests(unittest.TestCase):
    def read(self, relative: str) -> str:
        return (ROOT / relative).read_text(encoding="utf-8")

    def test_release_verification_tracks_version_file(self):
        version = self.read("VERSION").strip()
        text = self.read("docs/RELEASE-VERIFICATION.md")
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

    def test_active_user_docs_track_version_file(self):
        version = self.read("VERSION").strip()
        markers = {
            "README.md": [
                f"Current Ghost FTP version: **{version}**",
                f"Ghost-FTP-{version}-Setup-x64.exe",
                f"Ghost-FTP-{version}-Linux-amd64.deb",
                f"ghcr.io/bren-wp/ghost-ftp:{version}",
            ],
            "docs/README.md": [
                f"**Current Ghost FTP release: {version}**",
                f"ghostftp-v{version}",
                f"ghcr.io/bren-wp/ghost-ftp:{version}",
            ],
            "docs/INSTALLATION.md": [
                f"Ghost FTP **{version} Stable**",
                f"Ghost-FTP-{version}-Setup-x64.exe",
                f"Ghost-FTP-{version}-Linux-amd64.deb",
                f"ghcr.io/bren-wp/ghost-ftp:{version}",
            ],
            "docs/PACKAGES.md": [
                f"Ghost FTP {version} Stable candidate",
                f"ghcr.io/bren-wp/ghost-ftp:{version}",
            ],
            "docs/SUPPORT.md": [f"Ghost FTP **{version} Stable**"],
            "docs/GITHUB-RELEASES.md": [
                f"Ghost FTP **{version} Stable**",
                f"ghostftp-v{version}",
                f"Ghost-FTP-{version}-Setup-x64.exe",
                f"ghcr.io/bren-wp/ghost-ftp:{version}",
            ],
        }
        for relative, required in markers.items():
            text = self.read(relative)
            for marker in required:
                self.assertIn(marker, text, f"{relative} is missing {marker!r}")

    def test_release_docs_describe_canonical_branch_dispatch(self):
        verification = self.read("docs/RELEASE-VERIFICATION.md")
        releases = self.read("docs/GITHUB-RELEASES.md")
        for text in (verification, releases):
            self.assertIn("release/ghostftp-vX.Y.Z", text)
            self.assertIn("workflow_dispatch", text)
            self.assertIn("VERSION", text)
        self.assertIn("A push to `main`, including a change to `VERSION`, must never publish a release directly.", verification)
        self.assertIn("A push to `main`, including a commit that changes `VERSION`, must not publish a release directly.", releases)


if __name__ == "__main__":
    unittest.main()
