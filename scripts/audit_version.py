#!/usr/bin/env python3
"""Verify canonical Ghost FTP versioning across public release and active source surfaces."""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
VERSION_RE = re.compile(r"^\d+\.\d+\.\d+$")
GO_TOOLCHAIN = "1.27.1"
RETIRED_ROOTS = ("ios", "macos", "GhostFTP WEB")
CURRENT_LINE_DOCS = (
    "README.md",
    "CHANGELOG.md",
    "docs/README.md",
    "docs/GITHUB-RELEASES.md",
    "docs/INSTALLATION.md",
    "docs/PACKAGES.md",
    "docs/REFERENCE-UI.md",
    "docs/RELEASE-HISTORY.md",
    "docs/RELEASE-VERIFICATION.md",
    "docs/SETTINGS.md",
    "docs/SUPPORT.md",
    "docs/TESTING.md",
    "docs/VERSIONING.md",
)
RETIRED_PUBLIC_VERSION_PATTERNS = (
    re.compile(r"Ghost FTP(?:\s+\*\*)?\s*1\.\d+\.\d+"),
    re.compile(r"Ghost-FTP-1\.\d+\.\d+"),
    re.compile(r"ghostftp-v1\.\d+\.\d+"),
    re.compile(r"ghcr\.io/bren-wp/ghost-ftp:1\.\d+\.\d+"),
    re.compile(r"\bVERSION=1\.\d+\.\d+\b"),
    re.compile(r"\bTAG=ghostftp-v1\.\d+\.\d+\b"),
)


def fail(message: str) -> None:
    raise SystemExit("VERSION_AUDIT_FAILED: " + message)


def read(path: str) -> str:
    target = ROOT / path
    if not target.is_file():
        fail(f"missing {path}")
    return target.read_text(encoding="utf-8")


def require(text: str, markers: tuple[str, ...], where: str) -> None:
    for marker in markers:
        if marker not in text:
            fail(f"{where} is missing version/platform binding: {marker}")


def main() -> int:
    version = read("VERSION").strip()
    if not VERSION_RE.fullmatch(version):
        fail(f"VERSION is not semantic: {version!r}")
    parts = tuple(int(part) for part in version.split("."))
    if parts < (0, 0, 1):
        fail("public VERSION must be 0.0.1 or newer; 0.0.0 is reserved")

    if f"go {GO_TOOLCHAIN}" not in read("go.mod"):
        fail(f"go.mod must use Go {GO_TOOLCHAIN}")

    for rel in ("cmd/ghostftp/main.go", "cmd/installer/main.go", "cmd/windowsbootstrap/main.go"):
        text = read(rel)
        if 'var version = "dev"' not in text:
            fail(f"{rel} must retain the development version fallback")
        if re.search(r'var\s+version\s*=\s*"\d+\.\d+\.\d+"', text):
            fail(f"{rel} hard-codes a production version")

    brand_version = read("internal/brand/version.go")
    require(brand_version, ('strings.TrimSpace(version)', 'return "dev"', 'return version'), "internal/brand/version.go")
    if 'return version + " Beta"' in brand_version or 'strings.HasPrefix(version, "0.")' in brand_version:
        fail("product display version must not infer prerelease status from major version 0")

    readme = read("README.md")
    require(
        readme,
        (
            f"Current Ghost FTP version: **{version}**",
            "Development status: **Active**",
            "Release channel: **Current**",
            f"## {version}",
            f"ghostftp-v{version}",
            "prerelease=false",
            f"ghcr.io/bren-wp/ghost-ftp:{version}",
        ),
        "README.md",
    )
    if f"## {version}" not in read("CHANGELOG.md"):
        fail("CHANGELOG does not contain a section for VERSION")

    versioning = read("docs/VERSIONING.md")
    require(
        versioning,
        (
            f"Current source candidate: **{version}**",
            f"VERSION={version}",
            f"TAG=ghostftp-v{version}",
            "CHANNEL=Current",
            "PRERELEASE=false",
            f"ghcr.io/bren-wp/ghost-ftp:{version}",
            f"## {version} release checklist",
            "0.0.0",
            "0.0.1",
            "0.0.2",
            "major version `0` does not imply prerelease",
            "latest public version",
            "release-retention.yml",
            "optional production hardening layer",
            "WINDOWS_AUTHENTICODE=unsigned",
            "Absence of a production Authenticode certificate by itself is not a versioning failure.",
        ),
        "docs/VERSIONING.md",
    )

    for rel in CURRENT_LINE_DOCS:
        text = read(rel)
        for pattern in RETIRED_PUBLIC_VERSION_PATTERNS:
            match = pattern.search(text)
            if match:
                fail(f"active current-line documentation contains retired public identity {match.group(0)!r}: {rel}")

    windows_build = read("BUILD-WINDOWS.ps1")
    require(windows_build, ("Get-Content -LiteralPath $versionFile", "-X main.version=$version", "WINDOWS_PUBLIC_EXECUTABLES=2"), "BUILD-WINDOWS.ps1")
    windows_stage = read("BUILD-WINDOWS-ARCH-STAGE.ps1")
    require(windows_stage, ("Get-Content -LiteralPath $versionFile", "-X main.version=$version"), "BUILD-WINDOWS-ARCH-STAGE.ps1")
    linux_build = read("linux/BUILD.sh")
    require(linux_build, ("< VERSION", "-X main.version=${VERSION}"), "linux/BUILD.sh")
    linux_distro_build = read("linux/BUILD-DISTROS.sh")
    require(
        linux_distro_build,
        (
            "< VERSION",
            "-X main.version=${VERSION}",
            "for distro in Debian Ubuntu; do",
            'portable_name="Ghost-FTP-${VERSION}-Linux-Portable-${debarch}"',
            'rpm_out="dist/Ghost-FTP-${VERSION}-Linux-Fedora-${rpmarch}.rpm"',
        ),
        "linux/BUILD-DISTROS.sh",
    )
    local_build = read("scripts/BUILD-LOCAL.sh")
    require(local_build, ("< VERSION", "-X main.version=$VERSION"), "scripts/BUILD-LOCAL.sh")

    linux_control = read("linux/debian/control.in")
    if "@VERSION@" not in linux_control or re.search(r"(?m)^Version:\s*\d+\.\d+\.\d+", linux_control):
        fail("Linux DEB metadata is not bound to VERSION")

    android_build = read("android/app/build.gradle")
    require(
        android_build,
        (
            "rootProject.file('../VERSION').text.trim()",
            'versionName "${ghostFtpVersion}-dev"',
            "namespace 'app.ghostftp.client'",
            "applicationId 'app.ghostftp.client'",
            "tasks.register('packageGhostFtpApk', Copy)",
            "'Ghost-FTP-Android.apk'",
        ),
        "android/app/build.gradle",
    )
    android_workflow = read(".github/workflows/android-apk.yml")
    require(
        android_workflow,
        (
            "Ghost FTP Android APK",
            "packageGhostFtpApk",
            "android/dist/Ghost-FTP-Android.apk",
            "name: ghostftp-android-apk",
        ),
        ".github/workflows/android-apk.yml",
    )

    for retired in RETIRED_ROOTS:
        if (ROOT / retired).exists():
            fail(f"retired application surface must be removed: {retired}/")

    workflow_build_markers = (
        (".github/workflows/ci.yml", "bash linux/BUILD.sh"),
        (".github/workflows/release.yml", "bash linux/BUILD-DISTROS.sh"),
    )
    for workflow_rel, linux_marker in workflow_build_markers:
        workflow = read(workflow_rel)
        if f"go-version: '{GO_TOOLCHAIN}'" not in workflow:
            fail(f"{workflow_rel} does not pin Go {GO_TOOLCHAIN}")
        require(workflow, ("windows:", "linux:", linux_marker), workflow_rel)
        lowered = workflow.lower()
        for marker in ("ios/", "macos/", "ghostftp web/", "runs-on: macos"):
            if marker in lowered:
                fail(f"{workflow_rel} references retired application marker: {marker}")

    release_workflow = read(".github/workflows/release.yml")
    if "android/" in release_workflow.lower():
        fail("published Windows/Linux release workflow must not retroactively include Android development artifacts")
    if re.search(r"(?m)^\s*default:\s*['\"]?\d+\.\d+\.\d+", release_workflow):
        fail("release workflow contains a hard-coded production version")
    require(
        release_workflow,
        (
            "manual='${{ inputs.version }}'",
            "source_version=\"$(tr -d '\\r\\n' < VERSION)\"",
            "test \"$version\" != '0.0.0'",
            "RELEASE_TAG=ghostftp-v$version",
            "release_channel='current'",
            "release_title=\"Ghost FTP $version\"",
            "packages: write",
            "Publish verified bundle to GitHub Packages",
            "test \"$remote_prerelease\" = 'false'",
            "state=unsigned",
            "state=signed",
            "LINUX_DEBIAN_DEB=amd64,arm64,i386",
            "LINUX_UBUNTU_DEB=amd64,arm64,i386",
            "LINUX_FEDORA_RPM=x86_64,aarch64,i686",
            "LINUX_PORTABLE=amd64,arm64,i386",
            "PUBLIC_PLATFORM_ARTIFACTS=14",
            "PUBLIC_RELEASE_FILES=17",
        ),
        ".github/workflows/release.yml",
    )
    if "--prerelease" in release_workflow:
        fail("current 0.0.x release workflow must not mark the GitHub Release as prerelease")

    retention = read(".github/workflows/release-retention.yml")
    require(
        retention,
        (
            "Publish Ghost FTP",
            "test \"$release_prerelease\" = 'false'",
            "test \"$asset_count\" -eq 17",
            "gh release delete",
            "--cleanup-tag",
            "packages/container/ghost-ftp/versions",
            "Keeping current package version",
            "GHOSTFTP_RELEASE_RETENTION=PASS",
            "GHOSTFTP_PACKAGE_RETENTION=PASS (current=$version)",
            "LATEST_ONLY_RELEASE_RETENTION=YES",
        ),
        ".github/workflows/release-retention.yml",
    )

    require(
        read("scripts/audit_release.py"),
        (
            "MINIMUM_PUBLIC_VERSION=0.0.1",
            "PUBLIC_RELEASE_CHANNEL=CURRENT",
            "CURRENT_RELEASE_PRERELEASE_FLAG=FALSE",
            "LATEST_ONLY_RELEASE_RETENTION=YES",
            "PUBLIC_PLATFORM_ARTIFACTS=14",
            "PUBLIC_RELEASE_FILES=17",
            "LINUX_DEBIAN_DEB=amd64,arm64,i386",
            "LINUX_UBUNTU_DEB=amd64,arm64,i386",
            "LINUX_FEDORA_RPM=x86_64,aarch64,i686",
            "LINUX_PORTABLE=amd64,arm64,i386",
            "GHCR_CURRENT_BUNDLE=REQUIRED",
            "CURRENT_WINDOWS_RELEASE_REQUIRES_TRUSTED_AUTHENTICODE=NO",
            "TRUSTED_AUTHENTICODE_WHEN_CONFIGURED=VERIFIED",
        ),
        "scripts/audit_release.py",
    )

    bug_template = read(".github/ISSUE_TEMPLATE/bug_report.yml")
    if re.search(r"(?m)^\s*placeholder:\s*['\"]\d+\.\d+\.\d+['\"]", bug_template):
        fail("bug template hard-codes the current version")

    localization_audit = read("scripts/audit_localization.py")
    if 'version = read("VERSION").strip()' not in localization_audit:
        fail("localization audit does not read VERSION dynamically")

    print(f"VERSION_AUDIT=PASS ({version}; channel=current)")
    print(f"GO_TOOLCHAIN={GO_TOOLCHAIN}")
    print("PUBLIC_BRAND=Ghost FTP")
    print("RELEASE_TAG_NAMESPACE=ghostftp-vX.Y.Z")
    print("PUBLIC_RELEASE_PLATFORMS=WINDOWS,LINUX")
    print("ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID")
    print("ANDROID_VERSION_BOUND_TO_ROOT_VERSION=YES")
    print("PUBLIC_RELEASE_CHANNEL=CURRENT")
    print("CURRENT_RELEASE_PRERELEASE_FLAG=FALSE")
    print("MINIMUM_PUBLIC_VERSION=0.0.1")
    print("LATEST_ONLY_RELEASE_RETENTION=YES")
    print("ACTIVE_VERSIONING_DOC_BOUND_TO_VERSION=YES")
    print("CURRENT_WINDOWS_RELEASE_REQUIRES_TRUSTED_AUTHENTICODE=NO")
    print("TRUSTED_AUTHENTICODE_WHEN_CONFIGURED=VERIFIED")
    print("SELF_SIGNED_PRODUCTION_IDENTITY=BLOCKED")
    print("CURRENT_GITHUB_PACKAGE=GHCR_RELEASE_BUNDLE")
    return 0


if __name__ == "__main__":
    sys.exit(main())
