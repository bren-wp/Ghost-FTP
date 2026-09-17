#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


class ReleaseDocumentationContractTests(unittest.TestCase):
    def read(self, relative: str) -> str:
        return (ROOT / relative).read_text(encoding="utf-8")

    def test_release_verification_tracks_version_file(self):
        version = self.read("VERSION").strip()
        self.assertEqual(version, "0.0.7")
        text = self.read("docs/RELEASE-VERIFICATION.md")
        required = [
            f"Ghost FTP **{version}** is the active release candidate",
            f"VERSION={version}",
            f"TAG=ghostftp-v{version}",
            f"TITLE=Ghost FTP {version}",
            "CHANNEL=Current",
            "PRERELEASE=false",
            f"Ghost-FTP-{version}-Setup.exe",
            f"Ghost-FTP-{version}-Portable.exe",
            f"Ghost-FTP-{version}-Linux-Debian-Installer.run",
            f"Ghost-FTP-{version}-Linux-Debian-Portable.tar.gz",
            f"Ghost-FTP-{version}-Linux-Ubuntu-Installer.run",
            f"Ghost-FTP-{version}-Linux-Ubuntu-Portable.tar.gz",
            f"Ghost-FTP-{version}-Linux-Fedora-Installer.run",
            f"Ghost-FTP-{version}-Linux-Fedora-Portable.tar.gz",
            f"Ghost-FTP-{version}-Android.apk",
            f"Ghost-FTP-{version}-Chrome-Extension.zip",
            f"Ghost-FTP-{version}-Edge-Extension.zip",
            f"Ghost-FTP-{version}-Firefox-Extension.zip",
            f"Ghost-FTP-{version}-Opera-Extension.zip",
            "PUBLIC_PLATFORM_ARTIFACTS=13",
            "PUBLIC_RELEASE_FILES=16",
            "GHOSTFTP_ANDROID_CERT_SHA256",
            f"ghcr.io/bren-wp/ghost-ftp:{version}",
        ]
        for marker in required:
            self.assertIn(marker, text)
        for stale in (
            f"Ghost-FTP-{version}-Setup-x64.exe",
            f"Ghost-FTP-{version}-Setup-x86.exe",
            f"Ghost-FTP-{version}-Linux-Debian-amd64.deb",
            f"Ghost-FTP-{version}-Linux-Fedora-x86_64.rpm",
            "PUBLIC_PLATFORM_ARTIFACTS=18",
            "PUBLIC_RELEASE_FILES=21",
            "GHOSTFTP_ANDROID_SIGNER_SHA256",
        ):
            self.assertNotIn(stale, text)

    def test_release_facing_docs_match_007_shape(self):
        version = self.read("VERSION").strip()
        for relative in (
            "docs/INSTALLATION.md",
            "docs/GITHUB-RELEASES.md",
            "docs/RELEASE-VERIFICATION.md",
        ):
            text = self.read(relative)
            for marker in (
                f"Ghost-FTP-{version}-Setup.exe",
                f"Ghost-FTP-{version}-Portable.exe",
                f"Ghost-FTP-{version}-Linux-Debian-Installer.run",
                f"Ghost-FTP-{version}-Linux-Debian-Portable.tar.gz",
                f"Ghost-FTP-{version}-Linux-Ubuntu-Installer.run",
                f"Ghost-FTP-{version}-Linux-Ubuntu-Portable.tar.gz",
                f"Ghost-FTP-{version}-Linux-Fedora-Installer.run",
                f"Ghost-FTP-{version}-Linux-Fedora-Portable.tar.gz",
                f"Ghost-FTP-{version}-Android.apk",
                f"Ghost-FTP-{version}-Chrome-Extension.zip",
                f"Ghost-FTP-{version}-Edge-Extension.zip",
                f"Ghost-FTP-{version}-Firefox-Extension.zip",
                f"Ghost-FTP-{version}-Opera-Extension.zip",
                "13 platform artifacts",
                "16 public files",
            ):
                self.assertIn(marker, text, f"{relative} is missing {marker!r}")
            for stale in (
                f"Ghost-FTP-{version}-Linux-Debian-amd64.deb",
                f"Ghost-FTP-{version}-Linux-Ubuntu-amd64.deb",
                f"Ghost-FTP-{version}-Linux-Fedora-x86_64.rpm",
                f"Ghost-FTP-{version}-Linux-Portable-amd64.tar.gz",
                "18 platform artifacts / 21 public files",
            ):
                self.assertNotIn(stale, text, relative)

    def test_android_and_browser_boundaries_remain_truthful(self):
        version = self.read("VERSION").strip()
        installation = self.read("docs/INSTALLATION.md")
        verification = self.read("docs/RELEASE-VERIFICATION.md")
        for text in (installation, verification):
            self.assertIn(f"Ghost-FTP-{version}-Android.apk", text)
            self.assertIn("SFTP", text)
            self.assertIn("host-key", text.lower())
        self.assertIn("GHOSTFTP_ANDROID_CERT_SHA256", verification)
        self.assertNotIn("GHOSTFTP_ANDROID_SIGNER_SHA256", verification)

        extension_docs = self.read("extensions/README.md")
        for browser in ("Chrome", "Edge", "Firefox", "Opera"):
            self.assertIn(browser, extension_docs)
        extension_lower = extension_docs.lower()
        self.assertIn("ghostftp://connect", extension_lower)
        self.assertIn("open in ghost ftp", extension_lower)
        self.assertIn("zero browser permissions and zero host permissions", extension_lower)
        self.assertIn("credentials remain empty", extension_lower)
        self.assertIn("without automatically connecting", extension_lower)
        self.assertIn("no compatible desktop handler", extension_lower)
        self.assertIn("passwords, private-key passphrases, private keys, source query data and source fragments are never copied", extension_lower)

    def test_release_docs_describe_canonical_branch_dispatch(self):
        verification = self.read("docs/RELEASE-VERIFICATION.md")
        releases = self.read("docs/GITHUB-RELEASES.md")
        for text in (verification, releases):
            self.assertIn("release/ghostftp-vX.Y.Z", text)
            self.assertIn("workflow_dispatch", text)
            self.assertIn("VERSION", text)
        self.assertIn("must never publish a release directly", verification)
        self.assertIn("does not publish a release directly", releases)

    def test_published_history_is_not_misrepresented(self):
        for relative in ("docs/INSTALLATION.md", "docs/GITHUB-RELEASES.md", "docs/RELEASE-VERIFICATION.md"):
            text = self.read(relative)
            self.assertIn("last actually published github", text.lower())
            self.assertIn("0.0.6", text)
            self.assertIn("0.0.7", text)


if __name__ == "__main__":
    unittest.main()
