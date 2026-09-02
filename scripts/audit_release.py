#!/usr/bin/env python3
"""Validate the ByFTP production release workflow and publication contract."""

from __future__ import annotations

import re
import subprocess
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


def run_python_audit(rel: str, label: str) -> None:
    audit = ROOT / rel
    if not audit.is_file():
        fail(f"missing required file: {rel}")
    try:
        subprocess.run([sys.executable, str(audit)], cwd=ROOT, check=True)
    except subprocess.CalledProcessError as exc:
        fail(f"{label} failed with exit code {exc.returncode}")


def main() -> int:
    version = read("VERSION").strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        fail(f"invalid VERSION: {version!r}")

    run_python_audit("scripts/audit_repository.py", "repository-wide tracked-file audit")
    run_python_audit("scripts/audit_web.py", "ByFTP WEB audit/runtime gate")

    workflow = read(".github/workflows/release.yml")
    for marker in (
        "group: byftp-release",
        "quality:", "windows:", "linux:", "macos:", "android:", "ios:", "publish:",
        "needs: [quality, windows, linux, macos, android, ios]",
        "go-version: '1.27.1'", "gradle-version: '9.7.1'",
        "go telemetry off", "go test ./...", "go test -race ./...", "go vet ./...",
        "python scripts/audit_version.py", "python scripts/audit_repository.py", "python scripts/audit_web.py",
        "python scripts/audit_android.py", "python scripts/audit_ios.py", "python scripts/audit_docs.py",
        "python scripts/audit_security.py", "python scripts/audit_privacy.py", "python scripts/audit_release.py",
        "python scripts/package_web.py --output-dir dist", "byftp-web-release",
        ".\\BUILD-WINDOWS.ps1", ".\\scripts\\package_windows_bundles.ps1",
        "bash linux/BUILD.sh", "bash macos/BUILD.sh", "bash ios/BUILD.sh",
        "android-37*", "build-tools/36.0.0",
        ":app:testDebugUnitTest", ":app:lintDebug", ":app:lintRelease", ":app:assembleDebug", ":app:assembleRelease",
        "python scripts/package_android.py", "byftp-android-release", "byftp-ios-release",
        "Download WEB package", ".\\scripts\\prepare_release.ps1",
        "scripts\\publish_release.ps1", "Expected 18 public release files",
        "<PackageId>ByFTP.Windows</PackageId>", "dotnet nuget push", "--skip-duplicate",
    ):
        require(workflow, marker, ".github/workflows/release.yml")

    ci = read(".github/workflows/ci.yml")
    for marker in ("python scripts/audit_release.py", "go-version: '1.27.1'", "gradle-version: '9.7.1'"):
        require(ci, marker, ".github/workflows/ci.yml")

    android_root = read("android/build.gradle.kts")
    require(android_root, 'id("com.android.application") version "9.4.0" apply false', "android/build.gradle.kts")

    web_packager = read("scripts/package_web.py")
    for marker in (
        'git", "ls-files", "-z", "--", "ByFTP WEB"',
        "ByFTP-{version}-WEB-shared-hosting.zip",
        "tracked WEB symlink is not allowed",
        "archived composer.json version does not match canonical VERSION",
        "byftp-static-v{version}",
    ):
        require(web_packager, marker, "scripts/package_web.py")
    read("scripts/test_package_web.py")

    windows_bundler = read("scripts/package_windows_bundles.ps1")
    for marker in (
        "ByFTP-$version-Portable-$arch.exe", "ByFTP-$version-Setup-$arch.exe",
        "ByFTP-$version-Windows-$arch.zip", "scripts/verify_bundle.py",
        "WINDOWS_UNINSTALLER=none", "unexpectedly produced an uninstaller-named file",
    ):
        require(windows_bundler, marker, "scripts/package_windows_bundles.ps1")

    release_staging = read("scripts/prepare_release.ps1")
    for marker in (
        "ByFTP-$version-Portable-x64.exe", "ByFTP-$version-Setup-x64.exe",
        "ByFTP-$version-Portable-x86.exe", "ByFTP-$version-Setup-x86.exe",
        "ByFTP-$version-Linux-amd64.deb", "ByFTP-$version-Linux-arm64.deb", "ByFTP-$version-Linux-i386.deb",
        "ByFTP-$version-macOS-Universal.pkg",
        "ByFTP-$version-Android-debug.apk", "ByFTP-$version-Android-release-unsigned.apk",
        "ByFTP-$version-iOS-arm64-unsigned.ipa", "ByFTP-$version-iOS-arm64-unsigned-app.zip",
        "ByFTP-$version-WEB-shared-hosting.zip",
        "PUBLIC_PLATFORM_ARTIFACTS=15", "SHARED_METADATA_ARTIFACTS=3", "PUBLIC_RELEASE_FILES=18",
        "WINDOWS_UNINSTALLER=none", "Release staging contains an uninstaller-named public asset",
        "SHA256.txt", "BUILD-METADATA.txt", "RELEASE-NOTES.txt",
    ):
        require(release_staging, marker, "scripts/prepare_release.ps1")

    for legacy in (
        "scripts/BUILD-LINUX.sh", "scripts/BUILD-MACOS.sh", "scripts/BUILD-IOS.sh",
        ".github/workflows/__byftp_sync.yml", "cmd/uninstaller",
    ):
        if (ROOT / legacy).exists():
            fail(f"obsolete release/build surface still exists: {legacy}")

    windows_build = read("BUILD-WINDOWS.ps1")
    for marker in (
        "scripts/make_payload.py", "--app", "scripts/verify_release.py",
        "UNINSTALLER_BINARY", "unexpectedly produced an uninstaller binary",
    ):
        require(windows_build, marker, "BUILD-WINDOWS.ps1")
    for forbidden in ("./cmd/uninstaller", "--uninstaller", "-Uninstall-", "'uninstaller'"):
        if forbidden in windows_build:
            fail(f"Windows build still contains obsolete uninstaller path: {forbidden}")

    payload = read("scripts/make_payload.py")
    for marker in ("PAYLOAD_SCHEMA = 2", 'add(zf, args.app, "ByFTP.exe")'):
        require(payload, marker, "scripts/make_payload.py")
    for forbidden in ("--uninstaller", '"Uninstall.exe"'):
        if forbidden in payload:
            fail(f"installer payload generator still contains obsolete uninstaller marker: {forbidden}")

    installer = read("cmd/installer/main.go")
    for marker in (
        "payloadSchema          = 2", "func cleanupLegacyUninstaller(dir string) string",
        'legacyPath := filepath.Join(dir, "Uninstall.exe")', "platform.DeleteRegistryKey(legacyUninstallKey)",
        "transactionCommitted = true", "legacyCleanupWarning := cleanupLegacyUninstaller(dir)",
    ):
        require(installer, marker, "cmd/installer/main.go")

    windows_verifier = read("scripts/verify_release.py")
    for marker in ("UNINSTALLER_BINARY=ABSENT", "SETUP_PE_OK=YES", "PORTABLE_PE_OK=YES"):
        require(windows_verifier, marker, "scripts/verify_release.py")
    if 'add_argument("uninstaller"' in windows_verifier:
        fail("Windows release verifier still accepts an uninstaller binary")

    publisher = read("scripts/publish_release.ps1")
    for marker in (
        "function Invoke-GhJson", "function Try-GhJson", "gh release create", "gh release edit",
        "gh release upload", "Get-FileHash", "SHA256", "Assert-TagCommit", "Assert-RemoteAsset",
        "function Assert-CurrentMainCommit", "repos/$Repository/branches/main",
        "RELEASE_MAIN_HEAD_VERIFICATION=PASS", "RELEASE_PUBLISH_VERIFICATION=PASS",
    ):
        require(publisher, marker, "scripts/publish_release.ps1")

    guard_call = publisher.find("Assert-CurrentMainCommit", publisher.find('$tag = "v$Version"'))
    release_lookup = publisher.find("$release = Get-Release -Tag $tag")
    if guard_call < 0 or release_lookup < 0 or guard_call > release_lookup:
        fail("publisher must verify current main before release lookup/mutation")

    android_packager = read("scripts/package_android.py")
    for marker in ("ANDROID_PACKAGE_FAILED", "AndroidManifest.xml", "classes.dex", "resources.arsc", "validate_apk"):
        require(android_packager, marker, "scripts/package_android.py")

    ios_packager = read("scripts/package_ios.py")
    for marker in ("IOS_PACKAGE_FAILED", "Payload/ByFTP.app", "CFBundleShortVersionString", "Mach-O", "symlink"):
        require(ios_packager, marker, "scripts/package_ios.py")

    linux_build = read("linux/BUILD.sh")
    for marker in ("VERSION", "dpkg-deb", "build_arch amd64 amd64", "build_arch arm64 arm64", "build_arch 386 i386"):
        require(linux_build, marker, "linux/BUILD.sh")

    macos_build = read("macos/BUILD.sh")
    for marker in ("VERSION", "lipo -create", "pkgbuild", "ByFTP-${VERSION}-macOS-Universal.pkg"):
        require(macos_build, marker, "macos/BUILD.sh")

    ios_build = read("ios/BUILD.sh")
    for marker in ("VERSION", "xcodebuild", "generic/platform=iOS", "ARCHS=arm64", "scripts/package_ios.py"):
        require(ios_build, marker, "ios/BUILD.sh")

    verifier = read("scripts/verify_bundle.py")
    for marker in ("BUNDLE_VERIFICATION_FAILED", "BUNDLE-SHA256.txt", "Documentation/SECURITY.md"):
        require(verifier, marker, "scripts/verify_bundle.py")

    for rel in (
        "README.md", "CHANGELOG.md", "linux/README.md", "macos/README.md", "android/README.md", "ios/README.md",
        "ByFTP WEB/README.md", "docs/INSTALLATION.md", "docs/RELEASE-VERIFICATION.md", "docs/SECURITY.md", "docs/PRIVACY.md",
    ):
        read(rel)

    for rel in (
        ".github/workflows/release.yml", "scripts/publish_release.ps1", "scripts/verify_bundle.py",
        "scripts/package_android.py", "scripts/package_ios.py", "scripts/package_web.py",
        "scripts/package_windows_bundles.ps1", "scripts/prepare_release.ps1",
        "linux/BUILD.sh", "macos/BUILD.sh", "ios/BUILD.sh",
    ):
        if "brendigo" in read(rel).lower():
            fail(f"legacy branding remains in release surface: {rel}")

    print(f"RELEASE_AUDIT=PASS ({version})")
    print("REPOSITORY_WIDE_TRACKED_FILE_AUDIT=REQUIRED")
    print("WEB_RUNTIME_AND_SECURITY_GATE=REQUIRED")
    print("WEB_DEPLOYABLE_PACKAGE=REQUIRED")
    print("RELEASE_MAIN_HEAD_GUARD=REQUIRED")
    print("RELEASE_MATRIX=WINDOWS_X64_X86,LINUX_AMD64_ARM64_I386,MACOS_UNIVERSAL,ANDROID_DEBUG_AND_UNSIGNED_RELEASE_APK,IOS_ARM64_UNSIGNED_IPA_AND_APP_ZIP,WEB_SHARED_HOSTING_ZIP")
    print("PUBLIC_PLATFORM_ARTIFACTS=15")
    print("PUBLIC_RELEASE_FILES=18")
    print("WINDOWS_STANDALONE_UNINSTALLER=REMOVED")
    print("WINDOWS_INSTALLER_PAYLOAD=APP_ONLY_SCHEMA_2")
    print("ANDROID_APK_PUBLICATION=DEBUG_SIGNED_AND_RELEASE_UNSIGNED")
    print("IOS_IPA_PUBLICATION=UNSIGNED_ARM64_DEVICE_BUILD")
    print("PUBLISHER=CENTRALIZED_AND_CURRENT_MAIN_BOUND")
    return 0


if __name__ == "__main__":
    sys.exit(main())
