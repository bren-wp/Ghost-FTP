#!/usr/bin/env python3
"""Validate Ghost FTP version identity across maintained Windows/Linux/Android surfaces."""
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]

def fail(message: str) -> None:
    raise SystemExit("VERSION_AUDIT_FAILED: " + message)

def read(rel: str) -> str:
    path = ROOT / rel
    if not path.is_file():
        fail(f"missing required file: {rel}")
    return path.read_text(encoding="utf-8")

def main() -> int:
    version = read("VERSION").strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        fail(f"invalid VERSION: {version!r}")
    if version == "0.0.0":
        fail("0.0.0 is reserved")

    release = read(".github/workflows/release.yml")
    for marker in (
        "Ghost-FTP-${VERSION}-Setup.exe",
        "Ghost-FTP-${VERSION}-Portable.exe",
        "Ghost-FTP-${VERSION}-Android.apk",
        "PUBLIC_PLATFORM_ARTIFACTS=13",
        "PUBLIC_RELEASE_FILES=16",
    ):
        if marker not in release:
            fail(f"release version contract missing {marker}")
    for retired in ("macOS", "MACOS_", "macos/"):
        if retired in release:
            fail(f"retired macOS marker remains in release workflow: {retired}")

    android = read("android/app/build.gradle")
    if "versionName" not in android:
        fail("Android build does not declare versionName")

    print(f"VERSION_AUDIT=PASS ({version})")
    print("ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID")
    return 0

if __name__ == "__main__":
    sys.exit(main())
