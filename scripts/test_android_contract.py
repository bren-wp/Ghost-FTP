#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


class AndroidContractTests(unittest.TestCase):
    def read(self, rel: str) -> str:
        return (ROOT / rel).read_text(encoding="utf-8")

    def test_android_project_generates_named_apk_under_android(self) -> None:
        build = self.read("android/app/build.gradle")
        workflow = self.read(".github/workflows/android-apk.yml")
        for marker in (
            "tasks.register('packageGhostFtpApk', Copy)",
            "'Ghost-FTP-Android.apk'",
            "dist/Ghost-FTP-Android.apk",
            "dependsOn 'assembleDebug'",
        ):
            self.assertIn(marker, build)
        self.assertIn("android/dist/Ghost-FTP-Android.apk", workflow)
        self.assertIn("unzip -t android/dist/Ghost-FTP-Android.apk", workflow)
        self.assertIn("name: ghostftp-android-apk", workflow)

    def test_android_permissions_stay_narrow(self) -> None:
        manifest = self.read("android/app/src/main/AndroidManifest.xml")
        self.assertIn("android.permission.INTERNET", manifest)
        self.assertIn('android:allowBackup="false"', manifest)
        for forbidden in (
            "MANAGE_EXTERNAL_STORAGE",
            "READ_EXTERNAL_STORAGE",
            "WRITE_EXTERNAL_STORAGE",
            "ACCESS_FINE_LOCATION",
            "ACCESS_COARSE_LOCATION",
        ):
            self.assertNotIn(forbidden, manifest)

    def test_ftps_is_strict_and_has_no_trust_all_fallback(self) -> None:
        ftp = self.read("android/app/src/main/java/com/brendigo/ghostftp/FtpSession.java")
        for marker in (
            'command("AUTH TLS")',
            'parameters.setEndpointIdentificationAlgorithm("HTTPS")',
            'command("PBSZ 0")',
            'command("PROT P")',
            "tls.startHandshake()",
        ):
            self.assertIn(marker, ftp)
        for forbidden in (
            "X509TrustManager",
            "HostnameVerifier",
            "TrustManager[]",
            "setDefaultHostnameVerifier",
        ):
            self.assertNotIn(forbidden, ftp)

    def test_password_is_memory_only_and_storage_uses_saf(self) -> None:
        activity = self.read("android/app/src/main/java/com/brendigo/ghostftp/MainActivity.java")
        for marker in (
            "Intent.ACTION_OPEN_DOCUMENT_TREE",
            "takePersistableUriPermission",
            "password.setText(\"\")",
            'getSharedPreferences(PREFS, MODE_PRIVATE)',
        ):
            self.assertIn(marker, activity)
        self.assertNotIn('putString("password"', activity)
        self.assertNotIn('putString("passphrase"', activity)

    def test_sftp_is_fail_closed_until_host_key_verification_exists(self) -> None:
        readme = self.read("android/README.md")
        activity = self.read("android/app/src/main/java/com/brendigo/ghostftp/MainActivity.java")
        self.assertIn("SFTP is intentionally not exposed", readme)
        self.assertIn('new String[]{"FTPS", "FTP"}', activity)
        self.assertNotIn('"SFTP"', activity)

    def test_android_has_no_telemetry_or_ad_sdk_dependency(self) -> None:
        build = self.read("android/app/build.gradle")
        settings = self.read("android/settings.gradle")
        combined = (build + settings).lower()
        for forbidden in (
            "firebase",
            "analytics",
            "crashlytics",
            "appsflyer",
            "facebook",
            "admob",
            "com.google.android.gms:play-services-ads",
        ):
            self.assertNotIn(forbidden, combined)


if __name__ == "__main__":
    unittest.main()
