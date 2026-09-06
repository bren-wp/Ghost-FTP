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
    major, minor, patch = (int(part) for part in version.split("."))
    if (major, minor, patch) == (0, 0, 0):
        fail("0.0.0 is reserved")

    if f"go {GO_TOOLCHAIN}" not in read("go.mod"):
        fail(f"go.mod must use Go {GO_TOOLCHAIN}")

    for rel in ("cmd/ghostftp/main.go", "cmd/installer/main.go"):
        text = read(rel)
        if 'var version = "dev"' not in text:
            fail(f"{rel} must retain the development version fallback")
        if re.search(r'var\s+version\s*=\s*"\d+\.\d+\.\d+"', text):
            fail(f"{rel} hard-codes a production version")

    brand_version = read("internal/brand/version.go")
    require(brand_version, ('strings.HasPrefix(version, "0.")', 'return version + " Beta"'), "internal/brand/version.go")

    readme = read("README.md")
    if f"Current Ghost FTP version: **{version}**" not in readme:
        fail("README does not expose canonical VERSION")
    if f"## {version}" not in read("CHANGELOG.md"):
        fail("CHANGELOG does not contain a section for VERSION")
    if major == 0:
        if "Development status: **Beta**" not in readme:
            fail("pre-1.0 VERSION must be documented as Beta")
    else:
        if "Development status: **Stable**" not in readme:
            fail("1.x+ VERSION must be documented as Stable")
        if version == "1.0.0":
            lowered_readme = readme.lower()
            if "first stable release" not in lowered_readme and "first maintained release published as a normal stable github release" not in lowered_readme:
                fail("1.0.0 README must explicitly identify the first stable release")

    versioning = read("docs/VERSIONING.md")
    require(versioning, ("0.1.0", "0.x.y", "1.0.0", "Beta", "Stable", "Portable", "Setup"), "docs/VERSIONING.md")

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
            "RELEASE_CHANNEL=stable",
            "GITHUB_RELEASE_PRERELEASE=false",
            "packages: write",
            "PACKAGE_IMAGE=ghcr.io/${owner}/ghost-ftp",
            "PUBLIC_PLATFORM_ARTIFACTS=9",
            "PUBLIC_RELEASE_FILES=12",
            "RELEASE_ASSET_READBACK=PASS",
            "PACKAGE_READBACK=PASS",
        ),
        ".github/workflows/release.yml",
    )
    if "--prerelease" in release_workflow:
        fail("maintained stable release workflow must not publish prereleases")
    if "if: env.RELEASE_CHANNEL == 'stable'" in release_workflow:
        fail("stable-only package publication must not retain a split prerelease/stable channel gate")

    require(
        read("scripts/audit_platform_contract.py"),
        ("ACTIVE_APPLICATION_PLATFORMS=WINDOWS,LINUX", "RETIRED_APPLICATION_SURFACES=WEB,PWA"),
        "scripts/audit_platform_contract.py",
    )
    require(
        read("scripts/audit_release.py"),
        ("PUBLIC_PLATFORM_ARTIFACTS=9", "PUBLIC_RELEASE_FILES=12", "STABLE_GHCR_BUNDLE=REQUIRED", "PRERELEASE_PUBLICATION=BLOCKED"),
        "scripts/audit_release.py",
    )

    bug_template = read(".github/ISSUE_TEMPLATE/bug_report.yml")
    if re.search(r"(?m)^\s*placeholder:\s*['\"]\d+\.\d+\.\d+['\"]", bug_template):
        fail("bug template hard-codes the current version")

    localization_audit = read("scripts/audit_localization.py")
    if 'version = read("VERSION").strip()' not in localization_audit:
        fail("localization audit does not read VERSION dynamically")

    channel = "beta-history" if major == 0 else "stable"
    print(f"VERSION_AUDIT=PASS ({version}; channel={channel})")
    print(f"GO_TOOLCHAIN={GO_TOOLCHAIN}")
    print("PUBLIC_BRAND=Ghost FTP")
    print("RELEASE_TAG_NAMESPACE=ghostftp-vX.Y.Z")
    print("ACTIVE_APPLICATION_PLATFORMS=WINDOWS,LINUX")
    print("RETIRED_APPLICATION_SURFACES=ANDROID,IOS,MACOS,WEB,PWA")
    print("PRE_1_0_RELEASES=HISTORICAL_BETA_ONLY")
    print("FIRST_STABLE_VERSION=1.0.0")
    print("STABLE_RELEASE_PRERELEASE_FLAG=FALSE")
    print("MAINTAINED_PRERELEASE_PUBLICATION=BLOCKED")
    print("STABLE_GITHUB_PACKAGE=GHCR_RELEASE_BUNDLE")
    return 0


if __name__ == "__main__":
    sys.exit(main())
