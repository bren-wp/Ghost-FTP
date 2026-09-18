#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


class ReleaseDocumentationContractTests(unittest.TestCase):
    def read(self, relative: str) -> str:
        return (ROOT / relative).read_text(encoding="utf-8")

    def test_release_verification_tracks_version_file(self):
        version = self.read("VERSION").strip()
        self.assertEqual(version, "0.0.8")
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
            f"Ghost-FTP-{version}-macOS.app.zip",
            f"Ghost-FTP-{version}-Chrome-Extension.zip",
            f"Ghost-FTP-{version}-Edge-Extension.zip",
            f"Ghost-FTP-{version}-Firefox-Extension.zip",
            f"Ghost-FTP-{version}-Opera-Extension.zip",
            "PUBLIC_PLATFORM_ARTIFACTS=14",
            "PUBLIC_RELEASE_FILES=17",
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

    def test_release_facing_docs_match_008_shape(self):
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
                f"Ghost-FTP-{version}-macOS.app.zip",
                f"Ghost-FTP-{version}-Chrome-Extension.zip",
                f"Ghost-FTP-{version}-Edge-Extension.zip",
                f"Ghost-FTP-{version}-Firefox-Extension.zip",
                f"Ghost-FTP-{version}-Opera-Extension.zip",
                "14 platform artifacts",
                "17 public files",
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

    def test_active_ui_docs_use_008_authentic_evidence(self):
        readme = self.read("README.md")
        docs_index = self.read("docs/README.md")
        reference = self.read("docs/REFERENCE-UI.md")
        testing = self.read("docs/TESTING.md")

        for text in (readme, docs_index, reference):
            self.assertIn("images/0.0.8/", text)
            self.assertNotIn("images/0.0.6/ghost-ftp-main-workspace.png", text)

        self.assertIn("18-image 0.0.8 evidence contract", readme)
        self.assertIn("exactly **18 runtime images**", reference)
        self.assertIn("Windows 5, Linux 5 and Android 8", testing)
        self.assertIn("ghost-ftp-android-transfer-queue.png", docs_index)
        self.assertNotIn("Windows Site Manager", docs_index)
        self.assertNotIn("Android Transfers", docs_index)

    def test_release_docs_preserve_007_and_report_real_browser_handoff(self):
        verification = self.read("docs/RELEASE-VERIFICATION.md")
        releases = self.read("docs/GITHUB-RELEASES.md")
        history = self.read("docs/RELEASE-HISTORY.md")
        testing = self.read("docs/TESTING.md")

        for text in (verification, releases, history, testing):
            self.assertIn("0.0.7", text)

        self.assertIn("PROTECTED_RELEASE_TAG=ghostftp-v0.0.7", verification)
        self.assertIn("BROWSER_DESKTOP_HANDOFF=sanitized-ghostftp-connect-no-autoconnect", verification)
        self.assertIn("BROWSER_DESKTOP_HANDOFF=sanitized-ghostftp-connect-no-autoconnect", releases)
        self.assertNotIn("LATEST_ONLY_RELEASE_RETENTION=YES", verification)
        self.assertIn("exactly four packages", testing)
        self.assertIn("Ghost-FTP-0.0.8-Opera-Extension.zip", testing)
        self.assertIn("17 public release files", testing)

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

    def test_release_docs_identify_current_target_and_protected_baseline(self):
        for relative in ("docs/INSTALLATION.md", "docs/GITHUB-RELEASES.md", "docs/RELEASE-VERIFICATION.md"):
            text = self.read(relative)
            self.assertIn("0.0.8", text)
            self.assertIn("protected", text.lower())
            self.assertIn("0.0.7", text)
            self.assertIn("compatibility", text.lower())


if __name__ == "__main__":
    unittest.main()
