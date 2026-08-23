#!/usr/bin/env python3
"""Validate Android security, privacy, version and build invariants."""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def fail(message: str) -> None:
    raise SystemExit("ANDROID_AUDIT_FAILED: " + message)


def read(rel: str) -> str:
    path = ROOT / rel
    if not path.is_file():
        fail(f"missing required file: {rel}")
    return path.read_text(encoding="utf-8")


def require(rel: str, markers: tuple[str, ...]) -> str:
    text = read(rel)
    for marker in markers:
        if marker not in text:
            fail(f"{rel} is missing required marker: {marker}")
    return text


def main() -> int:
    version = read("VERSION").strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        fail("root VERSION is not semantic")

    build = require("android/app/build.gradle.kts", (
        'rootProject.file("../VERSION")',
        "versionName = canonicalVersion",
        "versionCode = canonicalVersionCode",
        "compileSdk = 37",
        "targetSdk = 37",
        "minSdk = 26",
        'commons-net:commons-net:3.13.0',
        'com.hierynomus:sshj:0.40.0',
        "warningsAsErrors = true",
    ))
    if re.search(r'versionName\s*=\s*"\d+\.\d+\.\d+', build):
        fail("Android build hard-codes a production version")

    manifest = require("android/app/src/main/AndroidManifest.xml", (
        "android.permission.INTERNET",
        'android:allowBackup="false"',
        'android:exported="true"',
    ))
    for forbidden in (
        "MANAGE_EXTERNAL_STORAGE",
        "READ_EXTERNAL_STORAGE",
        "WRITE_EXTERNAL_STORAGE",
        "QUERY_ALL_PACKAGES",
        "REQUEST_INSTALL_PACKAGES",
    ):
        if forbidden in manifest:
            fail(f"Android manifest requests forbidden broad permission: {forbidden}")

    connection = require("android/app/src/main/java/com/byftp/client/model/ConnectionConfig.java", (
        "SFTP requires an expected SHA-256 host-key fingerprint",
        "Port must be between 1 and 65535",
        "not a URL or path",
    ))
    if "PromiscuousVerifier" in connection:
        fail("connection validation references a permissive SFTP verifier")

    sftp = require("android/app/src/main/java/com/byftp/client/remote/SftpRemoteClient.java", (
        "next.addHostKeyVerifier(config.fingerprint())",
        "next.authPassword(config.username(), config.password())",
    ))
    if "PromiscuousVerifier" in sftp or "new Promiscuous" in sftp:
        fail("Android SFTP permits unverified host keys")

    ftp = require("android/app/src/main/java/com/byftp/client/remote/FtpRemoteClient.java", (
        "setEndpointCheckingEnabled(true)",
        'ftps.execPROT("P")',
        "enterLocalPassiveMode()",
        "FTP.BINARY_FILE_TYPE",
    ))
    if "TrustManager" in ftp or "TrustManagerUtils" in ftp:
        fail("Android FTPS overrides normal certificate trust")

    activity = require("android/app/src/main/java/com/byftp/client/MainActivity.java", (
        "Intent.ACTION_OPEN_DOCUMENT",
        "Intent.ACTION_CREATE_DOCUMENT",
        "getContentResolver().openInputStream",
        "getContentResolver().openOutputStream",
        "Executors.newSingleThreadExecutor()",
    ))
    for forbidden in ("SharedPreferences", "getSharedPreferences", "FirebaseAnalytics", "AdvertisingId"):
        if forbidden in activity:
            fail(f"Android activity contains forbidden persistence/analytics marker: {forbidden}")

    settings = read("android/settings.gradle.kts")
    if "mavenLocal()" in settings:
        fail("Android dependency resolution must not use mavenLocal")

    for rel in (
        "android/app/src/test/java/com/byftp/client/model/ConnectionConfigTest.java",
        "android/app/src/test/java/com/byftp/client/model/RemotePathsTest.java",
        "android/README.md",
    ):
        read(rel)

    print(f"ANDROID_AUDIT=PASS ({version})")
    print("ANDROID_SFTP_HOST_KEY_PINNING=REQUIRED")
    print("ANDROID_FTPS_ENDPOINT_CHECKING=ENABLED")
    print("ANDROID_BROAD_STORAGE_PERMISSION=BLOCKED")
    print("ANDROID_PASSWORD_PERSISTENCE=BLOCKED")
    print("ANDROID_VERSION_SOURCE=ROOT_VERSION")
    return 0


if __name__ == "__main__":
    sys.exit(main())
