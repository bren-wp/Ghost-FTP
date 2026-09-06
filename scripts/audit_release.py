#!/usr/bin/env python3
"""Fail-closed validation of the Ghost FTP Windows/Linux release contract."""

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
    return path.read_text(encoding="utf-8")


def require(rel: str, *markers: str) -> str:
    text = read(rel)
    for marker in markers:
        if marker not in text:
            fail(f"{rel} is missing required marker: {marker}")
    return text


def run(rel: str) -> None:
    try:
        subprocess.run([sys.executable, str(ROOT / rel)], cwd=ROOT, check=True)
    except subprocess.CalledProcessError as exc:
        fail(f"{rel} failed with exit code {exc.returncode}")


def main() -> int:
    version = read("VERSION").strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        fail(f"invalid VERSION: {version!r}")
    parts = tuple(int(part) for part in version.split("."))
    if parts < (0, 1, 0):
        fail("active release baseline must not precede 0.1.0")

    run("scripts/audit_brand_hardcut.py")
    run("scripts/audit_repository.py")
    run("scripts/audit_platform_contract.py")
    run("scripts/audit_desktop_surface.py")

    workflow = require(
        ".github/workflows/release.yml",
        "name: Publish Ghost FTP",
        "contents: write",
        "needs: [quality, windows, linux]",
        "RELEASE_TAG=ghostftp-v$version",
        "release_title=\"Ghost FTP $version Beta\"",
        "release_channel='beta'",
        "prerelease_args+=(--prerelease)",
        "GHOSTFTP_SIGNING_PFX_BASE64",
        "GHOSTFTP_SIGNING_PASSWORD",
        "GHOSTFTP_SIGNING_TIMESTAMP_URL",
        "WINDOWS_AUTHENTICODE=${WINDOWS_SIGNING_STATE}",
        "Stable Windows releases require a configured trusted Authenticode identity.",
        "python scripts/audit_platform_contract.py",
        "python scripts/audit_desktop_surface.py",
        "Ghost-FTP-${VERSION}-Portable-x64.exe",
        "Ghost-FTP-${VERSION}-Portable-x86.exe",
        "Ghost-FTP-${VERSION}-Setup-x64.exe",
        "Ghost-FTP-${VERSION}-Setup-x86.exe",
        "Ghost-FTP-${VERSION}-Setup-x32.exe",
        "Ghost-FTP-${VERSION}-Linux-amd64.deb",
        "Ghost-FTP-${VERSION}-Linux-arm64.deb",
        "Ghost-FTP-${VERSION}-Linux-i386.deb",
        "Ghost-FTP-${VERSION}-Linux-multiarch.zip",
        "PUBLIC_PLATFORM_ARTIFACTS=9",
        "PUBLIC_RELEASE_FILES=12",
        "main moved from release commit",
        "refusing to rewrite it",
        "RELEASE_ASSET_READBACK=PASS",
    )
    lowered = workflow.lower()
    for forbidden in (
        "packages: write", "package_nuget.py", "dotnet nuget", "nuget.pkg.github.com",
        "package_web.py", "audit_web.py", "android/", "ios/", "macos/", "runs-on: macos",
    ):
        if forbidden in lowered:
            fail(f"release workflow contains retired publication/platform marker: {forbidden}")

    require(
        ".github/workflows/ci.yml",
        "name: Ghost FTP CI",
        "python scripts/audit_platform_contract.py",
        "python scripts/audit_desktop_surface.py",
        "go test -race ./...",
        "Windows x64 and x86 production build",
        "Linux amd64 arm64 i386 production build",
        "Authenticode private-key pipeline smoke test",
        "New-DevCodeSigningCertificate.ps1",
        "Sign-WindowsArtifacts.ps1",
    )
    require(
        "BUILD-WINDOWS.ps1",
        "function Build-GhostFTPArchitecture",
        "function Sign-WindowsTarget",
        "GHOSTFTP_SIGNING_PFX_PATH",
        "GHOSTFTP_SIGNING_PASSWORD",
        "GHOSTFTP_SIGNING_TIMESTAMP_URL",
        '"Ghost-FTP-$version-Portable-$Label.exe"',
        '"Ghost-FTP-$version-Setup-$Label.exe"',
        "Sign-WindowsTarget -Path $portable",
        "scripts/make_payload.py",
        "Sign-WindowsTarget -Path $setup",
        "scripts/verify_release.py",
    )
    require(
        "cmd/installer/main.go",
        'uninstallKey = `Software\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\GhostFTP`',
        'appPathsKey  = `Software\\Microsoft\\Windows\\CurrentVersion\\App Paths\\GhostFTP.exe`',
        'appPath := filepath.Join(dir, "GhostFTP.exe")',
        "registerIntegratedUninstall(appPath, version)",
    )
    require(
        "cmd/installer/uninstall_registration_windows.go",
        '"UninstallString"', '"QuietUninstallString"', '"DisplayVersion"', '"NoModify"', '"NoRepair"',
    )
    require("scripts/make_payload.py", "PAYLOAD_SCHEMA = 2", 'add(zf, args.app, "GhostFTP.exe")')
    require("linux/BUILD.sh", '"$root/usr/bin/ghostftp"', "Ghost-FTP-${VERSION}-Linux-${debarch}.deb")
    require("linux/debian/control.in", "Package: ghost-ftp")

    for retired in ("android", "ios", "macos", "GhostFTP WEB"):
        if (ROOT / retired).exists():
            fail(f"retired application directory exists: {retired}/")
    for retired_file in (
        "scripts/package_nuget.py", "scripts/package_web.py", "scripts/test_package_web.py", "scripts/audit_web.py",
    ):
        if (ROOT / retired_file).exists():
            fail(f"retired release/tooling file exists: {retired_file}")

    channel = "beta" if parts[0] == 0 else "stable"
    print(f"RELEASE_AUDIT=PASS ({version}; channel={channel})")
    print("PUBLIC_BRAND=Ghost FTP")
    print("TECHNICAL_IDENTITY=GhostFTP")
    print("RELEASE_TAG_NAMESPACE=ghostftp-vX.Y.Z")
    print("ACTIVE_APPLICATION_PLATFORMS=WINDOWS,LINUX")
    print("PUBLICATION_SURFACE=GITHUB_RELEASE")
    print("PRE_1_0_CHANNEL=BETA")
    print("FIRST_STABLE_VERSION=1.0.0")
    print("AUTHENTICODE_PRIVATE_KEY_IN_REPOSITORY=BLOCKED")
    print("STABLE_WINDOWS_RELEASE_REQUIRES_TRUSTED_AUTHENTICODE=YES")
    print("PUBLIC_PLATFORM_ARTIFACTS=9")
    print("PUBLIC_RELEASE_FILES=12")
    print("WINDOWS_PORTABLE=x64,x86")
    print("WINDOWS_X32_ALIAS_OF_X86=REQUIRED")
    print("LINUX_DEB=amd64,arm64,i386")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
