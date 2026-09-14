#!/usr/bin/env python3
"""Fail closed when unsupported application platforms enter the active product tree."""

from __future__ import annotations

import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
VERSION_RE = re.compile(r"^\d+\.\d+\.\d+$")
RETIRED_ROOTS = ("ios/", "GhostFTP WEB/")
RETIRED_SCRIPTS = {
    "scripts/audit_android.py",
    "scripts/audit_android_localization.py",
    "scripts/audit_ios.py",
    "scripts/package_android.py",
    "scripts/package_ios.py",
}
LINUX_PLATFORM_STUBS = {
    "internal/platform/delete_other.go",
    "internal/platform/language_other.go",
    "internal/platform/other.go",
    "internal/platform/prompt_other.go",
    "internal/platform/registry_other.go",
    "internal/platform/shortcut_other.go",
}
ANDROID_REQUIRED = {
    "android/app/build.gradle",
    "android/app/src/main/AndroidManifest.xml",
    "android/app/src/main/java/app/ghostftp/client/MainActivity.java",
    "android/app/src/main/java/app/ghostftp/client/FtpSession.java",
    ".github/workflows/android-apk.yml",
    "scripts/test_android_contract.py",
    "scripts/test_android_release_signing_contract.py",
}
MACOS_REQUIRED = {
    "macos/README.md",
    "macos/PARITY.md",
    "macos/BUILD.sh",
    "macos/Sources/GhostFTPApp/main.swift",
    ".github/workflows/macos-app.yml",
    "scripts/test_macos_windows_parity_contract.py",
}
BROWSER_REQUIRED = {
    "ekstenzije/manifests/chrome.json",
    "ekstenzije/manifests/edge.json",
    "ekstenzije/manifests/firefox.json",
    "scripts/build_browser_extensions.py",
    "scripts/test_browser_extensions_contract.py",
    ".github/workflows/browser-extensions.yml",
}


def fail(message: str) -> None:
    raise SystemExit("PLATFORM_CONTRACT_AUDIT_FAILED: " + message)


def tracked_paths() -> list[str]:
    result = subprocess.run(["git", "ls-files", "-z"], cwd=ROOT, check=True, stdout=subprocess.PIPE)
    return [item.decode("utf-8", errors="strict") for item in result.stdout.split(b"\0") if item]


def read(rel: str) -> str:
    path = ROOT / rel
    if not path.is_file():
        fail(f"missing required file: {rel}")
    return path.read_text(encoding="utf-8")


def main() -> int:
    paths = tracked_paths()
    path_set = set(paths)
    for path in paths:
        normalized = path.replace("\\", "/")
        if normalized.startswith(RETIRED_ROOTS):
            fail(f"retired application platform/surface is tracked: {path}")
        if normalized in RETIRED_SCRIPTS:
            fail(f"retired platform tooling is tracked: {path}")

    for label, required in (
        ("Android", ANDROID_REQUIRED),
        ("macOS", MACOS_REQUIRED),
        ("browser helper", BROWSER_REQUIRED),
    ):
        missing = sorted(required - path_set)
        if missing:
            fail(f"active {label} source contract is incomplete: " + ", ".join(missing))

    if "internal/platform/filemove_other.go" in path_set:
        fail("generic unsupported-OS filemove fallback must not be restored")

    for rel in sorted(LINUX_PLATFORM_STUBS):
        if rel not in path_set:
            fail(f"required Linux platform stub is not tracked: {rel}")
        lines = read(rel).splitlines()
        if not lines or lines[0] != "//go:build linux":
            fail(f"Linux platform stub has a broad/non-Linux build contract: {rel}")

    ci = read(".github/workflows/ci.yml")
    ci_lower = ci.lower()
    for marker in ("runs-on: macos", "ios/", "macos/", "ghostftp web/"):
        if marker in ci_lower:
            fail(f"ci.yml desktop production contract unexpectedly references: {marker}")
    for marker in ("windows:", "linux:"):
        if marker not in ci:
            fail(f"Windows/Linux desktop CI contract is incomplete: missing {marker}")

    release = read(".github/workflows/release.yml")
    release_lower = release.lower()
    for marker in ("runs-on: macos", "ios/", "macos/", "ghostftp web/"):
        if marker in release_lower:
            fail(f"release.yml public contract unexpectedly references unsupported public surface: {marker}")
    for marker in (
        "windows:",
        "linux:",
        "android:",
        "browser:",
        "Production signed Android APK",
        "Chrome Edge Firefox release packages",
        "Ghost-FTP-${VERSION}-Android.apk",
        "Ghost-FTP-${VERSION}-Chrome-Extension.zip",
        "PUBLIC_PLATFORM_ARTIFACTS=18",
        "PUBLIC_RELEASE_FILES=21",
    ):
        if marker not in release:
            fail(f"cross-platform public release contract is incomplete: missing {marker}")

    android_workflow = read(".github/workflows/android-apk.yml")
    for marker in (
        "Ghost FTP Android APK",
        "android/dist/Ghost-FTP-Android-dev.apk",
        "name: ghostftp-android-dev-apk",
        "lintDebug",
        "lintRelease",
        "packageGhostFtpApk",
        "assembleRelease",
        "apksigner",
        "EPHEMERAL_CI_ONLY",
    ):
        if marker not in android_workflow:
            fail(f"Android development workflow is missing contract marker: {marker}")
    for forbidden in ("android/dist/Ghost-FTP-Android.apk", "name: ghostftp-android-apk"):
        if forbidden in android_workflow:
            fail(f"Android development workflow contains retired ambiguous artifact marker: {forbidden}")

    macos_workflow = read(".github/workflows/macos-app.yml")
    for marker in (
        "Ghost FTP macOS Development App",
        "runs-on: macos-",
        "bash macos/BUILD.sh",
        "ghostftp-macos-development",
    ):
        if marker not in macos_workflow:
            fail(f"macOS development workflow is missing contract marker: {marker}")

    version = read("VERSION").strip()
    if not VERSION_RE.fullmatch(version):
        fail(f"VERSION is not semantic: {version!r}")
    if tuple(int(part) for part in version.split(".")) < (0, 0, 1):
        fail("active product baseline must not precede 0.0.1")

    print(f"PLATFORM_CONTRACT_AUDIT=PASS ({version})")
    print("PUBLIC_RELEASE_PLATFORMS=WINDOWS,LINUX,ANDROID,BROWSER_HELPER")
    print("ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID,MACOS")
    print("ANDROID_APK_DEVELOPMENT_SURFACE=ACTIVE")
    print("ANDROID_RELEASE_BUILD_AND_SIGNING_SMOKE=ACTIVE")
    print("ANDROID_PUBLIC_RELEASE_ARTIFACT=YES_PRODUCTION_SIGNED")
    print("ANDROID_SFTP_PUBLIC_SUPPORT=NO_STRICT_HOST_KEY_BOUNDARY")
    print("BROWSER_PUBLIC_RELEASE_PACKAGES=CHROME,EDGE,FIREFOX")
    print("BROWSER_DESKTOP_HANDOFF=UNSUPPORTED")
    print("MACOS_APP_DEVELOPMENT_SURFACE=ACTIVE")
    print("MACOS_PUBLIC_RELEASE_ARTIFACT=NO")
    print("RETIRED_APPLICATION_PLATFORMS=IOS")
    print("RETIRED_APPLICATION_SURFACES=WEB,PWA")
    print("LINUX_PLATFORM_STUBS=EXPLICIT")
    print("MINIMUM_PUBLIC_VERSION=0.0.1")
    print("VERSIONING_PLATFORM_INDEPENDENT=YES")
    return 0


if __name__ == "__main__":
    sys.exit(main())
