#!/usr/bin/env python3
"""Verify VERSION is the single production version source for ByFTP."""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
VERSION_RE = re.compile(r"^\d+\.\d+\.\d+$")
GO_TOOLCHAIN = "1.27.0"
GRADLE_TOOLCHAIN = "9.7.0"


def fail(message: str) -> None:
    raise SystemExit("VERSION_AUDIT_FAILED: " + message)


def read(path: str) -> str:
    p = ROOT / path
    if not p.is_file():
        fail(f"missing {path}")
    return p.read_text(encoding="utf-8")


def main() -> int:
    version = read("VERSION").strip()
    if not VERSION_RE.fullmatch(version):
        fail(f"VERSION is not semantic: {version!r}")

    gomod = read("go.mod")
    if f"go {GO_TOOLCHAIN}" not in gomod:
        fail(f"go.mod must use Go {GO_TOOLCHAIN}")

    for rel in ("cmd/byftp/main.go", "cmd/installer/main.go"):
        text = read(rel)
        if 'var version = "dev"' not in text:
            fail(f"{rel} does not keep the safe development version fallback")
        stale = re.search(r'var\s+version\s*=\s*"\d+\.\d+\.\d+"', text)
        if stale:
            fail(f"{rel} contains a hard-coded production version: {stale.group(0)}")

    readme = read("README.md")
    if f"Current release: {version}" not in readme:
        fail("README does not expose VERSION as the current release")
    if f"## {version}" not in read("CHANGELOG.md"):
        fail("CHANGELOG does not contain a section for VERSION")

    for path in sorted((ROOT / "docs").rglob("*.md")):
        text = path.read_text(encoding="utf-8")
        rel = path.relative_to(ROOT)
        for pattern in (r"Current release:\s*(\d+\.\d+\.\d+)", r"Trenutačno izdanje:\s*(\d+\.\d+\.\d+)"):
            for match in re.finditer(pattern, text):
                if match.group(1) != version:
                    fail(f"{rel} advertises stale current release {match.group(1)}")

    windows_build = read("BUILD-WINDOWS.ps1")
    if "Get-Content -LiteralPath $versionFile" not in windows_build or "-X main.version=$version" not in windows_build:
        fail("Windows build does not inject VERSION into the runtime")

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
    for marker in ('rootProject.file("../VERSION")', "versionName = canonicalVersion", "versionCode = canonicalVersionCode"):
        if marker not in android:
            fail(f"Android build is not bound to canonical VERSION: missing {marker}")
    if re.search(r'versionName\s*=\s*"\d+\.\d+\.\d+', android):
        fail("Android build hard-codes a production version")

    ios_build = read("ios/BUILD.sh")
    for marker in ('MARKETING_VERSION="$VERSION"', 'CURRENT_PROJECT_VERSION="$BUILD_NUMBER"', "scripts/package_ios.py"):
        if marker not in ios_build:
            fail(f"iOS build is not bound to VERSION: missing {marker}")
    ios_project = read("ios/ByFTP.xcodeproj/project.pbxproj")
    if "MARKETING_VERSION = 0.0.0" not in ios_project:
        fail("iOS project does not keep the safe development marketing-version fallback")
    if re.search(r"MARKETING_VERSION = (?!0\.0\.0)\d+\.\d+\.\d+", ios_project):
        fail("iOS project hard-codes a production release version")
    if "CFBundleShortVersionString" not in read("scripts/package_ios.py"):
        fail("iOS packager does not verify the production version")

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
    for marker in ("$manual", "Get-Content -LiteralPath 'VERSION' -Raw"):
        if marker not in release_workflow:
            fail(f"release workflow lacks VERSION fallback marker: {marker}")

    for marker in (
        "<PackageId>ByFTP.Windows</PackageId>", "<Version>$env:VERSION</Version>", "dotnet nuget push",
        "nuget.pkg.github.com/bren-wp/index.json", "--skip-duplicate",
    ):
        if marker not in release_workflow:
            fail(f"GitHub Package is not bound to VERSION: missing {marker}")

    bug_template = read(".github/ISSUE_TEMPLATE/bug_report.yml")
    if re.search(r"(?m)^\s*placeholder:\s*['\"]\d+\.\d+\.\d+['\"]", bug_template):
        fail("bug template hard-codes the current version")

    localization_audit = read("scripts/audit_localization.py")
    if 'version = read("VERSION").strip()' not in localization_audit:
        fail("localization audit does not read VERSION dynamically")
    if re.search(r"Current release:\s*\d+\.\d+\.\d+", localization_audit):
        fail("localization audit hard-codes a release version")

    print(f"VERSION_AUDIT=PASS ({version})")
    print(f"GO_TOOLCHAIN={GO_TOOLCHAIN}")
    print(f"GRADLE_TOOLCHAIN={GRADLE_TOOLCHAIN}")
    print("PLATFORM_VERSION_SOURCES=WINDOWS,LINUX,MACOS,ANDROID,IOS")
    print("PLATFORM_BUILD_ENTRYPOINTS=ROOT_WINDOWS,LINUX_DIRECTORY,MACOS_DIRECTORY,IOS_DIRECTORY")
    print("GITHUB_PACKAGE_VERSION_SOURCE=VERSION")
    print("PRODUCTION_DOC_VERSION_DRIFT=BLOCKED")
    return 0


if __name__ == "__main__":
    sys.exit(main())
