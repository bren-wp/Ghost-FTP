#!/usr/bin/env python3
"""Validate the Ghost FTP production release and package contract."""

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
    if read("GhostFTP WEB/VERSION").strip() != version:
        fail("web VERSION does not match repository VERSION")

    run_python_audit("scripts/audit_brand_hardcut.py", "hard-cut brand audit")
    run_python_audit("scripts/audit_repository.py", "repository-wide tracked-file audit")
    run_python_audit("scripts/audit_web.py", "Ghost FTP web audit/runtime gate")

    workflow = read(".github/workflows/release.yml")
    require(
        workflow,
        (
            "name: Publish Ghost FTP",
            "group: ghostftp-release",
            "permissions:\n  contents: write\n  packages: write",
            "quality:", "windows:", "linux:", "macos:", "android:", "ios:", "publish:",
            "needs: [quality, windows, linux, macos, android, ios]",
            "go telemetry off", "go test ./...", "go test -race ./...", "go vet ./...",
            "python scripts/audit_brand_hardcut.py",
            "python scripts/audit_security.py", "python scripts/audit_privacy.py",
            "python scripts/package_web.py --output-dir dist",
            ".\\BUILD-WINDOWS.ps1", "bash linux/BUILD.sh", "bash macos/BUILD.sh", "bash ios/BUILD.sh",
            ":app:testDebugUnitTest", ":app:lintDebug", ":app:assembleDebug",
            "RELEASE_TAG=ghostftp-v$version",
            "dist/Ghost-FTP-*-Setup-*.exe",
            "dist/Ghost-FTP-*-Portable-*.exe",
            'staging/windows/Ghost-FTP-${VERSION}-Setup-x64.exe',
            'staging/windows/Ghost-FTP-${VERSION}-Setup-x86.exe',
            'staging/windows/Ghost-FTP-${VERSION}-Portable-x64.exe',
            'staging/windows/Ghost-FTP-${VERSION}-Portable-x86.exe',
            "Ghost-FTP-${VERSION}-Setup-x32.exe",
            "Ghost-FTP-${VERSION}-Linux-multiarch.zip",
            "Ghost-FTP-${VERSION}-macOS-Universal.pkg",
            "Ghost-FTP-${VERSION}-Android.apk",
            "Ghost-FTP-${VERSION}-iOS-arm64-unsigned.ipa",
            "Ghost-FTP-${VERSION}-Web.zip",
            "PUBLIC_PLATFORM_ARTIFACTS=10",
            "PUBLIC_RELEASE_FILES=13",
            "GITHUB_PACKAGE=GhostFTP",
            "python scripts/package_nuget.py",
            'packages-out/GhostFTP.${VERSION}.nupkg',
            "dotnet nuget push",
            "/packages/nuget/GhostFTP/versions",
            "GITHUB_PACKAGE_READBACK=PASS",
            "main moved from release commit",
            "refusing to rewrite it",
            "gh release create",
            "SHA256.txt", "BUILD-METADATA.txt", "RELEASE-NOTES.txt",
        ),
        ".github/workflows/release.yml",
    )

    for obsolete in (
        "scripts/package_windows_bundles.ps1",
        "scripts/prepare_release.ps1",
        "scripts/publish_release.ps1",
        "Expected 18 public release files",
        "PUBLIC_PLATFORM_ARTIFACTS=8",
        "PUBLIC_RELEASE_FILES=11",
    ):
        if obsolete in workflow:
            fail(f"release workflow still references obsolete release surface: {obsolete}")

    ci = read(".github/workflows/ci.yml")
    require(
        ci,
        (
            "name: Ghost FTP CI",
            'module github.com/bren-wp/Ghost-FTP',
            "python scripts/audit_brand_hardcut.py",
            "go test ./...", "go test -race ./...", "go vet ./...",
            "python scripts/audit_security.py", "python scripts/audit_privacy.py", "PHP syntax",
            "bash linux/BUILD.sh", ".\\BUILD-WINDOWS.ps1", "bash macos/BUILD.sh", "bash ios/BUILD.sh",
            'dist\\Ghost-FTP-$v-$kind-$arch.exe',
            "GhostFTP WEB/VERSION",
            "ios/GhostFTP/Info.plist",
            'namespace = "com.ghostftp.client"',
        ),
        ".github/workflows/ci.yml",
    )

    windows_build = read("BUILD-WINDOWS.ps1")
    require(
        windows_build,
        (
            "function Build-GhostFTPArchitecture",
            '"Ghost-FTP-$version-Portable-$Label.exe"',
            '"Ghost-FTP-$version-Setup-$Label.exe"',
            "scripts/make_payload.py", "scripts/verify_release.py",
            "UNINSTALLER_BINARY", "unexpectedly produced an uninstaller binary",
            "./cmd/ghostftp",
        ),
        "BUILD-WINDOWS.ps1",
    )
    for forbidden in ("./cmd/uninstaller", "--uninstaller", "-Uninstall-", "'uninstaller'"):
        if forbidden in windows_build:
            fail(f"Windows build contains obsolete release path: {forbidden}")

    payload = read("scripts/make_payload.py")
    require(payload, ("PAYLOAD_SCHEMA = 2", 'add(zf, args.app, "GhostFTP.exe")'), "scripts/make_payload.py")

    installer = read("cmd/installer/main.go")
    require(
        installer,
        (
            'uninstallKey = `Software\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\GhostFTP`',
            'appPathsKey        = `Software\\Microsoft\\Windows\\CurrentVersion\\App Paths\\GhostFTP.exe`',
            'appPath := filepath.Join(dir, "GhostFTP.exe")',
            'brand.ProductName + " will be installed for your Windows user account',
        ),
        "cmd/installer/main.go",
    )
    if "uninstallKey" in installer:
        fail("Windows installer still exposes a legacy uninstall-key concept")

    require(
        read("scripts/package_nuget.py"),
        (
            'PACKAGE_ID = "GhostFTP"',
            'output = output_dir / f"{PACKAGE_ID}.{version}.nupkg"',
            '"tools/win-x64/GhostFTP.exe"',
            '"tools/win-x86/GhostFTP.exe"',
            '"NuGet package file set does not match the contract"',
        ),
        "scripts/package_nuget.py",
    )

    require(
        read("macos/BUILD.sh"),
        ("Ghost-FTP-${VERSION}-macOS-Universal.pkg", "./cmd/ghostftp", "io.github.bren-wp.ghostftp"),
        "macos/BUILD.sh",
    )
    require(
        read("ios/BUILD.sh"),
        ("ios/GhostFTP.xcodeproj", "com.ghostftp.client", "scripts/package_ios.py", "CURRENT_PROJECT_VERSION=\"$BUILD_NUMBER\""),
        "ios/BUILD.sh",
    )
    require(
        read("scripts/package_ios.py"),
        ('f"Ghost-FTP-{version}-iOS-arm64-unsigned.ipa"', '"com.ghostftp.client"', '"GhostFTP.app"'),
        "scripts/package_ios.py",
    )

    require(
        read("linux/BUILD.sh"),
        ('"$root/usr/bin/ghostftp"', "linux/ghost-ftp.desktop", "Ghost-FTP-${VERSION}-Linux-${debarch}.deb"),
        "linux/BUILD.sh",
    )
    require(read("linux/debian/control.in"), ("Package: ghost-ftp",), "linux/debian/control.in")
    require(read("linux/ghost-ftp.desktop"), ("Name=Ghost FTP", "Exec=/usr/bin/ghostftp"), "linux/ghost-ftp.desktop")

    web_manifest = read("GhostFTP WEB/manifest.webmanifest")
    require(web_manifest, ('"name": "Ghost FTP Remote File Client"', '"short_name": "Ghost FTP"'), "GhostFTP WEB/manifest.webmanifest")

    print(f"RELEASE_AUDIT=PASS ({version})")
    print("PUBLIC_BRAND=Ghost FTP")
    print("TECHNICAL_IDENTITY=GhostFTP")
    print("RELEASE_TAG_NAMESPACE=ghostftp-vX.Y.Z")
    print("PUBLIC_PLATFORM_ARTIFACTS=10")
    print("PUBLIC_RELEASE_FILES=13")
    print("GITHUB_PACKAGE_ID=GhostFTP")
    print("WINDOWS_X32_ALIAS_OF_X86=REQUIRED")
    print("WINDOWS_PORTABLE=x64,x86")
    print("HISTORICAL_TAG_REWRITE=BLOCKED")
    return 0


if __name__ == "__main__":
    sys.exit(main())
