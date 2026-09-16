#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


class AndroidReleaseIdentityContractTests(unittest.TestCase):
    def read(self, rel: str) -> str:
        return (ROOT / rel).read_text(encoding="utf-8")

    def test_android_release_identity_tracks_repository_version_and_public_signing_boundary(self) -> None:
        version = self.read("VERSION").strip()
        gradle = self.read("android/app/build.gradle")
        activity = self.read("android/app/src/main/java/app/ghostftp/client/MainActivity.java")
        readme = self.read("android/README.md")
        uiux = self.read("android/UI-UX.md")
        release = self.read(".github/workflows/release.yml")

        self.assertTrue(version)
        self.assertIn("rootProject.file('../VERSION').text.trim()", gradle)
        self.assertIn("versionCode ghostFtpVersionCode", gradle)
        self.assertIn("versionName ghostFtpVersion", gradle)
        self.assertIn("applicationIdSuffix '.debug'", gradle)
        self.assertNotIn("versionNameSuffix '-dev'", gradle)
        self.assertNotIn('versionName "${ghostFtpVersion}-dev"', gradle)
        self.assertIn('infoLine("Version", BuildConfig.VERSION_NAME)', activity)
        self.assertNotIn('infoLine("Release status", "Repository build "', activity)
        self.assertNotIn("Repository build", activity)

        combined = "\n".join((activity, readme, uiux)).lower()
        for marker in (
            "0.0.3 remains the published windows/linux release",
            "already published ghost ftp 0.0.3 public release",
            "repository root version remains `0.0.3`",
            "post-0.0.3 development/debug-signed artifact",
            "android apk remains development-only",
        ):
            self.assertNotIn(marker, combined)

        self.assertIn("repository root `VERSION`", readme)
        self.assertIn(f"Ghost-FTP-{version}-Android.apk", readme)
        self.assertIn("production-signed", readme.lower())
        self.assertNotIn("Ghost-FTP-Android-dev.apk", readme)
        self.assertIn("BuildConfig.VERSION_NAME", uiux)
        self.assertIn("public Android release", uiux)
        self.assertIn("SFTP", readme)
        self.assertIn("intentionally not exposed", readme)

        self.assertIn(f"Ghost-FTP-${{VERSION}}-Android.apk", release)
        self.assertIn("GHOSTFTP_ANDROID_KEYSTORE_BASE64", release)
        self.assertIn("GHOSTFTP_ANDROID_CERT_SHA256", release)
        self.assertNotIn("GHOSTFTP_ANDROID_SIGNER_SHA256", release)
        self.assertIn("apksigner", release)
        self.assertNotIn("ghostftp-ci-smoke", release)


if __name__ == "__main__":
    unittest.main()
