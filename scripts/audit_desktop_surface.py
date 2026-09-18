#!/usr/bin/env python3
"""Fail closed if retired application surfaces re-enter active source."""

from __future__ import annotations

import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RETIRED_ROOTS = (
    "ios/",
    "macos/",
    "GhostFTP WEB/",
    "pwa/",
    "ghostftp-web/",
    "web/",
    "web-ftp/",
    "webftp/",
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

    required_surfaces = {
        "Windows": {
            "cmd/ghostftp/main.go",
            "BUILD-WINDOWS.ps1",
            ".github/workflows/ci.yml",
        },
        "Linux": {
            "linux/BUILD-DISTROS.sh",
            "internal/desktop/gui_linux.go",
            ".github/workflows/ci.yml",
        },
        "Android": {
            "android/app/build.gradle",
            "android/app/src/main/AndroidManifest.xml",
            "android/app/src/main/java/app/ghostftp/client/MainActivity.java",
            ".github/workflows/android-apk.yml",
        },
    }
    for label, required in required_surfaces.items():
        missing = sorted(required - path_set)
        if missing:
            fail(f"active {label} source contract is incomplete: " + ", ".join(missing))

    retired_macos_paths = sorted(
        path for path in paths
        if path.replace("\\", "/").startswith("macos/")
        or path in {".github/workflows/macos-app.yml", ".github/workflows/macos-production.yml"}
        or path.replace("\\", "/").endswith("_darwin.go")
        or "/darwin_" in path.replace("\\", "/")
        or Path(path).name.startswith("test_macos_")
    )
    if retired_macos_paths:
        fail("retired macOS source/workflow/test surface is tracked: " + ", ".join(retired_macos_paths[:30]))

    print("DESKTOP_SURFACE_AUDIT=PASS")
    print("DESKTOP_SURFACE_AUDIT_SCOPE=WINDOWS,LINUX,ANDROID,RETIRED_SURFACES")
    print("WINDOWS_SOURCE_SURFACE=ACTIVE")
    print("LINUX_SOURCE_SURFACE=ACTIVE")
    print("ANDROID_SOURCE_SURFACE=ACTIVE")
    print("MACOS_SOURCE_SURFACE=RETIRED")
    print("WEB_SURFACE=RETIRED")
    print("WEB_FTP_SURFACE=RETIRED")
    print("RETIRED_APPLICATION_PLATFORMS=IOS,MACOS")
    print("RETIRED_APPLICATION_SURFACES=PWA,WEB,WEB_FTP,MACOS")
    return 0


if __name__ == "__main__":
    sys.exit(main())
