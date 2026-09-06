#!/usr/bin/env python3
"""Fail closed if the retired Ghost FTP Web/PWA surface re-enters desktop source."""

from __future__ import annotations

import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RETIRED_ROOT = "GhostFTP WEB/"
RETIRED_MARKERS = (
    "manifest.webmanifest",
    "service-worker.js",
    "composer.json",
    "phpunit.xml",
)


def fail(message: str) -> None:
    raise SystemExit("DESKTOP_ONLY_AUDIT_FAILED: " + message)


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
    retired = [path for path in paths if path.replace("\\", "/").startswith(RETIRED_ROOT)]
    if retired:
        fail("retired Web/PWA source is tracked: " + ", ".join(sorted(retired)[:20]))

    if (ROOT / "GhostFTP WEB").exists():
        fail("retired GhostFTP WEB directory exists in the checkout")

    # Guard against a future reintroduction under another obvious root name.
    suspicious: list[str] = []
    for path in paths:
        normalized = path.replace("\\", "/").lower()
        if normalized.startswith(("web/", "pwa/", "ghostftp-web/")):
            if any(normalized.endswith(marker) for marker in RETIRED_MARKERS):
                suspicious.append(path)
    if suspicious:
        fail("desktop repository contains a Web/PWA application surface: " + ", ".join(sorted(suspicious)))

    print("DESKTOP_ONLY_AUDIT=PASS")
    print("ACTIVE_APPLICATION_PLATFORMS=WINDOWS,LINUX")
    print("RETIRED_WEB_PWA_SOURCE=BLOCKED")
    print("RETIRED_MOBILE_SOURCE=BLOCKED_BY_PLATFORM_CONTRACT")
    return 0


if __name__ == "__main__":
    sys.exit(main())
