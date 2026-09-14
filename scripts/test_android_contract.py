#!/usr/bin/env python3
"""Android regression suite with the current development/release packaging contract."""

import unittest

import _android_contract_regressions as _regressions


class AndroidContractTests(_regressions.AndroidContractTests):
    def test_android_project_generates_named_apk_under_android(self) -> None:
        build = self.read("android/app/build.gradle")
        workflow = self.read(".github/workflows/android-apk.yml")

        for marker in (
            "rootProject.file('../VERSION').text.trim()",
            "versionCode ghostFtpVersionCode",
            "versionName ghostFtpVersion",
            "versionNameSuffix '-dev'",
            "tasks.register('packageGhostFtpApk', Copy)",
            "'Ghost-FTP-Android-dev.apk'",
            "dist/Ghost-FTP-Android-dev.apk",
            "dependsOn 'assembleDebug'",
        ):
            self.assertIn(marker, build)

        self.assertNotIn('versionName "${ghostFtpVersion}-dev"', build)
        self.assertNotIn("rename { 'Ghost-FTP-Android.apk' }", build)

        for marker in (
            "android/dist/Ghost-FTP-Android-dev.apk",
            "unzip -t android/dist/Ghost-FTP-Android-dev.apk",
            "name: ghostftp-android-dev-apk",
            ":app:lintRelease",
            ":app:assembleRelease",
            '"$build_tools/apksigner" sign',
            '"$build_tools/apksigner" verify --verbose --print-certs',
        ):
            self.assertIn(marker, workflow)
        self.assertNotIn("android/dist/Ghost-FTP-Android.apk", workflow)


if __name__ == "__main__":
    unittest.main()
