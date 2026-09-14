#!/usr/bin/env python3
"""Fail-closed validation of the Ghost FTP cross-platform release contract."""

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
        "needs: [quality, windows, linux, android, browser]",
        "test \"$version\" != '0.0.0'",
        "RELEASE_TAG=ghostftp-v$version",
        "release_channel='current'",
        "release_title=\"Ghost FTP $version\"",
        "remote_prerelease",
        "test \"$remote_prerelease\" = 'false'",
        "GHOSTFTP_SIGNING_PFX_BASE64",
        "GHOSTFTP_SIGNING_PASSWORD",
        "GHOSTFTP_SIGNING_TIMESTAMP_URL",
        "Require protected Authenticode identity",
        "Official Ghost FTP publication requires GHOSTFTP_SIGNING_PFX_BASE64.",
        "Official Ghost FTP publication requires GHOSTFTP_SIGNING_PASSWORD.",
        "Get-AuthenticodeSignature -FilePath $path",
        "WINDOWS_AUTHENTICODE=${WINDOWS_SIGNING_STATE}",
        "Require protected Android production signing identity",
        "GHOSTFTP_ANDROID_KEYSTORE_BASE64",
        "GHOSTFTP_ANDROID_KEYSTORE_PASSWORD",
        "GHOSTFTP_ANDROID_KEY_ALIAS",
        "GHOSTFTP_ANDROID_KEY_PASSWORD",
        "GHOSTFTP_ANDROID_CERT_SHA256",
        "apksigner\" sign",
        "apksigner\" verify --verbose --print-certs",
        "ANDROID_PRODUCTION_SIGNATURE=PASS",
        "ANDROID_APK=production-signed",
        "ANDROID_SIGNER_SHA256=${ANDROID_SIGNER_SHA256}",
        "ANDROID_SFTP=hidden-until-strict-host-key-verification",
        "BROWSER_EXTENSION_PACKAGES=Chrome,Edge,Firefox",
        "BROWSER_DESKTOP_HANDOFF=unsupported",
        "python scripts/audit_platform_contract.py",
        "python scripts/audit_desktop_surface.py",
        "Ghost-FTP-${VERSION}-Portable.exe",
        "Ghost-FTP-${VERSION}-Setup.exe",
        "Ghost-FTP-${VERSION}-Android.apk",
        "Ghost-FTP-${VERSION}-Chrome-Extension.zip",
        "Ghost-FTP-${VERSION}-Edge-Extension.zip",
        "Ghost-FTP-${VERSION}-Firefox-Extension.zip",
        "Ghost-FTP-${VERSION}-Linux-Debian-amd64.deb",
        "Ghost-FTP-${VERSION}-Linux-Debian-arm64.deb",
        "Ghost-FTP-${VERSION}-Linux-Debian-i386.deb",
        "Ghost-FTP-${VERSION}-Linux-Ubuntu-amd64.deb",
        "Ghost-FTP-${VERSION}-Linux-Ubuntu-arm64.deb",
        "Ghost-FTP-${VERSION}-Linux-Ubuntu-i386.deb",
        "Ghost-FTP-${VERSION}-Linux-Fedora-x86_64.rpm",
        "Ghost-FTP-${VERSION}-Linux-Fedora-aarch64.rpm",
        "Ghost-FTP-${VERSION}-Linux-Fedora-i686.rpm",
        "Ghost-FTP-${VERSION}-Linux-Portable-amd64.tar.gz",
        "Ghost-FTP-${VERSION}-Linux-Portable-arm64.tar.gz",
        "Ghost-FTP-${VERSION}-Linux-Portable-i386.tar.gz",
        "Build distro packages",
        "Verify distro package metadata and binary parity",
        "GHOSTFTP_REQUIRE_DEB: '1'",
        "GHOSTFTP_REQUIRE_RPM: '1'",
        "bash linux/BUILD-DISTROS.sh",
        'cmp "$work/${distro,,}/usr/bin/ghostftp" "$portable_root/ghostftp"',
        'cmp "$work/fedora/usr/bin/ghostftp" "$portable_root/ghostftp"',
        "python scripts/build_browser_extensions.py",
        "WINDOWS_SETUP=universal-x86-x64-arm64",
        "WINDOWS_PORTABLE=universal-x86-x64-arm64",
        "WINDOWS_BOOTSTRAP_PE=x86",
        "WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64",
        "LINUX_DEBIAN_DEB=amd64,arm64,i386",
        "LINUX_UBUNTU_DEB=amd64,arm64,i386",
        "LINUX_FEDORA_RPM=x86_64,aarch64,i686",
        "LINUX_PORTABLE=amd64,arm64,i386",
        "PUBLIC_PLATFORM_ARTIFACTS=18",
        "PUBLIC_RELEASE_FILES=21",
        "test \"$count\" = '21'",
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
        "package_web.py", "audit_web.py", "ios/", "macos/", "runs-on: macos",
        "--prerelease",
        "linux-multiarch.zip",
        "linux-amd64.deb",
        "linux-arm64.deb",
        "linux-i386.deb",
        "portable-x64.exe",
        "portable-x86.exe",
        "portable-arm64.exe",
        "setup-x64.exe",
        "setup-x86.exe",
        "setup-x32.exe",
        "setup-arm64.exe",
        "state=unsigned",
        "publishing current release with explicitly unsigned windows artifacts",
        "ghost-ftp-android-dev.apk",
        "ghost-ftp-android-ci-smoke.apk",
    ):
        if forbidden in lowered:
            fail(f"release workflow contains retired/incompatible publication marker: {forbidden}")
    for forbidden in ("gh release upload", "--clobber"):
        if forbidden in workflow:
            fail(f"release workflow may rewrite current release assets: {forbidden}")
    if "New-DevCodeSigningCertificate.ps1" in workflow:
        fail("production release workflow must not create a self-signed publisher identity")
    if "keytool -genkeypair" in workflow:
        fail("production release workflow must not generate its Android publisher identity")

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
        "test \"$asset_count\" -eq 21",
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
        "wait_for_new_run()",
        "--json databaseId,headSha",
        "gh run watch \"$release_run_id\"",
        "test \"$release_conclusion\" = 'success'",
        "gh workflow run release-retention.yml",
        "gh run watch \"$retention_run_id\"",
        "test \"$retention_conclusion\" = 'success'",
        "RELEASE_RETENTION_CHAIN=PASS",
    )
    if "--force" in trigger:
        fail("release branch trigger must not force-move release identities")
    if trigger.index("test \"$release_conclusion\" = 'success'") > trigger.index("gh workflow run release-retention.yml"):
        fail("release retention may be dispatched before the canonical release succeeds")

    require(
        ".github/workflows/ci.yml",
        "name: Ghost FTP CI",
        "go test -race ./...",
        "Windows universal setup and portable production build",
        "Linux amd64 arm64 i386 production build",
        "Authenticode private-key pipeline smoke test",
        "Ghost-FTP-$v-$kind.exe",
        "Architecture-specific Windows executables leaked into public CI artifacts",
        "Verify DEB and portable packages",
        "GHOSTFTP_REQUIRE_DEB: '1'",
    )
    require(
        ".github/workflows/android-apk.yml",
        "ANDROID_RELEASE_SIGNING_PIPELINE_SMOKE=PASS",
        "ANDROID_RELEASE_SIGNING_PIPELINE_SMOKE_IDENTITY=EPHEMERAL_CI_ONLY",
        "Ghost-FTP-Android-dev.apk",
    )
    require(
        ".github/workflows/browser-extensions.yml",
        "Build official Chrome Edge Firefox packages",
        "Ghost-FTP-${version}-${browser}-Extension.zip",
    )
    require(
        "BUILD-WINDOWS.ps1",
        "BUILD-WINDOWS-ARCH-STAGE.ps1",
        "function Build-UniversalBootstrap",
        "function Sign-UniversalTarget",
        "GHOSTFTP_SIGNING_PFX_PATH",
        "GHOSTFTP_SIGNING_PASSWORD",
        "./cmd/windowsbootstrap",
        '"Ghost-FTP-$version-Portable.exe"',
        '"Ghost-FTP-$version-Setup.exe"',
        "scripts/verify_release.py",
        "'--arch','universal'",
        "WINDOWS_PUBLIC_SETUP=UNIVERSAL_X86_X64_ARM64",
        "WINDOWS_PUBLIC_PORTABLE=UNIVERSAL_X86_X64_ARM64",
        "WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64",
        "WINDOWS_PUBLIC_EXECUTABLES=2",
    )
    require(
        "scripts/verify_release.py",
        'PUBLIC_WINDOWS_RELEASE_WORKFLOW = "Publish Ghost FTP"',
        "ARM64 = 0xAA64",
        '"arm64": {"machine": ARM64',
        "public Windows release artifacts must be Authenticode signed",
        "require_public_release_signatures(ssigned, psigned)",
        'print("WINDOWS_ARCH=universal-x86-x64-arm64")',
        'print("WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64")',
    )
    require(
        "scripts/verify_release_digest_readback.py",
        "EXPECTED_RELEASE_FILES = 21",
        'f"Ghost-FTP-{version}-Android.apk"',
        'f"Ghost-FTP-{version}-Chrome-Extension.zip"',
        'f"Ghost-FTP-{version}-Edge-Extension.zip"',
        'f"Ghost-FTP-{version}-Firefox-Extension.zip"',
    )
    require(
        "scripts/pe_resources.py",
        "ARM64 = 0xAA64",
        'processor_architecture not in {"amd64", "x86", "arm64"}',
        'machine == ARM64 and magic == 0x20B',
    )
    require(
        "BUILD-WINDOWS-ARCH-STAGE.ps1",
        "function Build-GhostFTPArchitecture",
        "function Sign-WindowsTarget",
        "GHOSTFTP_SIGNING_PFX_PATH",
        "GHOSTFTP_SIGNING_PASSWORD",
        '"Ghost-FTP-$version-Portable-$Label.exe"',
        '"Ghost-FTP-$version-Setup-$Label.exe"',
        "./cmd/installer",
        "scripts/make_payload.py",
        "scripts/verify_release.py",
        "Build-GhostFTPArchitecture -GoArch 'arm64' -Label 'arm64'",
        "WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64",
    )
    require(
        "cmd/windowsbootstrap/main.go",
        "platform.NativeWindowsArchitecture()",
        'case "x86", "x64", "arm64":',
        'return "payload/" + arch + "/GhostFTP.exe", nil',
        "os.CreateTemp(localAppData",
        "verifyStaged(path, data)",
        "cmd.Run()",
        "platform.HardenProcessPrivacy()",
    )
    require(
        "internal/platform/windows_arch_windows.go",
        'NewProc("GetNativeSystemInfo")',
        "windowsArchitectureFromProcessor(info.ProcessorArchitecture)",
    )
    require(
        "internal/platform/windows_arch.go",
        "windowsProcessorArchitectureARM64 = 12",
        'return "arm64", nil',
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
        "linux/BUILD-DISTROS.sh",
        'portable_name="Ghost-FTP-${VERSION}-Linux-Portable-${debarch}"',
        'rpm_out="dist/Ghost-FTP-${VERSION}-Linux-Fedora-${rpmarch}.rpm"',
        'for distro in Debian Ubuntu; do',
        'deb_out="dist/Ghost-FTP-${VERSION}-Linux-${slug}-${debarch}.deb"',
        'cp "$binary" "$portable_root/ghostftp"',
        'cp "$binary" "$deb_root/usr/bin/ghostftp"',
        "GHOSTFTP_REQUIRE_DEB",
        "GHOSTFTP_REQUIRE_RPM",
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

    for required_macos in (
        "macos/README.md",
        "macos/PARITY.md",
        "macos/BUILD.sh",
        ".github/workflows/macos-app.yml",
    ):
        if not (ROOT / required_macos).is_file():
            fail(f"active macOS development source is incomplete: {required_macos}")

    for retired in ("ios", "GhostFTP WEB"):
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
    print("PUBLIC_RELEASE_PLATFORMS=WINDOWS,LINUX,ANDROID,BROWSER_HELPER")
    print("ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID,MACOS")
    print("ANDROID_PUBLIC_RELEASE_ARTIFACT=YES_PRODUCTION_SIGNED")
    print("ANDROID_SFTP_PUBLIC_SUPPORT=NO_STRICT_HOST_KEY_BOUNDARY")
    print("BROWSER_PUBLIC_RELEASE_PACKAGES=CHROME,EDGE,FIREFOX")
    print("BROWSER_DESKTOP_HANDOFF=UNSUPPORTED")
    print("MACOS_PUBLIC_RELEASE_ARTIFACT=NO")
    print("PUBLIC_RELEASE_CHANNEL=CURRENT")
    print("CURRENT_RELEASE_PRERELEASE_FLAG=FALSE")
    print("MINIMUM_PUBLIC_VERSION=0.0.1")
    print("LATEST_ONLY_RELEASE_RETENTION=YES")
    print("RELEASE_RETENTION_CHAIN=REQUIRED")
    print("AUTHENTICODE_PRIVATE_KEY_IN_REPOSITORY=BLOCKED")
    print("CURRENT_WINDOWS_RELEASE_REQUIRES_TRUSTED_AUTHENTICODE=YES")
    print("PUBLIC_WINDOWS_AUTHENTICODE=REQUIRED_AND_VERIFIED")
    print("ANDROID_PRODUCTION_SIGNING_IDENTITY=REQUIRED_AND_VERIFIED")
    print("SELF_SIGNED_PRODUCTION_IDENTITY=BLOCKED")
    print("PUBLIC_PLATFORM_ARTIFACTS=18")
    print("PUBLIC_RELEASE_FILES=21")
    print("WINDOWS_SETUP=UNIVERSAL_X86_X64_ARM64")
    print("WINDOWS_PORTABLE=UNIVERSAL_X86_X64_ARM64")
    print("WINDOWS_BOOTSTRAP_PE=x86")
    print("WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64")
    print("WINDOWS_ARM64_RUNTIME_EVIDENCE=NOT_NATIVE_CI")
    print("LINUX_DEBIAN_DEB=amd64,arm64,i386")
    print("LINUX_UBUNTU_DEB=amd64,arm64,i386")
    print("LINUX_FEDORA_RPM=x86_64,aarch64,i686")
    print("LINUX_PORTABLE=amd64,arm64,i386")
    print("GHCR_CURRENT_BUNDLE=REQUIRED")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
