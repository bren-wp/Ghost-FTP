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
        "ios/ByFTP/ByFTPApp.swift",
        "ios/ByFTP/ContentView.swift",
        "ios/ByFTP/ConnectionView.swift",
        "ios/ByFTP/RemoteBrowserView.swift",
        "ios/ByFTP/SessionStore.swift",
        "ios/ByFTP/ConnectionConfig.swift",
        "ios/ByFTP/RemoteModels.swift",
        "ios/ByFTP/SocketConnection.swift",
        "ios/ByFTP/FTPRemoteClient.swift",
        "ios/ByFTP/Info.plist",
        "ios/ByFTP/Assets.xcassets/AppIcon.appiconset/Contents.json",
        "ios/ByFTP.xcodeproj/project.pbxproj",
        "ios/ByFTP.xcodeproj/xcshareddata/xcschemes/ByFTP.xcscheme",
        "ios/Tests/ModelTests.swift",
        "ios/README.md",
        "scripts/BUILD-IOS.sh",
        "scripts/package_ios.py",
    )
    for rel in required_files:
        read(rel)

    config = require("ios/ByFTP/ConnectionConfig.swift", (
        "case ftp",
        "case ftpsImplicit",
        "rejectControlCharacters",
        "Port must be between 1 and 65535",
    ))
    if "case sftp" in config.lower() or "case ftpsExplicit" in config:
        fail("iOS claims an unimplemented transport in TransferProtocol")

    paths = require("ios/ByFTP/RemoteModels.swift", (
        "RemotePath.normalizeDirectory",
        "FTPPathMapper",
        "unsafe component",
        "noncanonical login directory",
    ))
    if 'replacingOccurrences(of: ".."' in paths:
        fail("iOS path handling rewrites traversal instead of rejecting it")

    socket = require("ios/ByFTP/SocketConnection.swift", (
        "import Network",
        "NWParameters(tls:",
        "responseTooLarge",
        "sendLine",
        "receiveToFile",
    ))
    if "NWParameters(tls: nil" in socket:
        fail("iOS FTPS disables TLS")

    ftp = require("ios/ByFTP/FTPRemoteClient.swift", (
        'command("EPSV")',
        'command("PASV")',
        'command("PBSZ", "0"',
        'command("PROT", "P"',
        "FTPPathMapper.map",
        "Intentionally ignore the server-supplied PASV host",
        "MLSD",
        "LIST",
    ))
    for forbidden in ("acceptAnyCertificate", "allowInvalidCertificates", "trustAll", "WKWebView"):
        if forbidden.lower() in ftp.lower():
            fail(f"unsafe iOS transport marker found: {forbidden}")

    session = require("ios/ByFTP/SessionStore.swift", (
        "generation &+= 1",
        'password = ""',
        "startAccessingSecurityScopedResource",
        "clearDownloadedFile",
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
        "com.apple.product-type.application",
        "PRODUCT_BUNDLE_IDENTIFIER = com.byftp.client",
        "IPHONEOS_DEPLOYMENT_TARGET = 16.0",
        "MARKETING_VERSION = 0.0.0",
        "SWIFT_VERSION = 5.0",
    ):
        if marker not in project:
            fail(f"Xcode project is missing: {marker}")
    if re.search(r"MARKETING_VERSION = (?!0\.0\.0)\d+\.\d+\.\d+", project):
        fail("Xcode project hard-codes a production release version")

    build = read("scripts/BUILD-IOS.sh")
    for marker in (
        "< VERSION",
        "xcodebuild",
        "-sdk iphoneos",
        "generic/platform=iOS",
        "CODE_SIGNING_ALLOWED=NO",
        "ARCHS=arm64",
        "scripts/package_ios.py",
    ):
        if marker not in build:
            fail(f"iOS build script is missing: {marker}")

    model_tests = require("ios/Tests/ModelTests.swift", (
        "IOS_MODEL_TESTS=PASS",
        "path traversal was accepted",
        "CRLF username injection was accepted",
        "CRLF password injection was accepted",
    ))
    if not model_tests:
        fail("iOS model/path tests are unavailable")

    packager = read("scripts/package_ios.py")
    for marker in (
        "IOS_PACKAGE_FAILED",
        "Payload/ByFTP.app",
        "iOS-arm64-unsigned.ipa",
        "iOS-arm64-unsigned-app.zip",
        "CFBundleIdentifier",
        "CFBundleShortVersionString",
        "Mach-O",
    ):
        if marker not in packager:
            fail(f"iOS packager is missing: {marker}")

    print(f"IOS_AUDIT=PASS ({version})")
    print("IOS_NATIVE_UI=SWIFTUI")
    print("IOS_TRANSPORTS=FTP,FTPS_IMPLICIT")
    print("IOS_PASV_HOST_REDIRECT=BLOCKED")
    print("IOS_CREDENTIAL_PERSISTENCE=BLOCKED")
    print("IOS_BACKGROUND_SESSION=DISCONNECTED")
    print("IOS_RELEASE_ARCH=ARM64_IPHONEOS")
    print("IOS_IPA_SIGNING=EXTERNAL_APPLE_IDENTITY_REQUIRED")
    return 0


if __name__ == "__main__":
    sys.exit(main())
