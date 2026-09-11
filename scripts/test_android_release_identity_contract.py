#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


class AndroidReleaseIdentityContractTests(unittest.TestCase):
    def read(self, rel: str) -> str:
        return (ROOT / rel).read_text(encoding="utf-8")

    def test_android_release_identity_tracks_repository_version_without_claiming_public_android_release(self) -> None:
        version = self.read("VERSION").strip()
        gradle = self.read("android/app/build.gradle")
        activity = self.read("android/app/src/main/java/app/ghostftp/client/MainActivity.java")
        readme = self.read("android/README.md")
        uiux = self.read("android/UI-UX.md")

        self.assertTrue(version)
        self.assertIn("rootProject.file('../VERSION').text.trim()", gradle)
        self.assertIn('versionName "${ghostFtpVersion}-dev"', gradle)
        self.assertIn(
            'infoLine("Release status", "Repository build " + BuildConfig.VERSION_NAME',
            activity,
        )
        self.assertIn("Android APK remains development-only", activity)

        stale_markers = (
            "0.0.3 remains the published Windows/Linux release",
            "already published Ghost FTP 0.0.3 public release",
            "repository root version remains `0.0.3`",
            "post-0.0.3 development/debug-signed artifact",
        )
        combined = "\n".join((activity, readme, uiux)).lower()
        for marker in stale_markers:
            self.assertNotIn(marker.lower(), combined)

        self.assertIn("repository root `VERSION`", readme)
        self.assertIn("development/debug-signed APK", readme)
        self.assertIn("BuildConfig.VERSION_NAME", uiux)
        self.assertIn("public Android release", uiux)


if __name__ == "__main__":
    unittest.main()
