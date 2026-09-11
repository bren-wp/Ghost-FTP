#!/usr/bin/env python3
"""Fail closed when unsupported application platforms enter the active product tree."""

from __future__ import annotations

import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
VERSION_RE = re.compile(r"^\d+\.\d+\.\d+$")
RETIRED_ROOTS = ("ios/", "macos/", "GhostFTP WEB/")
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
        lowered = normalized.lower()
        if normalized.startswith(RETIRED_ROOTS):
            fail(f"retired application platform/surface is tracked: {path}")
        if normalized in RETIRED_SCRIPTS:
            fail(f"retired platform tooling is tracked: {path}")
        if lowered.endswith("_darwin.go"):
            fail(f"retired macOS/Darwin platform source is tracked: {path}")

    missing_android = sorted(ANDROID_REQUIRED - path_set)
    if missing_android:
        fail("active Android source contract is incomplete: " + ", ".join(missing_android))

    if "internal/platform/filemove_other.go" in path_set:
        fail("generic unsupported-OS filemove fallback must not be restored")

    for rel in sorted(LINUX_PLATFORM_STUBS):
        if rel not in path_set:
            fail(f"required Linux platform stub is not tracked: {rel}")
        lines = read(rel).splitlines()
        if not lines or lines[0] != "//go:build linux":
            fail(f"Linux platform stub has a broad/non-Linux build contract: {rel}")

    ci = read(".github/workflows/ci.yml")
    release = read(".github/workflows/release.yml")
    for rel, text in (("ci.yml", ci), ("release.yml", release)):
        lowered = text.lower()
        for marker in ("runs-on: macos", "ios/", "macos/", "ghostftp web/"):
            if marker in lowered:
                fail(f"{rel} still references retired application platform marker: {marker}")
        for marker in ("windows:", "linux:"):
            if marker not in text:
                fail(f"Windows/Linux public workflow contract is incomplete in {rel}: missing {marker}")

    android_workflow = read(".github/workflows/android-apk.yml")
    for marker in (
        "Ghost FTP Android APK",
        "android/dist/Ghost-FTP-Android.apk",
        "name: ghostftp-android-apk",
        "lintDebug",
        "packageGhostFtpApk",
    ):
        if marker not in android_workflow:
            fail(f"Android development workflow is missing contract marker: {marker}")

    version = read("VERSION").strip()
    if not VERSION_RE.fullmatch(version):
        fail(f"VERSION is not semantic: {version!r}")
    major, minor, patch = (int(part) for part in version.split("."))
    if (major, minor, patch) < (0, 0, 1):
        fail("active product baseline must not precede 0.0.1")

    print(f"PLATFORM_CONTRACT_AUDIT=PASS ({version})")
    print("PUBLIC_RELEASE_PLATFORMS=WINDOWS,LINUX")
    print("ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID")
    print("ANDROID_APK_DEVELOPMENT_SURFACE=ACTIVE")
    print("RETIRED_APPLICATION_PLATFORMS=IOS,MACOS")
    print("RETIRED_APPLICATION_SURFACES=WEB,PWA")
    print("LINUX_PLATFORM_STUBS=EXPLICIT")
    print("DARWIN_SOURCE=BLOCKED")
    print("MINIMUM_PUBLIC_VERSION=0.0.1")
    print("VERSIONING_PLATFORM_INDEPENDENT=YES")
    return 0


if __name__ == "__main__":
    sys.exit(main())
