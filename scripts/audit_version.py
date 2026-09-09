#!/usr/bin/env python3
"""Verify canonical Ghost FTP versioning across Windows/Linux release surfaces."""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
VERSION_RE = re.compile(r"^\d+\.\d+\.\d+$")
GO_TOOLCHAIN = "1.27.1"
RETIRED_ROOTS = ("android", "ios", "macos", "GhostFTP WEB")
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
OLD_PUBLIC_VERSION_RE = re.compile(r"(?<![\d.])1\.\d+\.\d+(?![\d.])")


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
    major = parts[0]
    channel_label = "Beta" if major == 0 else "Stable"

    if f"go {GO_TOOLCHAIN}" not in read("go.mod"):
        fail(f"go.mod must use Go {GO_TOOLCHAIN}")

    for rel in ("cmd/ghostftp/main.go", "cmd/installer/main.go"):
        text = read(rel)
        if 'var version = "dev"' not in text:
            fail(f"{rel} must retain the development version fallback")
        if re.search(r'var\s+version\s*=\s*"\d+\.\d+\.\d+"', text):
            fail(f"{rel} hard-codes a production version")

    brand_version = read("internal/brand/version.go")
    require(
        brand_version,
        ('strings.HasPrefix(version, "0.")', 'return version + " Beta"'),
        "internal/brand/version.go",
    )

    readme = read("README.md")
    require(
        readme,
        (
            f"Current Ghost FTP version: **{version}**",
            f"Development status: **{channel_label}**",
            f"## {version} {channel_label}",
        ),
        "README.md",
    )
    if f"## {version}" not in read("CHANGELOG.md"):
        fail("CHANGELOG does not contain a section for VERSION")

    versioning = read("docs/VERSIONING.md")
    require(
        versioning,
        (
            f"Current source candidate: **{version} {channel_label}**",
            f"VERSION={version}",
            f"TAG=ghostftp-v{version}",
            f"## {version} release checklist",
            "0.0.0",
            "0.0.1",
            "0.0.2",
            "Beta",
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
        match = OLD_PUBLIC_VERSION_RE.search(text)
        if match:
            fail(f"active current-line documentation still contains retired 1.x version {match.group(0)}: {rel}")
        if "ghostftp-v1." in text:
            fail(f"active current-line documentation still contains retired 1.x tag: {rel}")

    windows_build = read("BUILD-WINDOWS.ps1")
    require(windows_build, ("Get-Content -LiteralPath $versionFile", "-X main.version=$version"), "BUILD-WINDOWS.ps1")
    linux_build = read("linux/BUILD.sh")
    require(linux_build, ("< VERSION", "-X main.version=${VERSION}"), "linux/BUILD.sh")
    local_build = read("scripts/BUILD-LOCAL.sh")
    require(local_build, ("< VERSION", "-X main.version=$VERSION"), "scripts/BUILD-LOCAL.sh")

    linux_control = read("linux/debian/control.in")
    if "@VERSION@" not in linux_control or re.search(r"(?m)^Version:\s*\d+\.\d+\.\d+", linux_control):
        fail("Linux DEB metadata is not bound to VERSION")

    for retired in RETIRED_ROOTS:
        if (ROOT / retired).exists():
            fail(f"retired application surface must be removed: {retired}/")

    for workflow_rel in (".github/workflows/ci.yml", ".github/workflows/release.yml"):
        workflow = read(workflow_rel)
        if f"go-version: '{GO_TOOLCHAIN}'" not in workflow:
            fail(f"{workflow_rel} does not pin Go {GO_TOOLCHAIN}")
        require(workflow, ("windows:", "linux:", "bash linux/BUILD.sh"), workflow_rel)
        lowered = workflow.lower()
        for marker in ("android/", "ios/", "macos/", "ghostftp web/", "runs-on: macos"):
            if marker in lowered:
                fail(f"{workflow_rel} references retired application marker: {marker}")

    release_workflow = read(".github/workflows/release.yml")
    if re.search(r"(?m)^\s*default:\s*['\"]?\d+\.\d+\.\d+", release_workflow):
        fail("release workflow contains a hard-coded production version")
    require(
        release_workflow,
        (
            "manual='${{ inputs.version }}'",
            "source_version=\"$(tr -d '\\r\\n' < VERSION)\"",
            "RELEASE_TAG=ghostftp-v$version",
            "RELEASE_CHANNEL",
            "--prerelease",
            "packages: write",
            "if: env.RELEASE_CHANNEL == 'stable'",
            "state=unsigned",
            "state=signed",
            "LINUX_PORTABLE=amd64,arm64,i386",
            "PUBLIC_PLATFORM_ARTIFACTS=12",
            "PUBLIC_RELEASE_FILES=15",
        ),
        ".github/workflows/release.yml",
    )

    retention = read(".github/workflows/release-retention.yml")
    require(
        retention,
        (
            "Publish Ghost FTP",
            "gh release delete",
            "--cleanup-tag",
            "packages/container/ghost-ftp/versions",
            "GHOSTFTP_RELEASE_RETENTION=PASS",
            "GHOSTFTP_PACKAGE_RETENTION=PASS",
            "LATEST_ONLY_RELEASE_RETENTION=YES",
        ),
        ".github/workflows/release-retention.yml",
    )

    require(
        read("scripts/audit_release.py"),
        (
            "MINIMUM_PUBLIC_VERSION=0.0.1",
            "LATEST_ONLY_RELEASE_RETENTION=YES",
            "PUBLIC_PLATFORM_ARTIFACTS=12",
            "PUBLIC_RELEASE_FILES=15",
            "LINUX_PORTABLE=amd64,arm64,i386",
            "STABLE_GHCR_BUNDLE=REQUIRED",
            "STABLE_WINDOWS_RELEASE_REQUIRES_TRUSTED_AUTHENTICODE=NO",
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

    channel = "beta" if major == 0 else "stable"
    print(f"VERSION_AUDIT=PASS ({version}; channel={channel})")
    print(f"GO_TOOLCHAIN={GO_TOOLCHAIN}")
    print("PUBLIC_BRAND=Ghost FTP")
    print("RELEASE_TAG_NAMESPACE=ghostftp-vX.Y.Z")
    print("ACTIVE_APPLICATION_PLATFORMS=WINDOWS,LINUX")
    print("PRE_1_0_CHANNEL=BETA")
    print("MINIMUM_PUBLIC_VERSION=0.0.1")
    print("LATEST_ONLY_RELEASE_RETENTION=YES")
    print("ACTIVE_VERSIONING_DOC_BOUND_TO_VERSION=YES")
    print("STABLE_RELEASE_PRERELEASE_FLAG=FALSE")
    print("STABLE_WINDOWS_RELEASE_REQUIRES_TRUSTED_AUTHENTICODE=NO")
    print("TRUSTED_AUTHENTICODE_WHEN_CONFIGURED=VERIFIED")
    print("SELF_SIGNED_PRODUCTION_IDENTITY=BLOCKED")
    print("STABLE_GITHUB_PACKAGE=GHCR_RELEASE_BUNDLE")
    return 0


if __name__ == "__main__":
    sys.exit(main())
