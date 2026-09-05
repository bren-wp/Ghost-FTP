#!/usr/bin/env python3
"""Verify VERSION is the canonical release version and follows the beta/stable policy."""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
VERSION_RE = re.compile(r"^\d+\.\d+\.\d+$")
GO_TOOLCHAIN = "1.27.1"


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
    if major == 0 and minor == 0 and patch == 0:
        fail("0.0.0 is reserved and cannot be used as a distributable build")

    gomod = read("go.mod")
    if f"go {GO_TOOLCHAIN}" not in gomod:
        fail(f"go.mod must use Go {GO_TOOLCHAIN}")

    for rel in ("cmd/ghostftp/main.go", "cmd/installer/main.go"):
        text = read(rel)
        if 'var version = "dev"' not in text:
            fail(f"{rel} does not keep the safe development version fallback")
        if re.search(r'var\s+version\s*=\s*"\d+\.\d+\.\d+"', text):
            fail(f"{rel} hard-codes a production version")

    brand_version = read("internal/brand/version.go")
    require(
        brand_version,
        (
            'strings.HasPrefix(version, "0.")',
            'return version + " Beta"',
        ),
        "internal/brand/version.go",
    )

    readme = read("README.md")
    if f"Current Ghost FTP version: **{version}**" not in readme:
        fail("README does not expose canonical Ghost FTP VERSION")
    if f"## {version}" not in read("CHANGELOG.md"):
        fail("CHANGELOG does not contain a section for VERSION")

    versioning = read("docs/VERSIONING.md")
    require(
        versioning,
        (
            "0.1.0",
            "0.x.y",
            "1.0.0",
            "Beta",
            "Portable",
            "Setup",
        ),
        "docs/VERSIONING.md",
    )
    if major == 0 and "Development status: **Beta**" not in readme:
        fail("pre-1.0 VERSION must be visibly documented as Beta")

    windows_build = read("BUILD-WINDOWS.ps1")
    require(windows_build, ("Get-Content -LiteralPath $versionFile", "-X main.version=$version"), "BUILD-WINDOWS.ps1")
    linux_build = read("linux/BUILD.sh")
    require(linux_build, ("< VERSION", "-X main.version=${VERSION}"), "linux/BUILD.sh")
    local_build = read("scripts/BUILD-LOCAL.sh")
    require(local_build, ("< VERSION", "-X main.version=$VERSION"), "scripts/BUILD-LOCAL.sh")

    linux_control = read("linux/debian/control.in")
    if "@VERSION@" not in linux_control or re.search(r"(?m)^Version:\s*\d+\.\d+\.\d+", linux_control):
        fail("Linux DEB metadata is not bound to canonical VERSION")

    for retired in ("android", "ios", "macos"):
        if (ROOT / retired).exists():
            fail(f"retired application platform directory must be removed: {retired}/")

    for workflow_rel in (".github/workflows/ci.yml", ".github/workflows/release.yml"):
        workflow = read(workflow_rel)
        if f"go-version: '{GO_TOOLCHAIN}'" not in workflow:
            fail(f"{workflow_rel} does not pin Go {GO_TOOLCHAIN}")
        require(workflow, ("windows:", "linux:", "bash linux/BUILD.sh"), workflow_rel)
        for marker in ("android/", "ios/", "macos/", "runs-on: macos"):
            if marker in workflow.lower():
                fail(f"{workflow_rel} references retired platform marker: {marker}")

    release_workflow = read(".github/workflows/release.yml")
    if re.search(r"(?m)^\s*default:\s*['\"]?\d+\.\d+\.\d+", release_workflow):
        fail("release workflow contains a hard-coded default production version")
    require(
        release_workflow,
        (
            "manual='${{ inputs.version }}'",
            "source_version=\"$(tr -d '\\r\\n' < VERSION)\"",
            "RELEASE_TAG=ghostftp-v$version",
            "RELEASE_CHANNEL",
            "--prerelease",
            "PUBLIC_PLATFORM_ARTIFACTS=9",
            "PUBLIC_RELEASE_FILES=12",
        ),
        ".github/workflows/release.yml",
    )

    require(read("scripts/audit_platform_contract.py"), ("ACTIVE_APPLICATION_PLATFORMS=WINDOWS,LINUX",), "scripts/audit_platform_contract.py")
    require(read("scripts/audit_release.py"), ("PUBLIC_PLATFORM_ARTIFACTS=9", "PUBLIC_RELEASE_FILES=12"), "scripts/audit_release.py")

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
    print("FIRST_STABLE_VERSION=1.0.0")
    print("WEB_COMPANION_VERSIONING=INDEPENDENT_FROM_DESKTOP_RELEASE")
    return 0


if __name__ == "__main__":
    sys.exit(main())
