#!/usr/bin/env python3
"""Fail closed when retired application platforms re-enter the active Ghost FTP tree."""
from __future__ import annotations
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
VERSION_RE = re.compile(r"^\d+\.\d+\.\d+$")
RETIRED_ROOTS = ("ios/", "macos/", "GhostFTP WEB/", "web/", "web-ftp/", "webftp/", "pwa/", "ghostftp-web/")
RETIRED_EXACT = {
    ".github/workflows/macos-app.yml",
    ".github/workflows/macos-production.yml",
}
ANDROID_REQUIRED = {
    "android/app/build.gradle",
    "android/app/src/main/AndroidManifest.xml",
    "android/app/src/main/java/app/ghostftp/client/MainActivity.java",
    ".github/workflows/android-apk.yml",
}
LINUX_REQUIRED = {
    "linux/BUILD.sh",
    "linux/BUILD-DISTROS.sh",
    ".github/workflows/linux-distro-packages.yml",
    ".github/workflows/linux-distro-install.yml",
}
BROWSER_REQUIRED = {
    "extensions/chrome/manifest.json",
    "extensions/edge/manifest.json",
    "extensions/firefox/manifest.json",
    "extensions/opera/manifest.json",
    ".github/workflows/browser-extensions.yml",
}

def fail(message: str) -> None:
    raise SystemExit("PLATFORM_CONTRACT_AUDIT_FAILED: " + message)

def tracked_paths() -> list[str]:
    result = subprocess.run(["git", "ls-files", "-z"], cwd=ROOT, check=True, stdout=subprocess.PIPE)
    return [p.decode("utf-8") for p in result.stdout.split(b"\0") if p]

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
            fail(f"retired application surface is tracked: {path}")
        if normalized in RETIRED_EXACT:
            fail(f"retired macOS workflow is tracked: {path}")
        if normalized.startswith("scripts/test_macos_"):
            fail(f"retired macOS regression test is tracked: {path}")

    for label, required in (("Android", ANDROID_REQUIRED), ("Linux", LINUX_REQUIRED), ("browser helper", BROWSER_REQUIRED)):
        missing = sorted(required - path_set)
        if missing:
            fail(f"active {label} contract is incomplete: " + ", ".join(missing))

    release = read(".github/workflows/release.yml")
    for marker in (
        "windows:", "linux:", "android:", "browser:",
        "Ghost-FTP-\${VERSION}-Setup.exe",
        "Ghost-FTP-\${VERSION}-Portable.exe",
        "Ghost-FTP-\${VERSION}-Android.apk",
        "PUBLIC_PLATFORM_ARTIFACTS=13",
        "PUBLIC_RELEASE_FILES=16",
    ):
        if marker not in release:
            fail(f"release contract missing {marker}")
    for retired in ("macos:", "MACOS_", "macOS", "macos/"):
        if retired in release:
            fail(f"release workflow still references retired macOS surface: {retired}")

    version = read("VERSION").strip()
    if not VERSION_RE.fullmatch(version):
        fail(f"VERSION is not semantic: {version!r}")

    print(f"PLATFORM_CONTRACT_AUDIT=PASS ({version})")
    print("PUBLIC_RELEASE_PLATFORMS=WINDOWS,LINUX,ANDROID,BROWSER_HELPER")
    print("ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID")
    print("RETIRED_APPLICATION_PLATFORMS=MACOS,IOS")
    return 0

if __name__ == "__main__":
    sys.exit(main())
