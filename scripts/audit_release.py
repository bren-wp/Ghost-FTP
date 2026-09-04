#!/usr/bin/env python3
"""Fail-closed validation of the Ghost FTP release and package contract."""

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
    if read("GhostFTP WEB/VERSION").strip() != version:
        fail("web VERSION does not match root VERSION")

    run("scripts/audit_brand_hardcut.py")
    run("scripts/audit_repository.py")
    run("scripts/audit_web.py")

    workflow = require(
        ".github/workflows/release.yml",
        "name: Publish Ghost FTP",
        "contents: write",
        "packages: write",
        "needs: [quality, windows, linux, macos, android, ios]",
        "RELEASE_TAG=ghostftp-v$version",
        "python scripts/audit_brand_hardcut.py",
        "python scripts/package_nuget.py",
        "dotnet nuget push",
        "/packages/nuget/GhostFTP/versions",
        "GITHUB_PACKAGE_READBACK=PASS",
        "Ghost-FTP-${VERSION}-Portable-x64.exe",
        "Ghost-FTP-${VERSION}-Portable-x86.exe",
        "Ghost-FTP-${VERSION}-Setup-x64.exe",
        "Ghost-FTP-${VERSION}-Setup-x86.exe",
        "Ghost-FTP-${VERSION}-Setup-x32.exe",
        "Ghost-FTP-${VERSION}-Linux-multiarch.zip",
        "Ghost-FTP-${VERSION}-macOS-Universal.pkg",
        "Ghost-FTP-${VERSION}-Android.apk",
        "Ghost-FTP-${VERSION}-iOS-arm64-unsigned.ipa",
        "Ghost-FTP-${VERSION}-Web.zip",
        "PUBLIC_PLATFORM_ARTIFACTS=10",
        "PUBLIC_RELEASE_FILES=13",
        "GITHUB_PACKAGE=GhostFTP",
        "main moved from release commit",
        "refusing to rewrite it",
        "RELEASE_ASSET_READBACK=PASS",
    )
    if "PUBLIC_PLATFORM_ARTIFACTS=8" in workflow or "PUBLIC_RELEASE_FILES=11" in workflow:
        fail("release workflow still exposes the 1.0.0 artifact count")

    require(
        ".github/workflows/ci.yml",
        "name: Ghost FTP CI",
        "GhostFTP WEB/VERSION",
        "ios/GhostFTP/Info.plist",
        'namespace = "com.ghostftp.client"',
        "python scripts/audit_brand_hardcut.py",
        "go test -race ./...",
        "Ghost-FTP-$v-$kind-$arch.exe",
    )

    require(
        "BUILD-WINDOWS.ps1",
        "function Build-GhostFTPArchitecture",
        '"Ghost-FTP-$version-Portable-$Label.exe"',
        '"Ghost-FTP-$version-Setup-$Label.exe"',
        "./cmd/ghostftp",
        "scripts/make_payload.py",
        "scripts/verify_release.py",
    )
    require(
        "cmd/installer/main.go",
        'uninstallKey = `Software\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\GhostFTP`',
        'appPathsKey  = `Software\\Microsoft\\Windows\\CurrentVersion\\App Paths\\GhostFTP.exe`',
        'appPath := filepath.Join(dir, "GhostFTP.exe")',
    )
    require("scripts/make_payload.py", "PAYLOAD_SCHEMA = 2", 'add(zf, args.app, "GhostFTP.exe")')

    require(
        "scripts/package_nuget.py",
        'PACKAGE_ID = "GhostFTP"',
        'output = output_dir / f"{PACKAGE_ID}.{version}.nupkg"',
        '"tools/win-x64/GhostFTP.exe"',
        '"tools/win-x86/GhostFTP.exe"',
        "NuGet package file set does not match the contract",
    )
    require("linux/BUILD.sh", '"$root/usr/bin/ghostftp"', "Ghost-FTP-${VERSION}-Linux-${debarch}.deb")
    require("linux/debian/control.in", "Package: ghost-ftp")
    require("macos/BUILD.sh", "Ghost-FTP-${VERSION}-macOS-Universal.pkg", "./cmd/ghostftp", "io.github.bren-wp.ghostftp")
    require("ios/BUILD.sh", "ios/GhostFTP.xcodeproj", "com.ghostftp.client", "scripts/package_ios.py")
    require("scripts/package_ios.py", 'f"Ghost-FTP-{version}-iOS-arm64-unsigned.ipa"', '"GhostFTP.app"')
    require("GhostFTP WEB/manifest.webmanifest", '"short_name": "Ghost FTP"')

    print(f"RELEASE_AUDIT=PASS ({version})")
    print("PUBLIC_BRAND=Ghost FTP")
    print("TECHNICAL_IDENTITY=GhostFTP")
    print("RELEASE_TAG_NAMESPACE=ghostftp-vX.Y.Z")
    print("PUBLIC_PLATFORM_ARTIFACTS=10")
    print("PUBLIC_RELEASE_FILES=13")
    print("GITHUB_PACKAGE_ID=GhostFTP")
    print("WINDOWS_PORTABLE=x64,x86")
    print("WINDOWS_X32_ALIAS_OF_X86=REQUIRED")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
