#!/usr/bin/env python3
"""Validate the ByFTP production release workflow and publication contract."""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def fail(message: str) -> None:
    raise SystemExit("RELEASE_AUDIT_FAILED: " + message)


def read(rel: str) -> str:
    path = ROOT / rel
    if not path.is_file():
        fail(f"missing required file: {rel}")
    return path.read_text(encoding="utf-8")


def require(text: str, marker: str, where: str) -> None:
    if marker not in text:
        fail(f"{where} is missing required marker: {marker}")


def main() -> int:
    version = read("VERSION").strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        fail(f"invalid VERSION: {version!r}")

    workflow = read(".github/workflows/release.yml")
    for marker in (
        "group: byftp-release",
        "quality:", "windows:", "linux:", "macos:", "android:", "ios:", "publish:",
        "needs: [quality, windows, linux, macos, android, ios]",
        "go telemetry off", "go test ./...", "go test -race ./...", "go vet ./...",
        "go-version: '1.27.0'", "gradle-version: '9.7.0'",
        "python scripts/audit_localization.py", "python scripts/audit_version.py",
        "python scripts/audit_android.py", "python scripts/audit_ios.py",
        "python scripts/audit_docs.py", "python scripts/audit_security.py",
        "python scripts/audit_privacy.py", "python scripts/audit_release.py",
        ".\\BUILD-WINDOWS.ps1", "bash linux/BUILD.sh", "bash macos/BUILD.sh", "bash ios/BUILD.sh",
        "android-37*", "build-tools/36.0.0",
        ":app:testDebugUnitTest", ":app:lintDebug", ":app:lintRelease", ":app:assembleDebug", ":app:assembleRelease",
        "python scripts/package_android.py", "byftp-android-release", "byftp-ios-release",
        "python scripts/verify_bundle.py $zip --version $version --arch $arch",
        "scripts\\publish_release.ps1", "Verify public release staging",
        "ByFTP-$env:VERSION-Windows-x64.zip", "ByFTP-$env:VERSION-Windows-x86.zip",
        "ByFTP-$env:VERSION-Linux-amd64.deb", "ByFTP-$env:VERSION-Linux-arm64.deb", "ByFTP-$env:VERSION-Linux-i386.deb",
        "ByFTP-$env:VERSION-macOS-Universal.pkg",
        "ByFTP-$env:VERSION-Android-debug.apk", "ByFTP-$env:VERSION-Android-release-unsigned.apk",
        "ByFTP-$env:VERSION-iOS-arm64-unsigned.ipa", "ByFTP-$env:VERSION-iOS-arm64-unsigned-app.zip",
        "ANDROID=debug-signed,release-unsigned", "IOS=arm64-unsigned-ipa,arm64-unsigned-app-zip",
        "<PackageId>ByFTP.Windows</PackageId>", "dotnet nuget push", "--skip-duplicate",
    ):
        require(workflow, marker, ".github/workflows/release.yml")

    for legacy in ("scripts/BUILD-LINUX.sh", "scripts/BUILD-MACOS.sh", "scripts/BUILD-IOS.sh", ".github/workflows/__byftp_sync.yml"):
        if (ROOT / legacy).exists():
            fail(f"obsolete release/build surface still exists: {legacy}")

    publisher = read("scripts/publish_release.ps1")
    for marker in (
        "function Invoke-GhJson", "function Try-GhJson", "@('api',", "gh release create", "gh release edit",
        "gh release upload", "Get-FileHash", "SHA256", "Assert-TagCommit", "Assert-RemoteAsset",
        "RELEASE_PUBLISH_VERIFICATION=PASS",
    ):
        require(publisher, marker, "scripts/publish_release.ps1")

    android_packager = read("scripts/package_android.py")
    for marker in (
        "ANDROID_PACKAGE_FAILED", "AndroidManifest.xml", "classes.dex", "resources.arsc",
        "Android-debug.apk", "Android-release-unsigned.apk", "validate_apk",
    ):
        require(android_packager, marker, "scripts/package_android.py")

    ios_packager = read("scripts/package_ios.py")
    for marker in (
        "IOS_PACKAGE_FAILED", "Payload/ByFTP.app", "iOS-arm64-unsigned.ipa",
        "iOS-arm64-unsigned-app.zip", "CFBundleIdentifier", "CFBundleShortVersionString",
        "Mach-O", "symlink",
    ):
        require(ios_packager, marker, "scripts/package_ios.py")

    linux_build = read("linux/BUILD.sh")
    for marker in ("VERSION", "linux/byftp.desktop", "linux/debian/control.in", "dpkg-deb", "build_arch amd64 amd64", "build_arch arm64 arm64", "build_arch 386 i386"):
        require(linux_build, marker, "linux/BUILD.sh")
    read("linux/byftp.desktop")
    read("linux/debian/control.in")
    read("linux/README.md")

    macos_build = read("macos/BUILD.sh")
    for marker in ("VERSION", "macos/Info.plist.in", "macos/launcher.zsh", "lipo -create", "pkgbuild", "ByFTP-${VERSION}-macOS-Universal.pkg"):
        require(macos_build, marker, "macos/BUILD.sh")
    for rel in ("macos/Info.plist.in", "macos/launcher.zsh", "macos/README.md"):
        read(rel)

    ios_build = read("ios/BUILD.sh")
    for marker in ("VERSION", "xcodebuild", "generic/platform=iOS", "ARCHS=arm64", "scripts/package_ios.py"):
        require(ios_build, marker, "ios/BUILD.sh")

    require(read("BUILD-WINDOWS.ps1"), "VERSION", "BUILD-WINDOWS.ps1")

    verifier = read("scripts/verify_bundle.py")
    for marker in ("BUNDLE_VERIFICATION_FAILED", "BUNDLE-SHA256.txt", "Documentation/SECURITY.md"):
        require(verifier, marker, "scripts/verify_bundle.py")

    for rel in (
        "README.md", "CHANGELOG.md", "linux/README.md", "macos/README.md", "android/README.md", "ios/README.md",
        "docs/INSTALLATION.md", "docs/RELEASE-VERIFICATION.md", "docs/SECURITY.md", "docs/PRIVACY.md",
    ):
        read(rel)

    for rel in (
        ".github/workflows/release.yml", "scripts/publish_release.ps1", "scripts/verify_bundle.py",
        "scripts/package_android.py", "scripts/package_ios.py", "linux/BUILD.sh", "macos/BUILD.sh", "ios/BUILD.sh",
    ):
        if "brendigo" in read(rel).lower():
            fail(f"legacy branding remains in release surface: {rel}")

    print(f"RELEASE_AUDIT=PASS ({version})")
    print("RELEASE_MATRIX=WINDOWS_X64_X86,LINUX_AMD64_ARM64_I386,MACOS_UNIVERSAL,ANDROID_DEBUG_AND_UNSIGNED_RELEASE_APK,IOS_ARM64_UNSIGNED_IPA_AND_APP_ZIP")
    print("PLATFORM_PACKAGING=LINUX_DIRECTORY,MACOS_DIRECTORY,IOS_DIRECTORY")
    print("OBSOLETE_PLATFORM_WRAPPERS=REMOVED")
    print("OBSOLETE_SOURCE_SYNC_WORKFLOW=REMOVED")
    print("ANDROID_APK_PUBLICATION=DEBUG_SIGNED_AND_RELEASE_UNSIGNED")
    print("ANDROID_PRODUCTION_SIGNING=EXTERNAL_IDENTITY_REQUIRED")
    print("IOS_IPA_PUBLICATION=UNSIGNED_ARM64_DEVICE_BUILD")
    print("IOS_PRODUCTION_SIGNING=EXTERNAL_APPLE_IDENTITY_REQUIRED")
    print("PUBLISHER=CENTRALIZED")
    print("RELEASE_GITHUB_API=WRAPPED_AND_AUDITED")
    return 0


if __name__ == "__main__":
    sys.exit(main())
