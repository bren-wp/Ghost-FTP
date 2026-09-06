#!/usr/bin/env python3
"""Fail closed if a retired non-desktop application surface re-enters source."""

from __future__ import annotations

import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RETIRED_ROOTS = (
    "android/",
    "ios/",
    "macos/",
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
        fail("non-desktop application surface is tracked: " + ", ".join(sorted(suspicious)))

    print("DESKTOP_SURFACE_AUDIT=PASS")
    print("ACTIVE_APPLICATION_PLATFORMS=WINDOWS,LINUX")
    print("NON_DESKTOP_APPLICATION_SOURCE=BLOCKED")
    return 0


if __name__ == "__main__":
    sys.exit(main())
