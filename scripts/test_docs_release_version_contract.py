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
            f"current maintained release is **{version}**",
            f"## Published {version} release identity",
            f"VERSION={version}",
            f"TAG=ghostftp-v{version}",
            f"TITLE=Ghost FTP {version}",
            "CHANNEL=Current",
            "PRERELEASE=false",
            f"Ghost-FTP-{version}-Setup.exe",
            f"Ghost-FTP-{version}-Portable.exe",
            f"Ghost-FTP-{version}-Linux-Debian-amd64.deb",
            f"Ghost-FTP-{version}-Linux-Ubuntu-amd64.deb",
            f"Ghost-FTP-{version}-Linux-Fedora-x86_64.rpm",
            f"Ghost-FTP-{version}-Linux-Portable-amd64.tar.gz",
            "PUBLIC_PLATFORM_ARTIFACTS=14",
            "PUBLIC_RELEASE_FILES=17",
            f"ghcr.io/bren-wp/ghost-ftp:{version}",
        ]
        for marker in required:
            self.assertIn(marker, text)
        self.assertNotIn(f"Ghost-FTP-{version}-Setup-x64.exe", text)
        self.assertNotIn(f"Ghost-FTP-{version}-Setup-x86.exe", text)

    def test_active_user_docs_track_version_file(self):
        version = self.read("VERSION").strip()
        markers = {
            "README.md": [
                f"Current Ghost FTP version: **{version}**",
                "Release channel: **Current**",
                f"Ghost-FTP-{version}-Setup.exe",
                f"Ghost-FTP-{version}-Portable.exe",
                f"Ghost-FTP-{version}-Linux-Debian-amd64.deb",
                f"Ghost-FTP-{version}-Linux-Fedora-x86_64.rpm",
                "14 platform artifacts / 17 public files",
                f"ghcr.io/bren-wp/ghost-ftp:{version}",
            ],
            "docs/README.md": [
                f"**Current Ghost FTP release: {version}**",
                "Release channel: **Current**",
                f"ghostftp-v{version}",
                f"Ghost-FTP-{version}-Setup.exe",
                f"Ghost-FTP-{version}-Linux-Ubuntu-amd64.deb",
                "14 platform artifacts / 17 public files",
                f"ghcr.io/bren-wp/ghost-ftp:{version}",
            ],
            "docs/INSTALLATION.md": [
                f"Ghost FTP **{version}** is the current published release",
                f"Ghost-FTP-{version}-Setup.exe",
                f"Ghost-FTP-{version}-Portable.exe",
                f"Ghost-FTP-{version}-Linux-Debian-amd64.deb",
                f"Ghost-FTP-{version}-Linux-Ubuntu-amd64.deb",
                f"Ghost-FTP-{version}-Linux-Fedora-x86_64.rpm",
                "14 platform artifacts / 17 public files",
                f"ghcr.io/bren-wp/ghost-ftp:{version}",
            ],
            "docs/PACKAGES.md": [
                f"Ghost FTP **{version}** publishes a verified **distribution bundle**",
                "14 platform artifacts / 17 public files",
                f"ghcr.io/bren-wp/ghost-ftp:{version}",
            ],
            "docs/SUPPORT.md": [
                f"Ghost FTP **{version}** is the current supported public release",
                "https://ghostftp.com",
            ],
            "docs/GITHUB-RELEASES.md": [
                f"Ghost FTP **{version}** is the current published release contract",
                f"ghostftp-v{version}",
                f"Ghost-FTP-{version}-Setup.exe",
                f"Ghost-FTP-{version}-Linux-Portable-amd64.tar.gz",
                "14 platform artifacts",
                "17 public files",
                f"ghcr.io/bren-wp/ghost-ftp:{version}",
                "Prerelease: false",
            ],
        }
        for relative, required in markers.items():
            text = self.read(relative)
            for marker in required:
                self.assertIn(marker, text, f"{relative} is missing {marker!r}")
            self.assertNotIn("prerelease=true", text, relative)

        for relative in ("README.md", "docs/README.md", "docs/INSTALLATION.md", "docs/GITHUB-RELEASES.md"):
            text = self.read(relative)
            self.assertNotIn(f"Ghost-FTP-{version}-Setup-x64.exe", text, relative)
            self.assertNotIn(f"Ghost-FTP-{version}-Setup-x86.exe", text, relative)

    def test_current_docs_do_not_revert_to_old_channel_status(self):
        stale = (
            "0.0.1 Beta",
            "CHANNEL=Beta",
            "PRERELEASE=true",
            "major 0 → beta",
            "major 0 -> beta",
        )
        for relative in (
            "README.md",
            "docs/INSTALLATION.md",
            "docs/PACKAGES.md",
            "docs/GITHUB-RELEASES.md",
            "docs/RELEASE-VERIFICATION.md",
        ):
            text = self.read(relative)
            for marker in stale:
                self.assertNotIn(marker, text, f"{relative} still contains stale channel wording: {marker!r}")

    def test_release_docs_describe_canonical_branch_dispatch(self):
        verification = self.read("docs/RELEASE-VERIFICATION.md")
        releases = self.read("docs/GITHUB-RELEASES.md")
        for text in (verification, releases):
            self.assertIn("release/ghostftp-vX.Y.Z", text)
            self.assertIn("workflow_dispatch", text)
            self.assertIn("VERSION", text)
        self.assertIn("A push to `main`, including a change to `VERSION`, must never publish a release directly.", verification)
        self.assertIn("does not publish a release directly", releases)


if __name__ == "__main__":
    unittest.main()
