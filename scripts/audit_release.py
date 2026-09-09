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
    if parts < (0, 0, 1):
        fail("active public release baseline must not precede 0.0.1")

    run("scripts/audit_brand_hardcut.py")
    run("scripts/audit_repository.py")
    run("scripts/audit_platform_contract.py")
    run("scripts/audit_desktop_surface.py")

    workflow = require(
        ".github/workflows/release.yml",
        "name: Publish Ghost FTP",
        "workflow_dispatch:",
        "contents: write",
        "packages: write",
        "needs: [quality, windows, linux]",
        "test \"$version\" != '0.0.0'",
        "RELEASE_TAG=ghostftp-v$version",
        "release_channel='current'",
        "release_title=\"Ghost FTP $version\"",
        "remote_prerelease",
        "test \"$remote_prerelease\" = 'false'",
        "GHOSTFTP_SIGNING_PFX_BASE64",
        "GHOSTFTP_SIGNING_PASSWORD",
        "GHOSTFTP_SIGNING_TIMESTAMP_URL",
        "state=unsigned",
        "state=signed",
        "WINDOWS_AUTHENTICODE=${WINDOWS_SIGNING_STATE}",
        "python scripts/audit_platform_contract.py",
        "python scripts/audit_desktop_surface.py",
        "Ghost-FTP-${VERSION}-Portable-x64.exe",
        "Ghost-FTP-${VERSION}-Portable-x86.exe",
        "Ghost-FTP-${VERSION}-Setup-x64.exe",
        "Ghost-FTP-${VERSION}-Setup-x86.exe",
        "Ghost-FTP-${VERSION}-Setup-x32.exe",
        "Ghost-FTP-${VERSION}-Linux-amd64.deb",
        "Ghost-FTP-${VERSION}-Linux-amd64.tar.gz",
        "Ghost-FTP-${VERSION}-Linux-arm64.deb",
        "Ghost-FTP-${VERSION}-Linux-arm64.tar.gz",
        "Ghost-FTP-${VERSION}-Linux-i386.deb",
        "Ghost-FTP-${VERSION}-Linux-i386.tar.gz",
        "Ghost-FTP-${VERSION}-Linux-multiarch.zip",
        "Verify DEB and portable packages",
        "GHOSTFTP_REQUIRE_DEB: '1'",
        'cmp "$work/deb/usr/bin/ghostftp" "$root/ghostftp"',
        "LINUX_PORTABLE=amd64,arm64,i386",
        "PUBLIC_PLATFORM_ARTIFACTS=12",
        "PUBLIC_RELEASE_FILES=15",
        "ghcr.io/${owner}/ghost-ftp",
        "Distribution bundle only; not a supported runtime container.",
        "Publish verified bundle to GitHub Packages",
        "main moved from release commit",
        "release already exists; refusing to rewrite published assets",
        "RELEASE_ASSET_READBACK=PASS",
    )

    lowered = workflow.lower()
    for forbidden in (
        "package_nuget.py", "dotnet nuget", "nuget.pkg.github.com",
        "package_web.py", "audit_web.py", "android/", "ios/", "macos/", "runs-on: macos",
        "--prerelease",
    ):
        if forbidden in lowered:
            fail(f"release workflow contains retired/incompatible publication marker: {forbidden}")
    for forbidden in ("gh release upload", "--clobber"):
        if forbidden in workflow:
            fail(f"release workflow may rewrite current release assets: {forbidden}")
    if "New-DevCodeSigningCertificate.ps1" in workflow:
        fail("production release workflow must not create a self-signed publisher identity")

    retention = require(
        ".github/workflows/release-retention.yml",
        "name: Retain Latest Ghost FTP Release",
        "workflow_run:",
        "Publish Ghost FTP",
        "contents: write",
        "packages: write",
        "current_tag=\"ghostftp-v${version}\"",
        "test \"$release_draft\" = 'false'",
        "test \"$release_prerelease\" = 'false'",
        "test \"$asset_count\" -eq 15",
        "test \"$tag_sha\" = \"$main_sha\"",
        "gh release delete",
        "--cleanup-tag",
        "git/matching-refs/tags/ghostftp-v",
        "git/matching-refs/heads/release/ghostftp-v",
        "packages/container/ghost-ftp/versions",
        "Keeping current package version",
        "GHOSTFTP_RELEASE_RETENTION=PASS",
        "GHOSTFTP_PACKAGE_RETENTION=PASS (current=$version)",
        "LATEST_ONLY_RELEASE_RETENTION=YES",
    )
    retention_lowered = retention.lower()
    for forbidden in ("push --force", "update-ref -d refs/heads/main", "delete main"):
        if forbidden in retention_lowered:
            fail(f"release retention may rewrite main history: {forbidden}")

    trigger = require(
        ".github/workflows/release-branch-trigger.yml",
        "name: Trigger Ghost FTP Release",
        "startsWith(github.event.ref, 'release/ghostftp-v')",
        "source_version=\"$(tr -d '\\r\\n' < VERSION)\"",
        "gh workflow run release.yml",
        "test \"$GITHUB_SHA\" = \"$main_sha\"",
    )
    if "--force" in trigger:
        fail("release branch trigger must not force-move release identities")

    require(
        ".github/workflows/ci.yml",
        "name: Ghost FTP CI",
        "go test -race ./...",
        "Windows x64 and x86 production build",
        "Linux amd64 arm64 i386 production build",
        "Authenticode private-key pipeline smoke test",
        "Verify DEB and portable packages",
        "GHOSTFTP_REQUIRE_DEB: '1'",
    )
    require(
        "BUILD-WINDOWS.ps1",
        "function Build-GhostFTPArchitecture",
        "function Sign-WindowsTarget",
        "GHOSTFTP_SIGNING_PFX_PATH",
        "GHOSTFTP_SIGNING_PASSWORD",
        '"Ghost-FTP-$version-Portable-$Label.exe"',
        '"Ghost-FTP-$version-Setup-$Label.exe"',
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
    require(
        "linux/BUILD.sh",
        'binary="dist/.ghostftp-linux-${debarch}"',
        'portable_name="Ghost-FTP-${VERSION}-Linux-${debarch}"',
        'cp "$binary" "$portable_root/ghostftp"',
        'cp "$binary" "$deb_root/usr/bin/ghostftp"',
        "GHOSTFTP_REQUIRE_DEB",
        "tar --sort=name --owner=0 --group=0 --numeric-owner",
        "gzip -n -9",
    )
    require("linux/debian/control.in", "Package: ghost-ftp")
    require(
        "docs/PACKAGES.md",
        "ghcr.io/bren-wp/ghost-ftp",
        f"ghcr.io/bren-wp/ghost-ftp:{version}",
        "distribution bundle",
        "not a runtime container",
        "SHA256.txt",
        "latest",
    )

    for retired in ("android", "ios", "macos", "GhostFTP WEB"):
        if (ROOT / retired).exists():
            fail(f"retired application directory exists: {retired}/")
    for retired_file in (
        "scripts/package_nuget.py", "scripts/package_web.py", "scripts/test_package_web.py", "scripts/audit_web.py",
    ):
        if (ROOT / retired_file).exists():
            fail(f"retired release/tooling file exists: {retired_file}")

    print(f"RELEASE_AUDIT=PASS ({version}; channel=current)")
    print("PUBLIC_BRAND=Ghost FTP")
    print("TECHNICAL_IDENTITY=GhostFTP")
    print("RELEASE_TAG_NAMESPACE=ghostftp-vX.Y.Z")
    print("ACTIVE_APPLICATION_PLATFORMS=WINDOWS,LINUX")
    print("PUBLIC_RELEASE_CHANNEL=CURRENT")
    print("CURRENT_RELEASE_PRERELEASE_FLAG=FALSE")
    print("MINIMUM_PUBLIC_VERSION=0.0.1")
    print("LATEST_ONLY_RELEASE_RETENTION=YES")
    print("AUTHENTICODE_PRIVATE_KEY_IN_REPOSITORY=BLOCKED")
    print("CURRENT_WINDOWS_RELEASE_REQUIRES_TRUSTED_AUTHENTICODE=NO")
    print("TRUSTED_AUTHENTICODE_WHEN_CONFIGURED=VERIFIED")
    print("SELF_SIGNED_PRODUCTION_IDENTITY=BLOCKED")
    print("PUBLIC_PLATFORM_ARTIFACTS=12")
    print("PUBLIC_RELEASE_FILES=15")
    print("WINDOWS_PORTABLE=x64,x86")
    print("WINDOWS_X32_ALIAS_OF_X86=REQUIRED")
    print("LINUX_DEB=amd64,arm64,i386")
    print("LINUX_PORTABLE=amd64,arm64,i386")
    print("GHCR_CURRENT_BUNDLE=REQUIRED")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
