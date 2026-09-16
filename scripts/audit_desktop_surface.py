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
    "pwa/",
    "ghostftp-web/",
    "web/",
)
RETIRED_APP_MARKERS = (
    "manifest.webmanifest",
    "service-worker.js",
    "phpunit.xml",
)
RETIRED_EXACT_PATHS = {
    ".github/workflows/web.yml",
    "docs/WEB.md",
    "scripts/check_web_contract.py",
    "scripts/test_web_contract.py",
    "docs/prompts/GHOST-FTP-WEB-APP-PROMPT.md",
    "docs/prompts/GHOSTFTP-COM-DARK-THEME-REDESIGN-PROMPT.md",
}


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
        if normalized.startswith(RETIRED_ROOTS) or normalized in RETIRED_EXACT_PATHS:
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
    macos_required = {
        "macos/README.md",
        "macos/PARITY.md",
        "macos/BUILD.sh",
        "macos/Sources/GhostFTPApp/main.swift",
        ".github/workflows/macos-app.yml",
    }

    for label, required in (
        ("Android", android_required),
        ("macOS", macos_required),
    ):
        missing = sorted(required - path_set)
        if missing:
            fail(f"active {label} source contract is incomplete: " + ", ".join(missing))

    print("DESKTOP_SURFACE_AUDIT=PASS")
    print("DESKTOP_SURFACE_AUDIT_SCOPE=ANDROID,MACOS,RETIRED_SURFACES")
    print("ANDROID_SOURCE_SURFACE=ACTIVE")
    print("MACOS_SOURCE_SURFACE=ACTIVE")
    print("WEB_SURFACE=RETIRED")
    print("WEB_FTP_SURFACE=RETIRED")
    print("RETIRED_APPLICATION_PLATFORMS=IOS")
    print("RETIRED_APPLICATION_SURFACES=PWA,WEB,WEB_FTP")
    return 0


if __name__ == "__main__":
    sys.exit(main())
