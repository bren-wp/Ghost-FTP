#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


class NoKeyReleaseContractTests(unittest.TestCase):
    def read(self, relative: str) -> str:
        return (ROOT / relative).read_text(encoding="utf-8")

    def test_no_key_workflow_is_explicit_and_secret_free(self):
        workflow = self.read(".github/workflows/release-no-key.yml")
        self.assertIn("Publish Ghost FTP No-Key Distribution", workflow)
        self.assertIn("release/ghostftp-v0.0.8-no-key", workflow)
        self.assertNotIn("secrets.", workflow)
        self.assertIn("WINDOWS_AUTHENTICODE=unsigned-no-key-distribution", workflow)
        self.assertIn("ANDROID_APK=debug-signed-no-secret", workflow)
        self.assertIn("MACOS_SIGNING=adhoc-validation-no-notarization", workflow)
        self.assertIn("DISTRIBUTION_MODE=no-secret-public-release", workflow)

    def test_no_key_release_preserves_security_boundaries(self):
        workflow = self.read(".github/workflows/release-no-key.yml")
        self.assertIn("go telemetry off", workflow)
        self.assertIn("scripts/audit_security.py", workflow)
        self.assertIn("scripts/audit_privacy.py", workflow)
        self.assertIn("Android SFTP remains hidden", workflow)
        self.assertNotIn("StrictHostKeyChecking=no", workflow)
        self.assertNotIn("InsecureSkipVerify", workflow)
        self.assertNotIn("gh release upload", workflow)
        self.assertNotIn("--clobber", workflow)

    def test_no_key_release_is_exact_main_and_immutable(self):
        workflow = self.read(".github/workflows/release-no-key.yml")
        self.assertIn('main_sha="$(gh api "repos/$GITHUB_REPOSITORY/commits/main" --jq .sha)"', workflow)
        self.assertIn('test "$GITHUB_SHA" = "$main_sha"', workflow)
        self.assertIn("release already exists; refusing to rewrite published assets", workflow)
        self.assertIn("tag already exists; refusing to move release identity", workflow)
        self.assertIn("test \"$(jq '.assets | length' <<< \"$json\")\" = '17'", workflow)
        self.assertIn('test "$tag_sha" = "$GITHUB_SHA"', workflow)

    def test_public_shape_is_all_supported_platforms(self):
        workflow = self.read(".github/workflows/release-no-key.yml")
        expected = (
            "Ghost-FTP-${version}-Setup.exe",
            "Ghost-FTP-${version}-Portable.exe",
            "Ghost-FTP-${version}-Android.apk",
            "Ghost-FTP-${version}-macOS.app.zip",
            "Ghost-FTP-${version}-Linux-Debian-Installer.run",
            "Ghost-FTP-${version}-Linux-Debian-Portable.tar.gz",
            "Ghost-FTP-${version}-Linux-Ubuntu-Installer.run",
            "Ghost-FTP-${version}-Linux-Ubuntu-Portable.tar.gz",
            "Ghost-FTP-${version}-Linux-Fedora-Installer.run",
            "Ghost-FTP-${version}-Linux-Fedora-Portable.tar.gz",
            "Ghost-FTP-${version}-Chrome-Extension.zip",
            "Ghost-FTP-${version}-Edge-Extension.zip",
            "Ghost-FTP-${version}-Firefox-Extension.zip",
            "Ghost-FTP-${version}-Opera-Extension.zip",
            "PUBLIC_PLATFORM_ARTIFACTS=14",
            "PUBLIC_RELEASE_FILES=17",
        )
        for marker in expected:
            self.assertIn(marker, workflow)


if __name__ == "__main__":
    unittest.main()
