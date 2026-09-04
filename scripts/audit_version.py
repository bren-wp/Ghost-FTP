#!/usr/bin/env python3
"""Verify VERSION is the single production version source for Ghost FTP."""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
VERSION_RE = re.compile(r"^\d+\.\d+\.\d+$")
GO_TOOLCHAIN = "1.27.1"
GRADLE_TOOLCHAIN = "9.7.1"
AGP_TOOLCHAIN = "9.4.0"


def fail(message: str) -> None:
    raise SystemExit("VERSION_AUDIT_FAILED: " + message)


def read(path: str) -> str:
    p = ROOT / path
    if not p.is_file():
        fail(f"missing {path}")
    return p.read_text(encoding="utf-8")


def require(text: str, markers: tuple[str, ...], where: str) -> None:
    for marker in markers:
        if marker not in text:
            fail(f"{where} is missing version binding: {marker}")


def main() -> int:
    version = read("VERSION").strip()
    if not VERSION_RE.fullmatch(version):
        fail(f"VERSION is not semantic: {version!r}")

    web_version = read("GhostFTP WEB/VERSION").strip()
    if web_version != version:
        fail(f"web VERSION must equal root VERSION: {web_version!r} != {version!r}")

    try:
        web_composer = json.loads(read("GhostFTP WEB/composer.json"))
    except json.JSONDecodeError as exc:
        fail(f"web composer.json is invalid JSON: {exc}")
    if web_composer.get("version") != version:
        fail("web Composer version is not bound to canonical VERSION")
    if web_composer.get("name") != "brendigo/ghost-ftp-web":
        fail("web Composer package name is not Ghost FTP")
    description = str(web_composer.get("description", ""))
    if "Ghost FTP" not in description or "GhostFTP" in description:
        fail("web Composer description contains stale public branding")

    service_worker = read("GhostFTP WEB/service-worker.js")
    if f"v{version}" not in service_worker:
        fail("web service-worker cache version is not bound to canonical VERSION")

    gomod = read("go.mod")
    if f"go {GO_TOOLCHAIN}" not in gomod:
        fail(f"go.mod must use Go {GO_TOOLCHAIN}")

    android_root = read("android/build.gradle.kts")
    if f'id("com.android.application") version "{AGP_TOOLCHAIN}" apply false' not in android_root:
        fail(f"android/build.gradle.kts must use Android Gradle Plugin {AGP_TOOLCHAIN}")

    for rel in ("cmd/ghostftp/main.go", "cmd/installer/main.go"):
        text = read(rel)
        if 'var version = "dev"' not in text:
            fail(f"{rel} does not keep the safe development version fallback")
        if re.search(r'var\s+version\s*=\s*"\d+\.\d+\.\d+"', text):
            fail(f"{rel} hard-codes a production version")

    readme = read("README.md")
    if f"Current Ghost FTP version: **{version}**" not in readme:
        fail("README does not expose canonical Ghost FTP VERSION")
    if f"## {version}" not in read("CHANGELOG.md"):
        fail("CHANGELOG does not contain a section for VERSION")

    repository_audit = read("scripts/audit_repository.py")
    require(
        repository_audit,
        (
            '"git", "ls-files", "-s", "-z"',
            "case-insensitive path collision",
            "generated/cache path is tracked",
            "MERGE_CONFLICT_RE",
        ),
        "scripts/audit_repository.py",
    )

    web_audit = read("scripts/audit_web.py")
    require(
        web_audit,
        (
            "WEB_AUDIT_FAILED",
            "run_runtime_checks",
            "PHP CLI is required",
            "node",
            "--check",
        ),
        "scripts/audit_web.py",
    )

    release_audit = read("scripts/audit_release.py")
    require(
        release_audit,
        (
            'run("scripts/audit_repository.py")',
            'run("scripts/audit_web.py")',
            "RELEASE_TAG_NAMESPACE=ghostftp-vX.Y.Z",
            "PUBLIC_RELEASE_FILES=13",
        ),
        "scripts/audit_release.py",
    )

    windows_build = read("BUILD-WINDOWS.ps1")
    require(
        windows_build,
        (
            "Get-Content -LiteralPath $versionFile",
            "-X main.version=$version",
        ),
        "BUILD-WINDOWS.ps1",
    )

    for rel in ("scripts/BUILD-LOCAL.sh", "ios/BUILD.sh", "linux/BUILD.sh", "macos/BUILD.sh"):
        if "< VERSION" not in read(rel):
            fail(f"{rel} does not read canonical VERSION")
    if "-X main.version=$VERSION" not in read("scripts/BUILD-LOCAL.sh"):
        fail("local build does not inject VERSION")
    if "-X main.version=${VERSION}" not in read("linux/BUILD.sh"):
        fail("Linux build does not inject VERSION")
    if "-X main.version=${VERSION}" not in read("macos/BUILD.sh"):
        fail("macOS build does not inject VERSION")

    for legacy in ("scripts/BUILD-LINUX.sh", "scripts/BUILD-MACOS.sh", "scripts/BUILD-IOS.sh"):
        if (ROOT / legacy).exists():
            fail(f"legacy platform build wrapper must be removed: {legacy}")

    linux_control = read("linux/debian/control.in")
    if "@VERSION@" not in linux_control or re.search(r"(?m)^Version:\s*\d+\.\d+\.\d+", linux_control):
        fail("Linux DEB metadata is not bound to canonical VERSION")

    macos_plist = read("macos/Info.plist.in")
    if "@VERSION@" not in macos_plist or re.search(r"<string>\d+\.\d+\.\d+</string>", macos_plist):
        fail("macOS Info.plist template hard-codes a production version")

    android = read("android/app/build.gradle.kts")
    require(
        android,
        ('rootProject.file("../VERSION")', "versionName = canonicalVersion", "versionCode = canonicalVersionCode"),
        "android/app/build.gradle.kts",
    )
    if re.search(r'versionName\s*=\s*"\d+\.\d+\.\d+', android):
        fail("Android build hard-codes a production version")

    ios_build = read("ios/BUILD.sh")
    require(
        ios_build,
        ('MARKETING_VERSION="$VERSION"', 'CURRENT_PROJECT_VERSION="$BUILD_NUMBER"', "scripts/package_ios.py"),
        "ios/BUILD.sh",
    )
    ios_project = read("ios/GhostFTP.xcodeproj/project.pbxproj")
    if "MARKETING_VERSION = 0.0.0" not in ios_project:
        fail("iOS project does not keep the safe development marketing-version fallback")
    if re.search(r"MARKETING_VERSION = (?!0\.0\.0)\d+\.\d+\.\d+", ios_project):
        fail("iOS project hard-codes a production release version")

    for workflow_rel in (".github/workflows/ci.yml", ".github/workflows/release.yml"):
        workflow = read(workflow_rel)
        if f"go-version: '{GO_TOOLCHAIN}'" not in workflow:
            fail(f"{workflow_rel} does not pin Go {GO_TOOLCHAIN}")
        if f"gradle-version: '{GRADLE_TOOLCHAIN}'" not in workflow:
            fail(f"{workflow_rel} does not pin Gradle {GRADLE_TOOLCHAIN}")
        for marker in ("bash linux/BUILD.sh", "bash macos/BUILD.sh", "bash ios/BUILD.sh"):
            if marker not in workflow:
                fail(f"{workflow_rel} is not using canonical platform build entry point: {marker}")

    release_workflow = read(".github/workflows/release.yml")
    if re.search(r"(?m)^\s*default:\s*['\"]?\d+\.\d+\.\d+", release_workflow):
        fail("release workflow contains a hard-coded default production version")
    require(
        release_workflow,
        (
            "manual='${{ inputs.version }}'",
            "source_version=\"$(tr -d '\\r\\n' < VERSION)\"",
            "RELEASE_TAG=ghostftp-v$version",
            "PUBLIC_PLATFORM_ARTIFACTS=10",
            "PUBLIC_RELEASE_FILES=13",
        ),
        ".github/workflows/release.yml",
    )
    if "GhostFTP.Windows" in release_workflow:
        fail("release workflow contains the retired NuGet package identity")
    if "dotnet nuget push" not in release_workflow or "packages: write" not in release_workflow:
        fail("release workflow does not publish the GhostFTP GitHub Package")

    bug_template = read(".github/ISSUE_TEMPLATE/bug_report.yml")
    if re.search(r"(?m)^\s*placeholder:\s*['\"]\d+\.\d+\.\d+['\"]", bug_template):
        fail("bug template hard-codes the current version")

    localization_audit = read("scripts/audit_localization.py")
    if 'version = read("VERSION").strip()' not in localization_audit:
        fail("localization audit does not read VERSION dynamically")
    if "Current Ghost FTP version: **{version}**" not in localization_audit:
        fail("localization audit is not bound to the Ghost FTP README version marker")

    print(f"VERSION_AUDIT=PASS ({version})")
    print(f"GO_TOOLCHAIN={GO_TOOLCHAIN}")
    print(f"GRADLE_TOOLCHAIN={GRADLE_TOOLCHAIN}")
    print(f"ANDROID_GRADLE_PLUGIN={AGP_TOOLCHAIN}")
    print("PUBLIC_BRAND=Ghost FTP")
    print("RELEASE_TAG_NAMESPACE=ghostftp-vX.Y.Z")
    print("PLATFORM_VERSION_SOURCES=WINDOWS,LINUX,MACOS,ANDROID,IOS,WEB")
    print("WEB_VERSION_BOUND_TO_ROOT_VERSION=YES")
    print("PRODUCTION_DOC_VERSION_DRIFT=BLOCKED")
    return 0


if __name__ == "__main__":
    sys.exit(main())
