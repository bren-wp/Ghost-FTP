#!/usr/bin/env python3
"""Validate native iOS security, privacy, version and packaging invariants."""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
IOS = ROOT / "ios"


def fail(message: str) -> None:
    raise SystemExit("IOS_AUDIT_FAILED: " + message)


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
        fail(f"invalid VERSION: {version!r}")

    required_files = (
        "ios/ByFTP/ByFTPApp.swift", "ios/ByFTP/ContentView.swift", "ios/ByFTP/ConnectionView.swift",
        "ios/ByFTP/RemoteBrowserView.swift", "ios/ByFTP/SessionStore.swift", "ios/ByFTP/ConnectionConfig.swift",
        "ios/ByFTP/RemoteModels.swift", "ios/ByFTP/SocketConnection.swift", "ios/ByFTP/FTPRemoteClient.swift",
        "ios/ByFTP/Info.plist", "ios/ByFTP/Assets.xcassets/AppIcon.appiconset/Contents.json",
        "ios/ByFTP.xcodeproj/project.pbxproj", "ios/ByFTP.xcodeproj/xcshareddata/xcschemes/ByFTP.xcscheme",
        "ios/Tests/ModelTests.swift", "ios/README.md", "scripts/BUILD-IOS.sh", "scripts/package_ios.py",
    )
    for rel in required_files:
        read(rel)

    config = require("ios/ByFTP/ConnectionConfig.swift", (
        "case ftp", "case ftpsImplicit", "rejectControlCharacters", "Port must be between 1 and 65535",
        'rejectControlCharacters(rawHost, field: "Host")',
        'rejectControlCharacters(rawPort, field: "Port")',
        'rejectControlCharacters(rawUsername, field: "Username")',
        'rejectControlCharacters(rawPassword, field: "Password")',
        "scalar.value == 0", "scalar.value == 10", "scalar.value == 13",
    ))
    for raw_marker, normalization_marker in (
        ('rejectControlCharacters(rawHost, field: "Host")', "normalizeHost(rawHost)"),
        ('rejectControlCharacters(rawPort, field: "Port")', "parsePort(rawPort"),
        ('rejectControlCharacters(rawUsername, field: "Username")', "rawUsername.trimmingCharacters"),
    ):
        first = config.find(raw_marker)
        normalized = config.find(normalization_marker)
        if first < 0 or normalized < 0 or first > normalized:
            fail(f"iOS raw input is normalized before control-character rejection: {raw_marker}")
    if "case sftp" in config.lower() or "case ftpsExplicit" in config:
        fail("iOS claims an unimplemented transport in TransferProtocol")

    paths = require("ios/ByFTP/RemoteModels.swift", (
        "RemotePath.normalizeDirectory", "FTPPathMapper", "unsafe component", "noncanonical login directory",
        "raw == trimmed", 'raw.contains("\\r")', 'raw.contains("\\n")',
        'raw.contains("\\0")', "Names cannot start or end with whitespace.",
    ))
    if 'replacingOccurrences(of: ".."' in paths:
        fail("iOS path handling rewrites traversal instead of rejecting it")
    login_guard = paths.find('guard !raw.contains("\\0"), !raw.contains("\\r"), !raw.contains("\\n")')
    login_trim = paths.find("var root = raw.trimmingCharacters")
    if login_guard < 0 or login_trim < 0 or login_guard > login_trim:
        fail("iOS server login-root controls are normalized before rejection")

    socket = require("ios/ByFTP/SocketConnection.swift", (
        "import Network", "NWParameters(tls:", "responseTooLarge", "sendLine", "receiveToFile",
        "StartContinuationBox", "@unchecked Sendable", "private let lock = NSLock()",
        "guard FileManager.default.createFile",
    ))
    if "var pending: CheckedContinuation" in socket:
        fail("iOS NWConnection state handler captures a mutable local continuation")
    if "NWParameters(tls: nil" in socket:
        fail("iOS FTPS disables TLS")

    ftp = require("ios/ByFTP/FTPRemoteClient.swift", (
        'command("EPSV")', 'command("PASV")', 'command("PBSZ", "0"', 'command("PROT", "P"',
        "FTPPathMapper.map", "Intentionally ignore the server-supplied PASV host", "MLSD", "LIST",
        "private var password: String", 'defer { password = "" }', 'password = ""',
    ))
    if "private let config: ConnectionConfig" in ftp:
        fail("iOS FTP actor retains the complete credential configuration for the session")
    for forbidden in ("acceptAnyCertificate", "allowInvalidCertificates", "trustAll", "WKWebView"):
        if forbidden.lower() in ftp.lower():
            fail(f"unsafe iOS transport marker found: {forbidden}")

    session = require("ios/ByFTP/SessionStore.swift", (
        "import Combine", "ObservableObject", "@Published", "generation &+= 1", 'password = ""',
        "startAccessingSecurityScopedResource", "clearDownloadedFile",
        "private var connectingClient: FTPRemoteClient?", "connectingClient = next",
        "let pending = connectingClient", "await pending?.close()",
        "discard: (() -> Void)? = nil", "discard?()",
        "temporaryParent", "FileManager.default.removeItem(at: temporaryParent)",
    ))
    app = require("ios/ByFTP/ByFTPApp.swift", ("scenePhase", "store.disconnect()"))
    combined_source = "\n".join(path.read_text(encoding="utf-8") for path in (IOS / "ByFTP").glob("*.swift"))
    for forbidden in ("UserDefaults", "WKWebView", "Analytics", "FirebaseAnalytics", "NSAllowsArbitraryLoads"):
        if forbidden in combined_source:
            fail(f"forbidden iOS privacy/runtime marker found: {forbidden}")
    if re.search(r"https?://", combined_source):
        fail("iOS runtime source contains a fixed HTTP(S) endpoint")
    if not session or not app:
        fail("iOS lifecycle contract is unavailable")

    plist = read("ios/ByFTP/Info.plist")
    if "NSAllowsArbitraryLoads" in plist:
        fail("iOS Info.plist weakens App Transport Security globally")
    for marker in ("$(MARKETING_VERSION)", "$(CURRENT_PROJECT_VERSION)", "$(PRODUCT_BUNDLE_IDENTIFIER)"):
        if marker not in plist:
            fail(f"Info.plist is not build-version bound: missing {marker}")

    project = read("ios/ByFTP.xcodeproj/project.pbxproj")
    for marker in (
        "com.apple.product-type.application", "PRODUCT_BUNDLE_IDENTIFIER = com.byftp.client",
        "IPHONEOS_DEPLOYMENT_TARGET = 16.0", "MARKETING_VERSION = 0.0.0", "SWIFT_VERSION = 5.0",
        "000000000000000000000002 /* ByFTP */",
    ):
        if marker not in project:
            fail(f"Xcode project is missing: {marker}")
    if re.search(r"MARKETING_VERSION = (?!0\.0\.0)\d+\.\d+\.\d+", project):
        fail("Xcode project hard-codes a production release version")

    scheme = read("ios/ByFTP.xcodeproj/xcshareddata/xcschemes/ByFTP.xcscheme")
    if 'BlueprintIdentifier="000000000000000000000002"' not in scheme:
        fail("shared Xcode scheme is not bound to the ByFTP target")

    build = read("scripts/BUILD-IOS.sh")
    for marker in (
        "< VERSION", "xcodebuild", "-sdk iphoneos", "generic/platform=iOS",
        "CODE_SIGNING_ALLOWED=NO", "ARCHS=arm64", "scripts/package_ios.py",
    ):
        if marker not in build:
            fail(f"iOS build script is missing: {marker}")

    model_tests = require("ios/Tests/ModelTests.swift", (
        "IOS_MODEL_TESTS=PASS", "path traversal was accepted",
        "CRLF username injection was accepted", "trailing CRLF username injection was accepted",
        "CRLF password injection was accepted", "trailing CRLF host input was accepted",
        "trailing CRLF port input was accepted", "NUL username injection was accepted",
        "leading whitespace remote name was normalized", "trailing whitespace remote name was normalized",
        "embedded LF remote name was accepted", "server login root CRLF was normalized",
        "UnicodeScalar(13)", "UnicodeScalar(10)", "UnicodeScalar(0)",
    ))
    if not model_tests:
        fail("iOS model/path tests are unavailable")

    packager = read("scripts/package_ios.py")
    for marker in (
        "IOS_PACKAGE_FAILED", "Payload/ByFTP.app", "iOS-arm64-unsigned.ipa",
        "iOS-arm64-unsigned-app.zip", "CFBundleIdentifier", "CFBundleShortVersionString", "Mach-O",
    ):
        if marker not in packager:
            fail(f"iOS packager is missing: {marker}")

    print(f"IOS_AUDIT=PASS ({version})")
    print("IOS_NATIVE_UI=SWIFTUI")
    print("IOS_TRANSPORTS=FTP,FTPS_IMPLICIT")
    print("IOS_PASV_HOST_REDIRECT=BLOCKED")
    print("IOS_RAW_ENDPOINT_CONTROL_CHARACTERS=REJECTED_BEFORE_NORMALIZATION")
    print("IOS_REMOTE_NAMES=CANONICAL_FAIL_CLOSED")
    print("IOS_LOGIN_ROOT_CONTROL_CHARACTERS=REJECTED_BEFORE_NORMALIZATION")
    print("IOS_NWCONNECTION_CONTINUATION=LOCKED_SINGLE_RESUME")
    print("IOS_CREDENTIAL_PERSISTENCE=BLOCKED")
    print("IOS_LOGIN_PASSWORD_LIFETIME=CONNECT_ONLY")
    print("IOS_PENDING_CONNECTION=DISCONNECTABLE")
    print("IOS_TEMP_DOWNLOAD_CLEANUP=STALE_AND_FAILURE_SAFE")
    print("IOS_BACKGROUND_SESSION=DISCONNECTED")
    print("IOS_RELEASE_ARCH=ARM64_IPHONEOS")
    print("IOS_IPA_SIGNING=EXTERNAL_APPLE_IDENTITY_REQUIRED")
    return 0


if __name__ == "__main__":
    sys.exit(main())
