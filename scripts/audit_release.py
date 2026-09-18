#!/usr/bin/env python3
"""Fail-closed validation of the Ghost FTP cross-platform release contract."""

from __future__ import annotations

import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PUBLIC_PLATFORM_ARTIFACTS = 13
PUBLIC_RELEASE_FILES = 16
BROWSERS = ("Chrome", "Edge", "Firefox", "Opera")
LINUX_DISTROS = ("Debian", "Ubuntu", "Fedora")


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


def forbid(rel: str, *markers: str) -> None:
    text = read(rel)
    lowered = text.lower()
    for marker in markers:
        if marker.lower() in lowered:
            fail(f"{rel} contains retired/incompatible release marker: {marker}")


def run(rel: str) -> None:
    try:
        subprocess.run([sys.executable, str(ROOT / rel)], cwd=ROOT, check=True)
    except subprocess.CalledProcessError as exc:
        fail(f"{rel} failed with exit code {exc.returncode}")


def main() -> int:
    version = read("VERSION").strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        fail(f"invalid VERSION: {version!r}")
    if tuple(int(part) for part in version.split(".")) < (0, 0, 1):
        fail("active public release baseline must not precede 0.0.1")

    for audit in (
        "scripts/audit_brand_hardcut.py",
        "scripts/audit_repository.py",
        "scripts/audit_platform_contract.py",
        "scripts/audit_desktop_surface.py",
    ):
        run(audit)

    workflow = require(
        ".github/workflows/release.yml",
        "name: Publish Ghost FTP",
        "workflow_dispatch:",
        "contents: write",
        "packages: write",
        "needs: [quality, windows, linux, android, browser]",
        "RELEASE_TAG=ghostftp-v$version",
        "release_channel='current'",
        "release_title=\"Ghost FTP $version\"",
        "test \"$remote_prerelease\" = 'false'",
        "Require protected Authenticode identity",
        "GHOSTFTP_SIGNING_PFX_BASE64",
        "GHOSTFTP_SIGNING_PASSWORD",
        "GHOSTFTP_SIGNING_TIMESTAMP_URL",
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
        "Windows universal x86 x64 ARM64 setup and portable",
        "Debian Ubuntu Fedora universal installer and portable bundles",
        "Chrome Edge Firefox Opera release packages",
        "WINDOWS_SETUP=universal-x86-x64-arm64",
        "WINDOWS_PORTABLE=universal-x86-x64-arm64",
        "WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64",
        "LINUX_DEBIAN_INSTALLER=universal-amd64-arm64-i386",
        "LINUX_DEBIAN_PORTABLE=universal-amd64-arm64-i386",
        "LINUX_UBUNTU_INSTALLER=universal-amd64-arm64-i386",
        "LINUX_UBUNTU_PORTABLE=universal-amd64-arm64-i386",
        "LINUX_FEDORA_INSTALLER=universal-amd64-arm64-i386",
        "LINUX_FEDORA_PORTABLE=universal-amd64-arm64-i386",
        "BROWSER_EXTENSION_PACKAGES=Chrome,Edge,Firefox,Opera",
        "BROWSER_DESKTOP_HANDOFF=sanitized-ghostftp-connect-no-autoconnect",
        f"PUBLIC_PLATFORM_ARTIFACTS={PUBLIC_PLATFORM_ARTIFACTS}",
        f"PUBLIC_RELEASE_FILES={PUBLIC_RELEASE_FILES}",
        f"test \"$count\" = '{PUBLIC_RELEASE_FILES}'",
        "ghcr.io/${owner}/ghost-ftp",
        "Distribution bundle only; not a supported runtime container.",
        "Publish verified bundle to GitHub Packages",
        "main moved from release commit",
        "release already exists; refusing to rewrite published assets",
        "RELEASE_ASSET_READBACK=PASS",
    )

    for artifact in (
        "Ghost-FTP-${VERSION}-Setup.exe",
        "Ghost-FTP-${VERSION}-Portable.exe",
        "Ghost-FTP-${VERSION}-Android.apk",
    ):
        if artifact not in workflow:
            fail(f"release workflow missing public artifact: {artifact}")
    for distro in LINUX_DISTROS:
        for suffix in ("Installer.run", "Portable.tar.gz"):
            artifact = f"Ghost-FTP-${{VERSION}}-Linux-{distro}-{suffix}"
            if artifact not in workflow:
                fail(f"release workflow missing Linux artifact: {artifact}")
    for browser in BROWSERS:
        artifact = f"Ghost-FTP-${{VERSION}}-{browser}-Extension.zip"
        if artifact not in workflow:
            fail(f"release workflow missing browser artifact: {artifact}")

    for forbidden in (
        "state=unsigned",
        "New-DevCodeSigningCertificate.ps1",
        "keytool -genkeypair",
        "--prerelease",
        "gh release upload",
        "--clobber",
        "Ghost-FTP-${VERSION}-Linux-Debian-amd64.deb",
        "Ghost-FTP-${VERSION}-Linux-Ubuntu-amd64.deb",
        "Ghost-FTP-${VERSION}-Linux-Fedora-x86_64.rpm",
        "Ghost-FTP-${VERSION}-Linux-Portable-amd64.tar.gz",
        "Ghost-FTP-Android-dev.apk",
        "Ghost-FTP-Android-ci-smoke.apk",
    ):
        if forbidden in workflow:
            fail(f"release workflow contains retired/incompatible publication marker: {forbidden}")
    lowered = workflow.lower()
    for forbidden in ("ios/", "macos/", "runs-on: macos", "package_nuget.py", "nuget.pkg.github.com"):
        if forbidden in lowered:
            fail(f"release workflow contains unsupported public surface: {forbidden}")

    retention = require(
        ".github/workflows/release-retention.yml",
        "workflow_run:",
        "Publish Ghost FTP",
        "contents: write",
        "packages: write",
        "current_tag=\"ghostftp-v${version}\"",
        "protected_tag=\"ghostftp-v0.0.7\"",
        "keep_release_tag",
        "test \"$release_draft\" = 'false'",
        "test \"$release_prerelease\" = 'false'",
        f"test \"$asset_count\" -eq {PUBLIC_RELEASE_FILES}",
        "test \"$tag_sha\" = \"$main_sha\"",
        "gh release delete",
        "--cleanup-tag",
        "git/matching-refs/tags/ghostftp-v",
        "git/matching-refs/heads/release/ghostftp-v",
        "packages/container/ghost-ftp/versions",
        "GHOSTFTP_RELEASE_RETENTION=PASS",
        "preserves-0.0.7-if-present",
        "PROTECTED_RELEASE_TAG=ghostftp-v0.0.7",
        "PROTECTED_RELEASE_POLICY=PRESERVE_TAG_RELEASE_AND_EXISTING_PACKAGE",
    )
    for forbidden in ("push --force", "update-ref -d refs/heads/main", "delete main"):
        if forbidden in retention.lower():
            fail(f"release retention may rewrite main history: {forbidden}")

    trigger = require(
        ".github/workflows/release-branch-trigger.yml",
        "name: Trigger Ghost FTP Release",
        "startsWith(github.event.ref, 'release/ghostftp-v')",
        "source_version=\"$(tr -d '\\r\\n' < VERSION)\"",
        "gh workflow run release.yml",
        "test \"$GITHUB_SHA\" = \"$main_sha\"",
        "gh run watch \"$release_run_id\"",
        "test \"$release_conclusion\" = 'success'",
        "gh workflow run release-retention.yml",
        "test \"$retention_conclusion\" = 'success'",
        "RELEASE_RETENTION_CHAIN=PASS",
    )
    if "--force" in trigger:
        fail("release branch trigger must not force-move release identities")
    if trigger.index("test \"$release_conclusion\" = 'success'") > trigger.index("gh workflow run release-retention.yml"):
        fail("release retention may be dispatched before canonical publication succeeds")

    require(
        ".github/workflows/ci.yml",
        "name: Ghost FTP CI",
        "go test -race ./...",
        "Windows universal setup and portable production build",
        "Linux universal distro bundles with amd64 arm64 i386 payloads",
        "Authenticode private-key pipeline smoke test",
        "Verify installer and portable bundles",
        "GHOSTFTP_PREFIX",
    )
    android_validation = require(
        ".github/workflows/android-apk.yml",
        "ANDROID_RELEASE_SIGNING_PIPELINE_SMOKE=PASS",
        "ANDROID_RELEASE_SIGNING_PIPELINE_SMOKE_IDENTITY=EPHEMERAL_CI_ONLY",
        ":app:assembleRelease",
        "android/app/build/outputs/apk/release/app-release-unsigned.apk",
    )
    for retired in (
        "Ghost-FTP-Android-dev.apk",
        "packageGhostFtpApk",
        "versionNameSuffix '-dev'",
    ):
        if retired in android_validation:
            fail(f"Android validation workflow contains retired development artifact marker: {retired}")
    require(
        ".github/workflows/browser-extensions.yml",
        "Build official Chrome Edge Firefox Opera packages",
        "for browser in Chrome Edge Firefox Opera",
        "Ghost-FTP-${version}-${browser}-Extension.zip",
    )

    require(
        "linux/BUILD-DISTROS.sh",
        "build_arch amd64 amd64",
        "build_arch arm64 arm64",
        "build_arch 386 i386",
        "for distro in Debian Ubuntu Fedora; do",
        'Ghost-FTP-${VERSION}-Linux-${distro}-Installer.run',
        'Ghost-FTP-${VERSION}-Linux-${distro}-Portable',
        "__GHOSTFTP_PAYLOAD_BELOW__",
        "GHOSTFTP_PREFIX",
        'exec "$base_dir/bin/$arch/ghostftp" "$@"',
        "tar --sort=name --owner=0 --group=0 --numeric-owner",
        "gzip -n -9",
    )
    forbid("linux/BUILD-DISTROS.sh", "dpkg-deb", "rpmbuild")

    require(
        "scripts/verify_release_digest_readback.py",
        f"EXPECTED_RELEASE_FILES = {PUBLIC_RELEASE_FILES}",
        'for distro in ("Debian", "Ubuntu", "Fedora")',
        'for browser in ("Chrome", "Edge", "Firefox", "Opera")',
        'f"Ghost-FTP-{version}-Android.apk"',
    )

    require(
        "BUILD-WINDOWS.ps1",
        "function Build-UniversalBootstrap",
        "function Sign-UniversalTarget",
        "GHOSTFTP_SIGNING_PFX_PATH",
        "GHOSTFTP_SIGNING_PASSWORD",
        "./cmd/windowsbootstrap",
        '"Ghost-FTP-$version-Portable.exe"',
        '"Ghost-FTP-$version-Setup.exe"',
        "WINDOWS_PUBLIC_SETUP=UNIVERSAL_X86_X64_ARM64",
        "WINDOWS_PUBLIC_PORTABLE=UNIVERSAL_X86_X64_ARM64",
        "WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64",
        "WINDOWS_PUBLIC_EXECUTABLES=2",
    )
    require(
        "scripts/verify_release.py",
        'PUBLIC_WINDOWS_RELEASE_WORKFLOW = "Publish Ghost FTP"',
        "public Windows release artifacts must be Authenticode signed",
        'print("WINDOWS_ARCH=universal-x86-x64-arm64")',
        'print("WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64")',
    )

    print(f"RELEASE_AUDIT=PASS ({version}; channel=current)")
    print("MINIMUM_PUBLIC_VERSION=0.0.1")
    print("PUBLIC_RELEASE_CHANNEL=CURRENT")
    print("CURRENT_RELEASE_PRERELEASE_FLAG=FALSE")
    print("PROTECTED_RELEASE_TAG=ghostftp-v0.0.7")
    print(f"PUBLIC_PLATFORM_ARTIFACTS={PUBLIC_PLATFORM_ARTIFACTS}")
    print(f"PUBLIC_RELEASE_FILES={PUBLIC_RELEASE_FILES}")
    print("WINDOWS_SETUP=UNIVERSAL_X86_X64_ARM64")
    print("WINDOWS_PORTABLE=UNIVERSAL_X86_X64_ARM64")
    print("WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64")
    print("WINDOWS_ARM64_RUNTIME_EVIDENCE=NOT_NATIVE_CI")
    print("LINUX_DEBIAN_INSTALLER=UNIVERSAL_AMD64_ARM64_I386")
    print("LINUX_DEBIAN_PORTABLE=UNIVERSAL_AMD64_ARM64_I386")
    print("LINUX_UBUNTU_INSTALLER=UNIVERSAL_AMD64_ARM64_I386")
    print("LINUX_UBUNTU_PORTABLE=UNIVERSAL_AMD64_ARM64_I386")
    print("LINUX_FEDORA_INSTALLER=UNIVERSAL_AMD64_ARM64_I386")
    print("LINUX_FEDORA_PORTABLE=UNIVERSAL_AMD64_ARM64_I386")
    print("LINUX_BUNDLE_ARCHITECTURES=AMD64,ARM64,I386")
    print("ANDROID_PUBLIC_RELEASE_ARTIFACT=YES_PRODUCTION_SIGNED")
    print("BROWSER_PUBLIC_RELEASE_PACKAGES=CHROME,EDGE,FIREFOX,OPERA")
    print("GHCR_CURRENT_BUNDLE=REQUIRED")
    print("CURRENT_WINDOWS_RELEASE_REQUIRES_TRUSTED_AUTHENTICODE=YES")
    print("PUBLIC_WINDOWS_AUTHENTICODE=REQUIRED_AND_VERIFIED")
    print("ANDROID_PRODUCTION_SIGNING_IDENTITY=REQUIRED_AND_VERIFIED")
    return 0


if __name__ == "__main__":
    sys.exit(main())
