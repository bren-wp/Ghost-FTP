#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]
WORKFLOW = ROOT / ".github/workflows/compatibility-release.yml"


class CompatibilityReleaseContractTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.text = WORKFLOW.read_text(encoding="utf-8")

    def test_exact_main_and_version_are_required(self):
        self.assertIn('main_sha="$(gh api "repos/$GITHUB_REPOSITORY/commits/main" --jq .sha)"', self.text)
        self.assertIn('test "$GITHUB_SHA" = "$main_sha"', self.text)
        self.assertIn('branch_version="${GITHUB_REF_NAME#compat/ghostftp-v}"', self.text)

    def test_windows_is_explicitly_unsigned(self):
        self.assertIn("WINDOWS_COMPATIBILITY_SIGNING=UNSIGNED", self.text)
        self.assertIn("WINDOWS_AUTHENTICODE=UNSIGNED_COMPATIBILITY_RELEASE", self.text)
        self.assertNotIn("GHOSTFTP_SIGNING_PFX_BASE64", self.text)

    def test_android_uses_one_run_compatibility_identity(self):
        self.assertIn("ghostftp-compatibility.jks", self.text)
        self.assertIn("TEMPORARY_COMPATIBILITY_CERTIFICATE", self.text)
        self.assertIn("Compatibility signer SHA-256", self.text)
        self.assertNotIn("GHOSTFTP_ANDROID_KEYSTORE_BASE64", self.text)

    def test_macos_never_claims_notarization(self):
        self.assertIn("macOS-Unsigned-Validation.app.zip", self.text)
        self.assertIn("ADHOC_VALIDATION_NOT_NOTARIZED", self.text)
        self.assertNotIn("MACOS_DEVELOPER_ID_P12_BASE64", self.text)
        self.assertNotIn("APPLE_NOTARY_API_KEY", self.text)

    def test_security_boundaries_remain_explicit(self):
        self.assertIn("ANDROID_SFTP=hidden-until-strict-host-key-verification", self.text)
        self.assertIn("Runtime FTP/FTPS security checks, privacy audits and telemetry-free behavior", self.text)
        self.assertIn("No release-signing requirement is being represented as satisfied when it is not.", self.text)

    def test_release_is_immutable_and_17_files(self):
        self.assertIn("Release $tag already exists; refusing to overwrite it.", self.text)
        self.assertIn("Tag $tag already exists; refusing to move it.", self.text)
        self.assertIn("PUBLIC_PLATFORM_ARTIFACTS=14", self.text)
        self.assertIn("PUBLIC_RELEASE_FILES=17", self.text)
        self.assertIn("test \"$count\" = '17'", self.text)
        self.assertIn("COMPATIBILITY_RELEASE_PUBLISHED=PASS", self.text)


if __name__ == "__main__":
    unittest.main()
