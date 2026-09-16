#!/usr/bin/env python3
"""Android regression suite for the maintained test and production release contract."""

import unittest

import _android_contract_regressions as _regressions


class AndroidContractTests(_regressions.AndroidContractTests):
    def test_android_project_uses_root_release_identity_without_dev_suffix(self) -> None:
        build = self.read("android/app/build.gradle")
        workflow = self.read(".github/workflows/android-apk.yml")

        for marker in (
            "rootProject.file('../VERSION').text.trim()",
            "versionCode ghostFtpVersionCode",
            "versionName ghostFtpVersion",
            "applicationIdSuffix '.debug'",
        ):
            self.assertIn(marker, build)

        for retired in (
            "versionNameSuffix '-dev'",
            "Ghost-FTP-Android-dev.apk",
            "ANDROID_DEV_APK",
            "packageGhostFtpApk",
        ):
            self.assertNotIn(retired, build)
            self.assertNotIn(retired, workflow)

        for marker in (
            ":app:testDebugUnitTest",
            ":app:lintDebug",
            ":app:lintRelease",
            ":app:assembleDebug",
            ":app:assembleRelease",
            "android/app/build/outputs/apk/debug/app-debug.apk",
            '"$build_tools/apksigner" sign',
            '"$build_tools/apksigner" verify --verbose --print-certs',
        ):
            self.assertIn(marker, workflow)

        self.assertNotIn("ghostftp-android-dev-apk", workflow)
        self.assertNotIn("Upload Android development APK", workflow)


if __name__ == "__main__":
    unittest.main()
