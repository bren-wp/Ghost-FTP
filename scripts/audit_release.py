#!/usr/bin/env python3
"""Verify ByFTP's cross-platform release contract and fail-closed publisher."""

from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def fail(message: str) -> None:
    raise SystemExit("RELEASE_AUDIT_FAILED: " + message)


def read(path: str) -> str:
    target = ROOT / path
    if not target.is_file():
        fail(f"missing {path}")
    return target.read_text(encoding="utf-8")


def require(path: str, markers: tuple[str, ...]) -> str:
    text = read(path)
    for marker in markers:
        if marker not in text:
            fail(f"{path} is missing required release invariant: {marker}")
    return text


def main() -> int:
    workflow = require(
        ".github/workflows/release.yml",
        (
            "name: Publish ByFTP",
            "group: byftp-release",
            "cancel-in-progress: false",
            "quality:",
            "Production quality, race, security and privacy",
            "needs: [quality, windows, linux, macos]",
            "go test -race ./...",
            "go telemetry off",
            "Windows x64 and x86",
            "Linux amd64 arm64 and i386",
            "macOS Universal",
            "scripts/BUILD-LINUX.sh",
            "scripts/BUILD-MACOS.sh",
            "publish_release.ps1",
            "scripts/verify_bundle.py",
            "--arch $arch",
            "RELEASE_QUALITY_GATE=passed",
            "ByFTP-$env:VERSION-Setup-x86.exe",
            "ByFTP-$env:VERSION-Portable-x86.exe",
            "ByFTP-$env:VERSION-Linux-amd64.deb",
            "ByFTP-$env:VERSION-Linux-arm64.deb",
            "ByFTP-$env:VERSION-Linux-i386.deb",
            "ByFTP-$env:VERSION-macOS-Universal.pkg",
            "$expected = @(",
            "$assets = @(",
            "Compare-Object -ReferenceObject $wanted -DifferenceObject $actual",
            "$expectedName = \"ByFTP.Windows.$env:VERSION.nupkg\"",
            "$packages.Count -ne 1",
            "$packages[0].Name -ne $expectedName",
        ),
    )
    if re.search(r"(?m)^\s*default:\s*['\"]?\d+\.\d+\.\d+", workflow):
        fail("release workflow must not hard-code a default production version")
    if re.search(r"(?m)^\s*tags:\s*$", workflow):
        fail("release workflow must not retrigger itself from its own tag")
    if "v2.15.0" in workflow or "release delete-asset" in workflow:
        fail("release workflow contains obsolete one-off migration cleanup")
    if "gh release create" in workflow or "gh release upload" in workflow:
        fail("release.yml must not bypass the centralized publish_release.ps1 publisher")

    asset_match = re.search(r"\$assets\s*=\s*@\((.*?)\)\s*\n\s*foreach", workflow, re.S)
    if not asset_match:
        fail("explicit public $assets allowlist was not found")
    public_assets = asset_match.group(1)
    expected_asset_markers = (
        "Portable-x64.exe", "Setup-x64.exe", "Windows-x64.zip",
        "Portable-x86.exe", "Setup-x86.exe", "Windows-x86.zip",
        "Linux-amd64.deb", "Linux-arm64.deb", "Linux-i386.deb",
        "macOS-Universal.pkg", "SHA256.txt", "RELEASE-NOTES.txt", "BUILD-METADATA.txt",
    )
    for marker in expected_asset_markers:
        if marker not in public_assets:
            fail(f"public release asset allowlist is missing: {marker}")
    for forbidden in ("Source.zip", "Uninstall-", "verification.txt"):
        if forbidden in public_assets:
            fail(f"public release asset allowlist contains internal/obsolete artifact: {forbidden}")

    publisher = require(
        "scripts/publish_release.ps1",
        (
            "Resolve-TagCommit", "Assert-TagCommit", "Assert-RemoteAsset",
            "Get-FileHash", "sha256:", "gh release upload",
            "RELEASE_PUBLISH_VERIFICATION=PASS",
            "$finalByName.Count -ne $localAssets.Count",
        ),
    )
    if "--clobber" in publisher:
        fail("publisher must not overwrite an existing release asset automatically")

    verifier = require(
        "scripts/verify_bundle.py",
        (
            "BUNDLE-SHA256.txt", "x64", "x86", "duplicate path",
            "unsafe path", "SHA-256 mismatch", "BUNDLE_VERIFICATION=PASS", "forbidden",
            "MAX_ENTRIES", "MAX_UNCOMPRESSED_BYTES",
        ),
    )
    if "extractall(" in verifier or ".extract(" in verifier:
        fail("ZIP verifier must not extract untrusted paths to disk")

    windows = require(
        "BUILD-WINDOWS.ps1",
        (
            "Build-ByFTPArchitecture -GoArch 'amd64' -Label 'x64'",
            "Build-ByFTPArchitecture -GoArch '386' -Label 'x86'",
            "$telemetryMode = (go telemetry).Trim()",
            "python scripts/verify_release.py",
        ),
    )
    linux = require(
        "scripts/BUILD-LINUX.sh",
        ("build_arch amd64 amd64", "build_arch arm64 arm64", "build_arch 386 i386", "dpkg-deb", 'telemetry="$(go telemetry)"'),
    )
    macos = require(
        "scripts/BUILD-MACOS.sh",
        ('GOARCH="$arch"', "lipo -create", "pkgbuild", "ByFTP.app", 'telemetry="$(go telemetry)"'),
    )
    if not windows or not linux or not macos:
        fail("platform build contract is unavailable")

    notes = require(
        "scripts/release_notes.py",
        ("Setup x86", "Linux amd64", "Linux arm64", "Linux i386", "macOS Universal", "SHA256.txt"),
    )
    if not notes:
        fail("release-note contract is unavailable")

    ci = require(
        ".github/workflows/ci.yml",
        (
            "go telemetry off", "python scripts/audit_localization.py", "python scripts/audit_docs.py",
            "python scripts/audit_security.py", "python scripts/audit_privacy.py", "python scripts/audit_release.py",
            "python -m unittest discover -s scripts -p 'test_*.py'", "go test -race ./...", "go vet ./...",
            "Windows x64 and x86 production build", "Linux DEB amd64 arm64 and i386", "macOS Universal PKG",
        ),
    )
    if "BUILD-WINDOWS.ps1" not in ci:
        fail("CI must use the canonical Windows production build")

    # User-facing docs and generated release notes must describe only current
    # public packages. CHANGELOG may retain historical names for accurate history.
    for rel in ("README.md", "docs/INSTALACIJA.md", "scripts/release_notes.py"):
        text = read(rel)
        for forbidden in ("verification.txt", "Source.zip", "Uninstall-"):
            if forbidden in text:
                fail(f"{rel} advertises obsolete/internal release artifact: {forbidden}")

    print("RELEASE_AUDIT=PASS")
    print("RELEASE_SINGLE_TRIGGER=ENABLED")
    print("RELEASE_SERIALIZATION=ENABLED")
    print("RELEASE_QUALITY_RACE_GATE=ENABLED")
    print("RELEASE_STAGING_ALLOWLIST=ENABLED")
    print("RELEASE_RERUN_REPAIR=ENABLED")
    print("RELEASE_TAG_COMMIT_BINDING=ENABLED")
    print("RELEASE_ASSET_DIGEST_FAIL_CLOSED=ENABLED")
    print("WINDOWS_X64_X86=ENABLED")
    print("LINUX_DEB=AMD64,ARM64,I386")
    print("MACOS_PKG=UNIVERSAL")
    print("PUBLIC_RELEASE_INTERNAL_ARTIFACTS=FORBIDDEN")
    return 0


if __name__ == "__main__":
    main()
