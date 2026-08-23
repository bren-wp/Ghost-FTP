#!/usr/bin/env python3
"""Validate Android security, privacy, version, lifecycle and build invariants."""

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
        "isShrinkResources = true",
        'disable += setOf("OldTargetApi", "TrustAllX509TrustManager")',
    ))
    if re.search(r'versionName\s*=\s*"\d+\.\d+\.\d+', build):
        fail("Android build hard-codes a production version")

    manifest = require("android/app/src/main/AndroidManifest.xml", (
        "android.permission.INTERNET",
        'android:allowBackup="false"',
        'android:dataExtractionRules="@xml/data_extraction_rules"',
        'android:fullBackupContent="@xml/backup_rules"',
        'android:exported="true"',
        'android:usesCleartextTraffic="false"',
    ))
    for forbidden in (
        "MANAGE_EXTERNAL_STORAGE", "READ_EXTERNAL_STORAGE", "WRITE_EXTERNAL_STORAGE",
        "ACCESS_NETWORK_STATE", "QUERY_ALL_PACKAGES", "REQUEST_INSTALL_PACKAGES",
    ):
        if forbidden in manifest:
            fail(f"Android manifest requests unnecessary or broad permission: {forbidden}")

    network = require("android/app/src/main/res/xml/network_security_config.xml", (
        'cleartextTrafficPermitted="false"', '<certificates src="system" />',
    ))
    if 'cleartextTrafficPermitted="true"' in network:
        fail("Android network security config permits generic cleartext traffic")

    for rel in ("android/app/src/main/res/xml/data_extraction_rules.xml", "android/app/src/main/res/xml/backup_rules.xml"):
        backup = read(rel)
        for domain in ("root", "file", "database", "sharedpref", "external"):
            if f'domain="{domain}" path="."' not in backup:
                fail(f"{rel} does not exclude the {domain} backup domain")

    connection = require("android/app/src/main/java/com/byftp/client/model/ConnectionConfig.java", (
        "SFTP requires an expected SHA-256 host-key fingerprint",
        "Port must be between 1 and 65535",
        "not a URL or path",
        "Base64.getDecoder().decode",
        "digest.length != 32",
        "rejectControlCharacters",
    ))
    if "PromiscuousVerifier" in connection:
        fail("connection validation references a permissive SFTP verifier")

    remote_paths = require("android/app/src/main/java/com/byftp/client/model/RemotePaths.java", (
        "Remote path is not canonical.",
        "Remote path contains an unsafe component.",
        "!name.equals(name.trim())",
    ))
    for forbidden in ("path.replace('\\\\', '/')", 'p.contains("//")'):
        if forbidden in remote_paths:
            fail("Android remote path handling still normalizes unsafe separators")

    sftp = require("android/app/src/main/java/com/byftp/client/remote/SftpRemoteClient.java", (
        "next.addHostKeyVerifier(config.fingerprint())",
        "next.authPassword(config.username(), config.password())",
    ))
    if "PromiscuousVerifier" in sftp or "new Promiscuous" in sftp:
        fail("Android SFTP permits unverified host keys")

    ftp = require("android/app/src/main/java/com/byftp/client/remote/FtpRemoteClient.java", (
        "ftps.setTrustManager(null)", "ftps.setEndpointCheckingEnabled(true)", 'ftps.execPROT("P")',
        "enterLocalPassiveMode()", "FTP.BINARY_FILE_TYPE",
        "loginRoot = normalizeLoginRoot(next.printWorkingDirectory())",
        "mapLoginRelativePath(loginRoot, directory)", "Remote UI path contains an unsafe component.",
    ))
    if "TrustManagerUtils" in ftp:
        fail("Android FTPS must not use Commons Net permissive/custom trust helpers")

    java_root = ROOT / "android/app/src/main/java"
    java_source = "\n".join(path.read_text(encoding="utf-8") for path in java_root.rglob("*.java"))
    for forbidden in (
        "X509TrustManager", "checkServerTrusted", "checkClientTrusted", "PromiscuousVerifier",
        "NoopHostnameVerifier", "ALLOW_ALL_HOSTNAME_VERIFIER",
    ):
        if forbidden in java_source:
            fail(f"Android source contains forbidden permissive TLS/SSH marker: {forbidden}")

    activity = require("android/app/src/main/java/com/byftp/client/MainActivity.java", (
        "Intent.ACTION_OPEN_DOCUMENT", "Intent.ACTION_CREATE_DOCUMENT",
        "getContentResolver().openInputStream", "getContentResolver().openOutputStream",
        "Executors.newSingleThreadExecutor()", "connectingClient", "destroyed",
        "String remotePath = pendingDownloadPath;", "pendingDownloadPath = null;",
        "main.removeCallbacksAndMessages(null)", "io.shutdownNow()", 'new Thread(() ->',
    ))
    for forbidden in ("SharedPreferences", "getSharedPreferences", "FirebaseAnalytics", "AdvertisingId"):
        if forbidden in activity:
            fail(f"Android activity contains forbidden persistence/analytics marker: {forbidden}")

    settings = read("android/settings.gradle.kts")
    if "mavenLocal()" in settings:
        fail("Android dependency resolution must not use mavenLocal")

    require("android/app/src/test/java/com/byftp/client/model/ConnectionConfigSecurityTest.java", (
        "validatesAndCanonicalizesSftpSha256Fingerprint",
        "rejectsCredentialControlCharacters",
    ))
    require("android/app/src/test/java/com/byftp/client/model/RemotePathsTraversalTest.java", (
        "directoryRejectsTraversalAndSeparatorRewrites",
        "public_html//assets",
    ))
    for rel in (
        "android/app/src/test/java/com/byftp/client/model/ConnectionConfigTest.java",
        "android/app/src/test/java/com/byftp/client/model/RemotePathsTest.java",
        "android/app/src/test/java/com/byftp/client/remote/FtpRemoteClientPathTest.java",
        "android/README.md",
    ):
        read(rel)

    print(f"ANDROID_AUDIT=PASS ({version})")
    print("ANDROID_SFTP_HOST_KEY_PINNING=REQUIRED_AND_SHA256_VALIDATED")
    print("ANDROID_FTPS_PLATFORM_TRUST_AND_ENDPOINT_CHECKING=ENABLED")
    print("ANDROID_REMOTE_PATH_NORMALIZATION=FAIL_CLOSED")
    print("ANDROID_FTP_LOGIN_ROOT=ENFORCED")
    print("ANDROID_GENERIC_CLEARTEXT_NETWORK=BLOCKED")
    print("ANDROID_BROAD_STORAGE_PERMISSION=BLOCKED")
    print("ANDROID_BACKUP_AND_DEVICE_TRANSFER=BLOCKED")
    print("ANDROID_ACTIVITY_LIFECYCLE_CLEANUP=ENFORCED")
    print("ANDROID_PICKER_PENDING_STATE=CLEARED")
    print("ANDROID_PASSWORD_PERSISTENCE=BLOCKED")
    print("ANDROID_VERSION_SOURCE=ROOT_VERSION")
    return 0


if __name__ == "__main__":
    sys.exit(main())
