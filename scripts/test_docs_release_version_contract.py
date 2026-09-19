#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


class ReleaseDocumentationContractTests(unittest.TestCase):
    def read(self, relative: str) -> str:
        return (ROOT / relative).read_text(encoding="utf-8")

    def test_active_release_shape_is_windows_linux_android(self):
        version = self.read("VERSION").strip()
        self.assertEqual(version, "0.0.8")

        verification = self.read("docs/RELEASE-VERIFICATION.md")
        for marker in (
            "PUBLIC_RELEASE_PLATFORMS=WINDOWS,LINUX,ANDROID,BROWSER_HELPER",
            "ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID",
            "PUBLIC_PLATFORM_ARTIFACTS=13",
            "PUBLIC_RELEASE_FILES=16",
        ):
            self.assertIn(marker, verification)

        for relative in (
            "README.md",
            "docs/README.md",
            "docs/PLATFORM-PARITY.md",
            "docs/REFERENCE-UI.md",
            "docs/TESTING.md",
        ):
            text = self.read(relative)
            self.assertIn("Windows", text, relative)
            self.assertIn("Linux", text, relative)
            self.assertIn("Android", text, relative)

    def test_macos_is_retired_from_active_source_and_release(self):
        self.assertFalse((ROOT / "macos").exists())
        self.assertFalse((ROOT / ".github/workflows/macos-app.yml").exists())
        self.assertFalse((ROOT / ".github/workflows/macos-production.yml").exists())

        release = self.read(".github/workflows/release.yml")
        no_key = self.read(".github/workflows/release-no-key.yml")
        for text in (release, no_key):
            self.assertNotIn("macos:", text.lower())
            self.assertNotIn("Ghost-FTP-${VERSION}-macOS", text)
            self.assertNotIn("MACOS_APP=", text)
            self.assertNotIn("APPLE_NOTARY", text)

    def test_readme_uses_real_repository_branding_and_runtime_images(self):
        readme = self.read("README.md")
        self.assertIn('src="build/icon.png"', readme)
        self.assertIn("Authentic application screenshots", readme)
        self.assertIn("docs/images/ghost-ftp-main-workspace.png", readme)
        self.assertIn("docs/images/ghost-ftp-linux-main-workspace.png", readme)
        self.assertIn("docs/images/ghost-ftp-android-files.png", readme)
        self.assertIn("product evidence, not generated mockups", readme)

        for relative in (
            "docs/images/ghost-ftp-main-workspace.png",
            "docs/images/ghost-ftp-linux-main-workspace.png",
            "docs/images/ghost-ftp-android-files.png",
            "build/icon.png",
        ):
            self.assertTrue((ROOT / relative).is_file(), relative)

    def test_reference_ui_preserves_master_information_hierarchy(self):
        reference = self.read("docs/REFERENCE-UI.md")
        for marker in (
            "Files",
            "Connections / Sites",
            "Transfer Queue / Transfers",
            "Settings",
            "Back",
            "Forward",
            "Refresh",
            "New Folder",
            "Upload",
            "Download",
            "Bookmarks",
            "More",
            "Local Files",
            "Remote Files",
            "File",
            "Direction",
            "Progress",
            "Status",
            "Speed",
            "ETA",
            "Clear Completed",
        ):
            self.assertIn(marker, reference)

    def test_security_and_android_sftp_boundary_stay_truthful(self):
        security = self.read("docs/SECURITY.md")
        readme = self.read("README.md")
        for text in (security, readme):
            self.assertIn("FTPS", text)
            self.assertIn("SFTP", text)
            self.assertIn("host-key", text.lower())
        self.assertIn("Android", security)
        self.assertIn("hidden", security.lower())

    def test_release_workflows_and_retention_use_new_counts(self):
        release = self.read(".github/workflows/release.yml")
        no_key = self.read(".github/workflows/release-no-key.yml")
        retention = self.read(".github/workflows/release-retention.yml")

        for text in (release, no_key):
            self.assertIn("PUBLIC_PLATFORM_ARTIFACTS=13", text)
            self.assertIn("PUBLIC_RELEASE_FILES=16", text)
        self.assertIn('test "$asset_count" -eq 16', retention)
        self.assertIn("PROTECTED_RELEASE_TAG=ghostftp-v0.0.7", retention)

        self.assertNotIn("PUBLIC_PLATFORM_ARTIFACTS=14", release)
        self.assertNotIn("PUBLIC_RELEASE_FILES=17", release)
        self.assertNotIn("assets=17", no_key)


if __name__ == "__main__":
    unittest.main()
