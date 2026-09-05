#!/usr/bin/env python3
"""Fail closed when retired application platforms re-enter the active product tree."""

from __future__ import annotations

import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RETIRED_ROOTS = ("android/", "ios/", "macos/")
RETIRED_SCRIPTS = {
    "scripts/audit_android.py",
    "scripts/audit_android_localization.py",
    "scripts/audit_ios.py",
    "scripts/package_android.py",
    "scripts/package_ios.py",
}


def fail(message: str) -> None:
    raise SystemExit("PLATFORM_CONTRACT_AUDIT_FAILED: " + message)


def tracked_paths() -> list[str]:
    result = subprocess.run(
        ["git", "ls-files", "-z"],
        cwd=ROOT,
        check=True,
        stdout=subprocess.PIPE,
    )
    return [item.decode("utf-8", errors="strict") for item in result.stdout.split(b"\0") if item]


def read(rel: str) -> str:
    path = ROOT / rel
    if not path.is_file():
        fail(f"missing required file: {rel}")
    return path.read_text(encoding="utf-8")


def main() -> int:
    paths = tracked_paths()
    for path in paths:
        normalized = path.replace("\\", "/")
        if normalized.startswith(RETIRED_ROOTS):
            fail(f"retired application platform is tracked: {path}")
        if normalized in RETIRED_SCRIPTS:
            fail(f"retired platform tooling is tracked: {path}")

    ci = read(".github/workflows/ci.yml")
    release = read(".github/workflows/release.yml")
    for rel, text in (("ci.yml", ci), ("release.yml", release)):
        lowered = text.lower()
        for marker in ("runs-on: macos", "android/", "ios/", "macos/"):
            if marker in lowered:
                fail(f"{rel} still references retired application platform marker: {marker}")

    for marker in ("windows:", "linux:"):
        if marker not in ci or marker not in release:
            fail(f"Windows/Linux workflow contract is incomplete: missing {marker}")

    version = read("VERSION").strip()
    if not version.startswith("2."):
        fail("Windows/Linux-only product contract requires the 2.x major line")

    print("PLATFORM_CONTRACT_AUDIT=PASS")
    print("ACTIVE_APPLICATION_PLATFORMS=WINDOWS,LINUX")
    print("RETIRED_APPLICATION_PLATFORMS=ANDROID,IOS,MACOS")
    return 0


if __name__ == "__main__":
    sys.exit(main())
