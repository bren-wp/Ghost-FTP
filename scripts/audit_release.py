#!/usr/bin/env python3
"""Validate the Ghost FTP production release contract."""

from __future__ import annotations

import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def fail(message: str) -> None:
    raise SystemExit("RELEASE_AUDIT_FAILED: " + message)


def read(rel: str) -> str:
    path = ROOT / rel
    if not path.is_file():
        fail(f"missing required file: {rel}")
    try:
        return path.read_text(encoding="utf-8")
    except UnicodeError as exc:
        fail(f"{rel} is not valid UTF-8: {exc}")


def require(text: str, markers: tuple[str, ...], where: str) -> None:
    for marker in markers:
        if marker not in text:
            fail(f"{where} is missing required marker: {marker}")


def run_python_audit(rel: str, label: str) -> None:
    audit = ROOT / rel
    if not audit.is_file():
        fail(f"missing required file: {rel}")
    try:
        subprocess.run([sys.executable, str(audit)], cwd=ROOT, check=True)
    except subprocess.CalledProcessError as exc:
        fail(f"{label} failed with exit code {exc.returncode}")


def main() -> int:
    version = read("VERSION").strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        fail(f"invalid VERSION: {version!r}")
    if read("ByFTP WEB/VERSION").strip() != version:
        fail("web VERSION does not match the repository VERSION")

    # Keep repository/web packaging gates active, but release publication itself
    # has exactly one implementation: .github/workflows/release.yml.
    run_python_audit("scripts/audit_repository.py", "repository-wide tracked-file audit")
    run_python_audit("scripts/audit_web.py", "Ghost FTP web audit/runtime gate")

    workflow = read(".github/workflows/release.yml")
    require(
        workflow,
        (
            "name: Publish Ghost FTP",
            "group: ghostftp-release",
            "permissions:\n  contents: write",
            "quality:",
            "windows:",
            "linux:",
            "macos:",
            "android:",
            "ios:",
            "publish:",
            "needs: [quality, windows, linux, macos, android, ios]",
            "go telemetry off",
            "go test ./...",
            "go test -race ./...",
            "go vet ./...",
            "python scripts/audit_security.py",
            "python scripts/audit_privacy.py",
            "python scripts/package_web.py --output-dir dist",
            ".\\BUILD-WINDOWS.ps1",
            "bash linux/BUILD.sh",
            "bash macos/BUILD.sh",
            "bash ios/BUILD.sh",
            ":app:testDebugUnitTest",
            ":app:lintDebug",
            ":app:assembleDebug",
            "RELEASE_TAG=ghostftp-v$version",
            "Ghost-FTP-${VERSION}-Setup-x64.exe",
            "Ghost-FTP-${VERSION}-Setup-x86.exe",
            "Ghost-FTP-${VERSION}-Setup-x32.exe",
            "Ghost-FTP-${VERSION}-Linux-multiarch.zip",
            "Ghost-FTP-${VERSION}-macOS-Universal.pkg",
            "Ghost-FTP-${VERSION}-Android.apk",
            "Ghost-FTP-${VERSION}-iOS-arm64-unsigned.ipa",
            "Ghost-FTP-${VERSION}-Web.zip",
            "PUBLIC_PLATFORM_ARTIFACTS=8",
            "PUBLIC_RELEASE_FILES=11",
            "main moved from release commit",
            "refusing to rewrite it",
            "gh release create",
            "SHA256.txt",
            "BUILD-METADATA.txt",
            "RELEASE-NOTES.txt",
        ),
        ".github/workflows/release.yml",
    )

    if "packages: write" in workflow or "dotnet nuget push" in workflow:
        fail("release workflow contains obsolete package-registry publication")
    for obsolete in (
        "scripts/package_windows_bundles.ps1",
        "scripts/prepare_release.ps1",
        "scripts/publish_release.ps1",
        "Expected 18 public release files",
        "PUBLIC_PLATFORM_ARTIFACTS=15",
    ):
        if obsolete in workflow:
            fail(f"release workflow still references obsolete release surface: {obsolete}")

    ci = read(".github/workflows/ci.yml")
    require(
        ci,
        (
            "name: Ghost FTP CI",
            "ProductName = \"Ghost FTP\"",
            "go test ./...",
            "go test -race ./...",
            "go vet ./...",
            "python scripts/audit_security.py",
            "python scripts/audit_privacy.py",
            "PHP syntax",
            "bash linux/BUILD.sh",
            ".\\BUILD-WINDOWS.ps1",
            "bash macos/BUILD.sh",
            "bash ios/BUILD.sh",
        ),
        ".github/workflows/ci.yml",
    )

    release_notes = read("scripts/release_notes.py")
    require(
        release_notes,
        (
            'tag = f"ghostftp-v{version}"',
            "Ghost FTP {version}",
            "Setup-x32.exe",
            "x32 and x86 refer to the same 32-bit architecture",
            "Android.apk",
            "iOS-arm64-unsigned.ipa",
            "Web.zip",
        ),
        "scripts/release_notes.py",
    )

    linux_build = read("linux/BUILD.sh")
    require(
        linux_build,
        (
            '"$root/usr/bin/ghostftp"',
            "linux/ghost-ftp.desktop",
            "Ghost-FTP-${VERSION}-Linux-${debarch}.deb",
            "build_arch amd64 amd64",
            "build_arch arm64 arm64",
            "build_arch 386 i386",
        ),
        "linux/BUILD.sh",
    )
    require(read("linux/debian/control.in"), ("Package: ghost-ftp",), "linux/debian/control.in")
    require(read("linux/ghost-ftp.desktop"), ("Name=Ghost FTP", "Exec=/usr/bin/ghostftp"), "linux/ghost-ftp.desktop")

    windows_build = read("BUILD-WINDOWS.ps1")
    require(
        windows_build,
        (
            "scripts/make_payload.py",
            "scripts/verify_release.py",
            "UNINSTALLER_BINARY",
            "unexpectedly produced an uninstaller binary",
        ),
        "BUILD-WINDOWS.ps1",
    )
    for forbidden in ("./cmd/uninstaller", "--uninstaller", "-Uninstall-", "'uninstaller'"):
        if forbidden in windows_build:
            fail(f"Windows build still contains obsolete uninstaller path: {forbidden}")

    # ByFTP.exe remains an intentional Windows upgrade-compatibility payload ID.
    payload = read("scripts/make_payload.py")
    require(payload, ("PAYLOAD_SCHEMA = 2", 'add(zf, args.app, "ByFTP.exe")'), "scripts/make_payload.py")
    installer = read("cmd/installer/main.go")
    require(
        installer,
        (
            'legacyUninstallKey = `Software\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\ByFTP`',
            'appPathsKey        = `Software\\Microsoft\\Windows\\CurrentVersion\\App Paths\\ByFTP.exe`',
            'appPath := filepath.Join(dir, "ByFTP.exe")',
            "All user-visible branding is Ghost FTP",
        ),
        "cmd/installer/main.go",
    )

    web_manifest = read("ByFTP WEB/manifest.webmanifest")
    require(web_manifest, ('"name": "Ghost FTP Remote File Client"', '"short_name": "Ghost FTP"'), "ByFTP WEB/manifest.webmanifest")

    changelog = read("CHANGELOG.md")
    require(changelog, (f"## {version} - 2026-09-04", "Legacy ByFTP history", "ghostftp-v1.0.0"), "CHANGELOG.md")

    obsolete_files = (
        "scripts/prepare_release.ps1",
        "scripts/publish_release.ps1",
        "scripts/package_windows_bundles.ps1",
        "scripts/audit_release_version_guard.py",
        "scripts/test_release_version_guard.py",
        "scripts/test_release_tools.py",
        "linux/byftp.desktop",
        "docs/images/byftp-header.png",
        "cmd/uninstaller",
    )
    for rel in obsolete_files:
        if (ROOT / rel).exists():
            fail(f"obsolete release/brand surface still exists: {rel}")

    print(f"RELEASE_AUDIT=PASS ({version})")
    print("PUBLIC_BRAND=Ghost FTP")
    print("RELEASE_TAG_NAMESPACE=ghostftp-vX.Y.Z")
    print("PUBLIC_PLATFORM_ARTIFACTS=8")
    print("PUBLIC_RELEASE_FILES=11")
    print("WINDOWS_X32_ALIAS_OF_X86=REQUIRED")
    print("HISTORICAL_TAG_REWRITE=BLOCKED")
    print("DUPLICATE_RELEASE_PUBLISHERS=REMOVED")
    return 0


if __name__ == "__main__":
    sys.exit(main())
