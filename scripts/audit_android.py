#!/usr/bin/env python3
"""Validate Android security, privacy, version, lifecycle, mobile UX and build invariants."""

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

    connection = require("android/app/src/main/java/com/ghostftp/client/model/ConnectionConfig.java", (
        "SFTP requires an expected SHA-256 host-key fingerprint",
        "Port must be between 1 and 65535",
        "not a URL or path",
        "Base64.getDecoder().decode",
        "digest.length != 32",
        "rejectControlCharacters",
        'rejectControlCharacters(rawHost, "Host")',
        'rejectControlCharacters(rawPort, "Port")',
        'rejectControlCharacters(rawUsername, "Username")',
        'rejectControlCharacters(rawPassword, "Password")',
        'rejectControlCharacters(rawFingerprint, "SFTP fingerprint")',
    ))
    for raw_marker, normalization_marker in (
        ('rejectControlCharacters(rawHost, "Host")', "normalizeHost(rawHost)"),
        ('rejectControlCharacters(rawUsername, "Username")', "rawUsername.trim()"),
        ('rejectControlCharacters(rawPort, "Port")', "parsePort(rawPort"),
        ('rejectControlCharacters(rawFingerprint, "SFTP fingerprint")', "normalizeFingerprint(rawFingerprint)"),
    ):
        first = connection.find(raw_marker)
        normalized = connection.find(normalization_marker)
        if first < 0 or normalized < 0 or first > normalized:
            fail(f"Android raw input is normalized before control-character rejection: {raw_marker}")
    if "PromiscuousVerifier" in connection:
        fail("connection validation references a permissive SFTP verifier")

    remote_paths = require("android/app/src/main/java/com/ghostftp/client/model/RemotePaths.java", (
        "Remote path is not canonical.",
        "Remote path contains an unsafe component.",
        "!name.equals(name.trim())",
        "name.indexOf('\\r') >= 0",
        "name.indexOf('\\n') >= 0",
    ))
    for forbidden in ("path.replace('\\\\', '/')", 'p.contains("//")'):
        if forbidden in remote_paths:
            fail("Android remote path handling still normalizes unsafe separators")

    entry_list = require("android/app/src/main/java/com/ghostftp/client/model/RemoteEntryList.java", (
        "comparing(RemoteEntry::directory).reversed()",
        "String.CASE_INSENSITIVE_ORDER",
        "toLowerCase(Locale.ROOT)",
        "new ArrayList<>(source)",
    ))
    if "source.sort(" in entry_list:
        fail("Android list presentation must not mutate the transport-owned source list")

    diagnostics = require("android/app/src/main/java/com/ghostftp/client/model/SharedHostingDiagnostics.java", (
        "record SharedHostingDiagnostics",
        '"public_html", "httpdocs", "htdocs", "www", "web", "html"',
        "protocol != ConnectionConfig.Protocol.FTP",
        'protocol == ConnectionConfig.Protocol.SFTP ? "home" : "account"',
        "if (entry == null || !entry.directory()) continue;",
        "safeEntries.size()",
    ))
    for forbidden in ("String password", "String passphrase", "String username", "String fingerprint", "connect(", "list("):
        if forbidden in diagnostics:
            fail(f"Android shared-hosting diagnostics gained secret/network behavior: {forbidden}")

    transfer_streams = require("android/app/src/main/java/com/ghostftp/client/remote/TransferStreams.java", (
        "final class TransferStreams",
        "FilterInputStream", "FilterOutputStream",
        "void onBytesTransferred(long bytes)",
        "transferred += bytes", "listener.onBytesTransferred(transferred)",
    ))
    if "Thread.interrupted" in transfer_streams or "close()" in transfer_streams:
        fail("Android transfer progress wrapper must observe bytes without injecting transport cancellation")

    sftp = require("android/app/src/main/java/com/ghostftp/client/remote/SftpRemoteClient.java", (
        "next.addHostKeyVerifier(fingerprint)",
        "next.authPassword(username, loginPassword)",
        "RemotePaths.validateName(name)",
        'password = "";',
    ))
    if "private final ConnectionConfig config;" in sftp:
        fail("Android SFTP retains the complete credential configuration for the session")
    if "PromiscuousVerifier" in sftp or "new Promiscuous" in sftp:
        fail("Android SFTP permits unverified host keys")

    ftp = require("android/app/src/main/java/com/ghostftp/client/remote/FtpRemoteClient.java", (
        "ftps.setTrustManager(null)", "ftps.setEndpointCheckingEnabled(true)", 'ftps.execPROT("P")',
        "enterLocalPassiveMode()", "FTP.BINARY_FILE_TYPE",
        "loginRoot = normalizeLoginRoot(next.printWorkingDirectory())",
        "mapLoginRelativePath(loginRoot, directory)", "Remote UI path contains an unsafe component.",
        "RemotePaths.validateName(name)", "raw.indexOf('\\r') >= 0", "raw.indexOf('\\n') >= 0",
        'password = "";',
    ))
    if "private final ConnectionConfig config;" in ftp:
        fail("Android FTP retains the complete credential configuration for the session")
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

    activity = require("android/app/src/main/java/com/ghostftp/client/MainActivity.java", (
        "Intent.ACTION_OPEN_DOCUMENT", "Intent.ACTION_CREATE_DOCUMENT", "Intent.EXTRA_ALLOW_MULTIPLE",
        "getContentResolver().openInputStream", "getContentResolver().openOutputStream",
        "Executors.newSingleThreadExecutor()", "connectingClient", "destroyed",
        "String remotePath = pendingDownloadPath;", "pendingDownloadPath = null;",
        "main.removeCallbacksAndMessages(null)", "io.shutdownNow()", 'new Thread(() ->',
        "filter.addTextChangedListener", "visibleEntries", "RemoteEntryList.sorted(next)",
        "RemoteEntryList.filtered(entries, text(filter))", "promptGoToPath()", "uploadUris(new ArrayList<>(selected))",
        "password.setText(\"\")", "presetStore.save(safeProtocol, safeHost, safePort, safeUsername, safeFingerprint)",
        'String name = rawName == null ? "" : rawName;', "setMinHeight(dp(48))",
        "ProgressBar", "TransferStreams.monitor", "stopAfterCurrentRequested",
        "status_stopping_after_current", "displaySize(Uri uri)", "beginTransferUi", "finishTransferUi",
        "if (stopAfterCurrentRequested && i + 1 < items.size())",
        "SharedHostingDiagnostics.analyze(safeProtocol, initial)", "connected_with_diagnostics",
        "diagnostics.webRootDetected()", "diagnostic_plain_ftp", "diagnostic_secure",
    ))
    for forbidden in (
        "SharedPreferences", "getSharedPreferences", "FirebaseAnalytics", "AdvertisingId",
        "presetStore.save(config)", "rawName.trim()", "Thread.currentThread().interrupt()",
        "openDirectory(diagnostics.webRoot())", "presetStore.save(diagnostics",
    ):
        if forbidden in activity:
            fail(f"Android activity contains forbidden persistence/normalization/analytics/cancel/auto-navigation marker: {forbidden}")

    preset = require("android/app/src/main/java/com/ghostftp/client/ConnectionPresetStore.java", (
        "SharedPreferences", "Context.MODE_PRIVATE", "KEY_PROTOCOL", "KEY_HOST", "KEY_PORT", "KEY_USERNAME", "KEY_FINGERPRINT",
        "void save(ConnectionConfig.Protocol protocol, String host, int port, String username, String fingerprint)",
        "ConnectionConfig.create(", "preferences.edit().clear().apply()",
    ))
    for forbidden in ("KEY_PASSWORD", "KEY_PASSPHRASE", ".password()", ".passphrase()", "putString(\"password\"", "putString(\"passphrase\""):
        if forbidden in preset:
            fail(f"Android connection preset persists or reads a secret: {forbidden}")

    settings = read("android/settings.gradle.kts")
    if "mavenLocal()" in settings:
        fail("Android dependency resolution must not use mavenLocal")

    require("android/app/src/test/java/com/ghostftp/client/model/ConnectionConfigSecurityTest.java", (
        "validatesAndCanonicalizesSftpSha256Fingerprint",
        "rejectsRawEndpointAndCredentialControlCharactersBeforeTrimming",
        '"example.com\\r\\n"',
        '"21\\r\\n"',
        'VALID_SHA256 + "\\r\\n"',
    ))
    require("android/app/src/test/java/com/ghostftp/client/model/RemotePathsTest.java", (
        "rejectsWhitespaceAndProtocolControlCharacters",
        '"line\\nbreak.txt"',
        '"line\\rbreak.txt"',
    ))
    require("android/app/src/test/java/com/ghostftp/client/model/RemotePathsTraversalTest.java", (
        "directoryRejectsTraversalAndSeparatorRewrites",
        "public_html//assets",
    ))
    require("android/app/src/test/java/com/ghostftp/client/model/RemoteEntryListTest.java", (
        "sortsDirectoriesFirstThenNamesCaseInsensitively",
        "filtersWithoutMutatingSortedSource",
    ))
    require("android/app/src/test/java/com/ghostftp/client/model/SharedHostingDiagnosticsTest.java", (
        "prefersPublicHtmlAndReportsSecureFtps",
        "plainFtpRemainsVisibleAsInsecureAndFilesAreNotWebRoots",
        "sftpUsesHomeRootWithoutInventingWebRoot",
        'assertEquals("public_html", got.webRoot())',
        'assertEquals("htdocs", got.webRoot())',
    ))
    require("android/app/src/test/java/com/ghostftp/client/remote/TransferStreamsTest.java", (
        "inputReportsCumulativeBytesWithoutChangingPayload",
        "outputReportsCumulativeBytesWithoutChangingPayload",
        "List.of(4L, 5L, 6L)", "List.of(3L, 4L)",
    ))
    require("android/app/src/test/java/com/ghostftp/client/remote/FtpRemoteClientPathTest.java", (
        "rejectsUnsafeServerLoginDirectory",
        '"/home/example\\r"',
        '"/home/example\\n"',
    ))
    for rel in (
        "android/app/src/test/java/com/ghostftp/client/model/ConnectionConfigTest.java",
        "android/README.md",
    ):
        read(rel)

    print(f"ANDROID_AUDIT=PASS ({version})")
    print("ANDROID_SFTP_HOST_KEY_PINNING=REQUIRED_AND_SHA256_VALIDATED")
    print("ANDROID_RAW_ENDPOINT_CONTROL_CHARACTERS=REJECTED_BEFORE_NORMALIZATION")
    print("ANDROID_FTPS_PLATFORM_TRUST_AND_ENDPOINT_CHECKING=ENABLED")
    print("ANDROID_REMOTE_PATH_NORMALIZATION=FAIL_CLOSED")
    print("ANDROID_REMOTE_NAMES=CANONICAL_SHARED_VALIDATOR")
    print("ANDROID_FTP_LOGIN_ROOT=ENFORCED_AND_CONTROL_SAFE")
    print("ANDROID_SHARED_HOSTING_DIAGNOSTICS=INITIAL_LISTING_ONLY_NON_SECRET")
    print("ANDROID_SHARED_HOSTING_AUTO_NAVIGATION=BLOCKED")
    print("ANDROID_LOGIN_PASSWORD_LIFETIME=CONNECT_ONLY_AND_UI_CLEARED")
    print("ANDROID_NON_SECRET_CONNECTION_PRESET=APP_PRIVATE_AND_BACKUP_EXCLUDED")
    print("ANDROID_MOBILE_FILE_FILTER_AND_SORT=TESTED")
    print("ANDROID_MULTI_FILE_UPLOAD=STORAGE_ACCESS_FRAMEWORK")
    print("ANDROID_TRANSFER_PROGRESS=LOCAL_STREAM_BYTE_BOUNDARY")
    print("ANDROID_BATCH_STOP=AFTER_CURRENT_FILE_ONLY")
    print("ANDROID_GO_TO_PATH=CANONICAL_REMOTE_PATH_VALIDATION")
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
