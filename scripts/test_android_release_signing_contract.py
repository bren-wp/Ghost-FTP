#!/usr/bin/env python3
from pathlib import Path
import re
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


class AndroidReleaseSigningContractTests(unittest.TestCase):
    def test_android_release_identity_comes_from_root_version(self) -> None:
        version = read("VERSION").strip()
        self.assertRegex(version, r"^\d+\.\d+\.\d+$")
        build = read("android/app/build.gradle")
        self.assertIn("def ghostFtpVersion = rootProject.file('../VERSION').text.trim()", build)
        self.assertIn("versionName ghostFtpVersion", build)
        self.assertIn("versionCode ghostFtpVersionCode", build)
        self.assertIn("versionNameSuffix '-dev'", build)
        self.assertNotIn('versionName "${ghostFtpVersion}-dev"', build)
        self.assertIn("Ghost-FTP-Android-dev.apk", build)
        self.assertNotIn("rename { 'Ghost-FTP-Android.apk' }", build)

    def test_android_version_code_is_semver_derived_and_bounded(self) -> None:
        build = read("android/app/build.gradle")
        for marker in (
            "ghostFtpVersion.tokenize('.')",
            "def ghostFtpVersionMajor = ghostFtpVersionParts[0].toInteger()",
            "def ghostFtpVersionMinor = ghostFtpVersionParts[1].toInteger()",
            "def ghostFtpVersionPatch = ghostFtpVersionParts[2].toInteger()",
            "ghostFtpVersionMinor > 999",
            "ghostFtpVersionPatch > 999",
            "ghostFtpVersionMajor * 1000000",
            "ghostFtpVersionMinor * 1000",
            "ghostFtpVersionPatch",
            "ghostFtpVersionCode <= 0",
            "ghostFtpVersionCode > 2100000000",
        ):
            self.assertIn(marker, build)

    def test_ci_builds_release_and_exercises_real_apksigner(self) -> None:
        workflow = read(".github/workflows/android-apk.yml")
        for marker in (
            ":app:lintRelease",
            ":app:assembleRelease",
            "app-release-unsigned.apk",
            "keytool -genkeypair",
            '"$build_tools/apksigner" sign',
            '"$build_tools/apksigner" verify --verbose --print-certs',
            "ANDROID_RELEASE_SIGNING_PIPELINE_SMOKE=PASS",
            "EPHEMERAL_CI_ONLY",
            "ghostftp-android-dev-apk",
            "Ghost-FTP-Android-dev.apk",
        ):
            self.assertIn(marker, workflow)
        self.assertNotIn("Ghost-FTP-Android.apk", workflow)

    def test_production_signing_secrets_never_enter_development_workflow(self) -> None:
        workflow = read(".github/workflows/android-apk.yml")
        for secret in (
            "GHOSTFTP_ANDROID_KEYSTORE_BASE64",
            "GHOSTFTP_ANDROID_KEYSTORE_PASSWORD",
            "GHOSTFTP_ANDROID_KEY_ALIAS",
            "GHOSTFTP_ANDROID_KEY_PASSWORD",
        ):
            self.assertNotIn(secret, workflow)


if __name__ == "__main__":
    unittest.main()
