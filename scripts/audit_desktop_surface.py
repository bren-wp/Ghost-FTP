#!/usr/bin/env python3
"""Fail closed if retired application surfaces re-enter active source."""

from __future__ import annotations

import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RETIRED_ROOTS = (
    "ios/",
    "GhostFTP WEB/",
    "web/",
    "pwa/",
    "ghostftp-web/",
)
RETIRED_APP_MARKERS = (
    "manifest.webmanifest",
    "service-worker.js",
    "phpunit.xml",
)


def fail(message: str) -> None:
    raise SystemExit("DESKTOP_SURFACE_AUDIT_FAILED: " + message)


def tracked_paths() -> list[str]:
    try:
        raw = subprocess.check_output(
            ["git", "ls-files", "-z"], cwd=ROOT, stderr=subprocess.STDOUT
        )
    except (OSError, subprocess.CalledProcessError) as exc:
        fail(f"git ls-files failed: {exc}")
    return [item.decode("utf-8", "strict") for item in raw.split(b"\0") if item]


def main() -> int:
    paths = tracked_paths()
    path_set = set(paths)
    retired: list[str] = []
    suspicious: list[str] = []
    for path in paths:
        normalized = path.replace("\\", "/")
        lowered = normalized.lower()
        if normalized.startswith(RETIRED_ROOTS):
            retired.append(path)
            continue
        if lowered.startswith(("client-web/", "app-web/")) and any(
            lowered.endswith(marker) for marker in RETIRED_APP_MARKERS
        ):
            suspicious.append(path)

    if retired:
        fail("retired application source is tracked: " + ", ".join(sorted(retired)[:20]))
    if suspicious:
        fail("retired web application surface is tracked: " + ", ".join(sorted(suspicious)))

    android_required = {
        "android/app/build.gradle",
        "android/app/src/main/AndroidManifest.xml",
        "android/app/src/main/java/app/ghostftp/client/MainActivity.java",
        ".github/workflows/android-apk.yml",
    }
    missing_android = sorted(android_required - path_set)
    if missing_android:
        fail("active Android source contract is incomplete: " + ", ".join(missing_android))

    macos_required = {
        "macos/README.md",
        "macos/PARITY.md",
        "macos/BUILD.sh",
        "macos/Sources/GhostFTPApp/main.swift",
        ".github/workflows/macos-app.yml",
    }
    missing_macos = sorted(macos_required - path_set)
    if missing_macos:
        fail("active macOS source contract is incomplete: " + ", ".join(missing_macos))

    print("DESKTOP_SURFACE_AUDIT=PASS")
    print("PUBLIC_RELEASE_PLATFORMS=WINDOWS,LINUX")
    print("ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID,MACOS")
    print("ANDROID_APK_DEVELOPMENT_SURFACE=ACTIVE")
    print("MACOS_APP_DEVELOPMENT_SURFACE=ACTIVE")
    print("RETIRED_APPLICATION_PLATFORMS=IOS")
    print("RETIRED_APPLICATION_SURFACES=WEB,PWA")
    return 0


if __name__ == "__main__":
    sys.exit(main())
