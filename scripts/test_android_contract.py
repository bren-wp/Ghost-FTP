#!/usr/bin/env python3
from __future__ import annotations

import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class AndroidContractTests(unittest.TestCase):
    def read(self, rel: str) -> str:
        return (ROOT / rel).read_text(encoding="utf-8")

    def test_manifest_is_native_private_and_has_only_network_permission(self) -> None:
        manifest = self.read("android/app/src/main/AndroidManifest.xml")
        self.assertIn('android.permission.INTERNET', manifest)
        self.assertIn('android:allowBackup="false"', manifest)
        self.assertIn('android:fullBackupContent="false"', manifest)
        self.assertIn('android:usesCleartextTraffic="true"', manifest)
        self.assertNotIn('READ_EXTERNAL_STORAGE', manifest)
        self.assertNotIn('WRITE_EXTERNAL_STORAGE', manifest)
        self.assertNotIn('MANAGE_EXTERNAL_STORAGE', manifest)

    def test_sftp_host_key_policy_is_tofu_then_fail_closed_on_change(self) -> None:
        source = self.read("android/app/src/main/java/com/brendigo/ghostftp/protocol/SftpClient.java")
        build = self.read("android/app/build.gradle")
        self.assertIn("com.github.mwiede:jsch:2.28.7", build)
        for marker in (
            'new File(context.getFilesDir(), "known_hosts")',
            'next.setConfig("StrictHostKeyChecking", "ask")',
            "HostKeyRepository.CHANGED",
            "prompt.reportChangedHostKey",
            "return false;",
            "HostKeyRepository.NOT_INCLUDED",
            "prompt.confirmNewHostKey",
            'MessageDigest.getInstance("SHA-256")',
        ):
            self.assertIn(marker, source)

    def test_ftps_requires_tls_hostname_verification_and_protected_data_channel(self) -> None:
        source = self.read("android/app/src/main/java/com/brendigo/ghostftp/protocol/FtpFtpsClient.java")
        for marker in (
            'command("AUTH TLS")',
            'parameters.setEndpointIdentificationAlgorithm("HTTPS")',
            'command("PBSZ 0")',
            'command("PROT P")',
            'data = wrapTls(data, spec.host(), data.getPort())',
            'new InetSocketAddress(peerAddress, port)',
        ):
            self.assertIn(marker, source)
        self.assertNotIn("TrustAll", source)
        self.assertNotIn("ALLOW_ALL", source)

    def test_ui_uses_saf_and_cleans_failed_download_without_storage_permission(self) -> None:
        source = self.read("android/app/src/main/java/com/brendigo/ghostftp/MainActivity.java")
        for marker in (
            "Intent.ACTION_OPEN_DOCUMENT",
            "Intent.ACTION_CREATE_DOCUMENT",
            "getContentResolver().openInputStream(uri)",
            'getContentResolver().openOutputStream(uri, "w")',
            "DocumentsContract.deleteDocument",
            "entry.isSymlink()",
            "upload_conflict",
        ):
            self.assertIn(marker, source)

    def test_android_tree_has_no_tracking_or_ad_sdk(self) -> None:
        searchable = "\n".join(
            self.read(rel).lower()
            for rel in (
                "android/app/build.gradle",
                "android/app/src/main/AndroidManifest.xml",
                "android/app/src/main/java/com/brendigo/ghostftp/MainActivity.java",
                "android/app/src/main/java/com/brendigo/ghostftp/protocol/FtpFtpsClient.java",
                "android/app/src/main/java/com/brendigo/ghostftp/protocol/SftpClient.java",
            )
        )
        for forbidden in (
            "firebase-analytics",
            "firebase-crashlytics",
            "com.google.android.gms:play-services-ads",
            "appsflyer",
            "adjust-sdk",
            "facebook-android-sdk",
        ):
            self.assertNotIn(forbidden, searchable)

    def test_build_contract_generates_apk_under_android_and_uploads_same_tree(self) -> None:
        build = self.read("android/BUILD.sh")
        workflow = self.read(".github/workflows/android.yml")
        readme = self.read("android/README.md")
        marker = "android/dist/Ghost-FTP-Android-debug.apk"
        self.assertIn("dist/Ghost-FTP-Android-debug.apk", build)
        self.assertIn(marker, workflow)
        self.assertIn(marker, readme)
        self.assertIn("ghostftp-android-apk", workflow)
        self.assertIn("aapt dump badging", workflow)
        self.assertIn("sha256sum -c", workflow)


if __name__ == "__main__":
    unittest.main()
