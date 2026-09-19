#!/usr/bin/env python3
"""Fail-closed validation of the Windows/Linux/Android Ghost FTP release contract."""
from __future__ import annotations
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PUBLIC_PLATFORM_ARTIFACTS = 9
PUBLIC_RELEASE_FILES = 12
DISTROS = ("Debian", "Ubuntu", "Fedora")
PROTECTED_RELEASE_TAG = "ghostftp-v0.0.7"
PROTECTED_RELEASE_MARKER = "PROTECTED_RELEASE_TAG=ghostftp-v0.0.7"

def fail(message: str) -> None:
    raise SystemExit("RELEASE_AUDIT_FAILED: " + message)

def read(rel: str) -> str:
    path = ROOT / rel
    if not path.is_file():
        fail(f"missing required file: {rel}")
    return path.read_text(encoding="utf-8")

def main() -> int:
    version = read("VERSION").strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        fail(f"invalid VERSION: {version!r}")

    release = read(".github/workflows/release.yml")
    required = [
        "name: Publish Ghost FTP",
        "needs: [quality, windows, linux, android]",
        "WINDOWS_AUTHENTICODE=${WINDOWS_SIGNING_STATE}",
        "ANDROID_APK=production-signed",
        "ANDROID_SIGNER_SHA256=${ANDROID_SIGNER_SHA256}",
        "PUBLIC_RELEASE_PLATFORMS=WINDOWS,LINUX,ANDROID",
        "ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID",
        f"PUBLIC_PLATFORM_ARTIFACTS={PUBLIC_PLATFORM_ARTIFACTS}",
        f"PUBLIC_RELEASE_FILES={PUBLIC_RELEASE_FILES}",
        f"test \"$count\" = '{PUBLIC_RELEASE_FILES}'",
        "Ghost-FTP-${VERSION}-Setup.exe",
        "Ghost-FTP-${VERSION}-Portable.exe",
        "Ghost-FTP-${VERSION}-Android.apk",
        "RELEASE_ASSET_READBACK=PASS",
    ]
    for marker in required:
        if marker not in release:
            fail(f"release.yml missing {marker}")

    for distro in DISTROS:
        for suffix in ("Installer.run", "Portable.tar.gz"):
            marker = f"Ghost-FTP-${{VERSION}}-Linux-{distro}-{suffix}"
            if marker not in release:
                fail(f"release.yml missing {marker}")

    for retired in ("macos:", "MACOS_", "macOS", "macos/"):
        if retired in release:
            fail(f"release.yml still references retired macOS surface: {retired}")

    retention = read(".github/workflows/release-retention.yml")
    if f'protected_tag="{PROTECTED_RELEASE_TAG}"' not in retention:
        fail(f"release retention does not preserve {PROTECTED_RELEASE_TAG}")
    if "test \"$asset_count\" -eq 12" not in retention:
        fail("release retention still expects a stale public asset count")


    print(f"RELEASE_AUDIT=PASS ({version}; {PUBLIC_PLATFORM_ARTIFACTS} platform artifacts; {PUBLIC_RELEASE_FILES} public files)")
    print(PROTECTED_RELEASE_MARKER)
    return 0

if __name__ == "__main__":
    sys.exit(main())
