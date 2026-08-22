#!/usr/bin/env python3
"""Verify VERSION is the single production version source for ByFTP."""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
VERSION_RE = re.compile(r"^\d+\.\d+\.\d+$")


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

    changelog = read("CHANGELOG.md")
    if f"## {version}" not in changelog:
        fail("CHANGELOG does not contain a section for VERSION")

    # Long-lived documentation must not advertise a stale current release.
    for path in sorted((ROOT / "docs").rglob("*.md")):
        text = path.read_text(encoding="utf-8")
        rel = path.relative_to(ROOT)
        for pattern in (
            r"Current release:\s*(\d+\.\d+\.\d+)",
            r"Trenutačno izdanje:\s*(\d+\.\d+\.\d+)",
        ):
            for match in re.finditer(pattern, text):
                if match.group(1) != version:
                    fail(f"{rel} advertises stale current release {match.group(1)}")

    windows_build = read("BUILD-WINDOWS.ps1")
    if "Get-Content -LiteralPath $versionFile" not in windows_build or "-X main.version=$version" not in windows_build:
        fail("Windows build does not inject VERSION into the runtime")

    for rel in ("scripts/BUILD-LOCAL.sh", "scripts/BUILD-LINUX.sh", "scripts/BUILD-MACOS.sh"):
        text = read(rel)
        if "< VERSION" not in text:
            fail(f"{rel} does not read canonical VERSION")
    if "-X main.version=$VERSION" not in read("scripts/BUILD-LOCAL.sh"):
        fail("local build does not inject VERSION")
    if "-X main.version=${VERSION}" not in read("scripts/BUILD-LINUX.sh"):
        fail("Linux build does not inject VERSION")
    if "-X main.version=${VERSION}" not in read("scripts/BUILD-MACOS.sh"):
        fail("macOS build does not inject VERSION")

    release_workflow = read(".github/workflows/release.yml")
    if re.search(r"(?m)^\s*default:\s*['\"]?\d+\.\d+\.\d+", release_workflow):
        fail("release workflow contains a hard-coded default production version")
    for marker in ("$manual", "Get-Content -LiteralPath 'VERSION' -Raw"):
        if marker not in release_workflow:
            fail(f"release workflow lacks VERSION fallback marker: {marker}")

    package_markers = (
        "<PackageId>ByFTP.Windows</PackageId>",
        "<Version>$env:VERSION</Version>",
        "dotnet nuget push",
        "nuget.pkg.github.com/bren-wp/index.json",
        "--skip-duplicate",
    )
    for marker in package_markers:
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
    print("PLATFORM_VERSION_SOURCES=WINDOWS,LINUX,MACOS")
    print("GITHUB_PACKAGE_VERSION_SOURCE=VERSION")
    print("PRODUCTION_DOC_VERSION_DRIFT=BLOCKED")
    return 0


if __name__ == "__main__":
    sys.exit(main())
