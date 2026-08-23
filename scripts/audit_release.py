#!/usr/bin/env python3
"""Validate the ByFTP production release workflow and publication contract."""

from __future__ import annotations

import re
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


def require(text: str, marker: str, where: str) -> None:
    if marker not in text:
        fail(f"{where} is missing required marker: {marker}")


def main() -> int:
    version = read("VERSION").strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        fail(f"invalid VERSION: {version!r}")

    workflow = read(".github/workflows/release.yml")
    for marker in (
        "group: byftp-release", "quality:", "windows:", "linux:", "macos:", "android:", "publish:",
        "needs: [quality, windows, linux, macos, android]",
        "go telemetry off", "go test ./...", "go test -race ./...", "go vet ./...",
        "python scripts/audit_localization.py", "python scripts/audit_version.py", "python scripts/audit_android.py",
        "python scripts/audit_docs.py", "python scripts/audit_security.py", "python scripts/audit_privacy.py", "python scripts/audit_release.py",
        ".\\BUILD-WINDOWS.ps1", "bash scripts/BUILD-LINUX.sh", "bash scripts/BUILD-MACOS.sh",
        "gradle-version: '9.5.0'", "android-37*", "build-tools/36.0.0", ":app:testDebugUnitTest", ":app:lintDebug", ":app:assembleDebug",
        "python scripts/verify_bundle.py $zip --version $version --arch $arch", "scripts\\publish_release.ps1",
        "Verify public release staging", "ByFTP-$env:VERSION-Windows-x64.zip", "ByFTP-$env:VERSION-Windows-x86.zip",
        "ByFTP-$env:VERSION-Linux-amd64.deb", "ByFTP-$env:VERSION-Linux-arm64.deb", "ByFTP-$env:VERSION-Linux-i386.deb",
        "ByFTP-$env:VERSION-macOS-Universal.pkg", "<PackageId>ByFTP.Windows</PackageId>", "dotnet nuget push", "--skip-duplicate",
    ):
        require(workflow, marker, ".github/workflows/release.yml")

    publisher = read("scripts/publish_release.ps1")
    for marker in (
        "function Invoke-GhJson", "function Try-GhJson", "@('api',", "gh release create", "gh release edit",
        "gh release upload", "Get-FileHash", "SHA256", "Assert-TagCommit", "Assert-RemoteAsset", "RELEASE_PUBLISH_VERIFICATION=PASS",
    ):
        require(publisher, marker, "scripts/publish_release.ps1")

    for rel in ("BUILD-WINDOWS.ps1", "scripts/BUILD-LINUX.sh", "scripts/BUILD-MACOS.sh"):
        require(read(rel), "VERSION", rel)

    verifier = read("scripts/verify_bundle.py")
    for marker in ("BUNDLE_VERIFICATION_FAILED", "BUNDLE-SHA256.txt", "Documentation/SECURITY.md"):
        require(verifier, marker, "scripts/verify_bundle.py")

    for rel in (
        "README.md", "CHANGELOG.md", "android/README.md", "docs/INSTALLATION.md", "docs/RELEASE-VERIFICATION.md",
        "docs/SECURITY.md", "docs/PRIVACY.md",
    ):
        read(rel)

    for rel in (".github/workflows/release.yml", "scripts/publish_release.ps1", "scripts/verify_bundle.py"):
        if "brendigo" in read(rel).lower():
            fail(f"legacy branding remains in release surface: {rel}")

    print(f"RELEASE_AUDIT=PASS ({version})")
    print("RELEASE_MATRIX=WINDOWS_X64_X86,LINUX_AMD64_ARM64_I386,MACOS_UNIVERSAL,ANDROID_SOURCE_GATE")
    print("ANDROID_PUBLIC_APK=REQUIRES_EXTERNAL_PRODUCTION_SIGNING_IDENTITY")
    print("PUBLISHER=CENTRALIZED")
    print("RELEASE_GITHUB_API=WRAPPED_AND_AUDITED")
    return 0


if __name__ == "__main__":
    sys.exit(main())
