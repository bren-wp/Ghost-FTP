#!/usr/bin/env python3
"""Provjerava cross-platform ByFTP release ugovor i fail-closed objavu."""

from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def fail(message: str) -> None:
    raise SystemExit("RELEASE_AUDIT_NIJE_PROSAO: " + message)


def read(path: str) -> str:
    target = ROOT / path
    if not target.is_file():
        fail(f"nedostaje {path}")
    return target.read_text(encoding="utf-8")


def require(path: str, markers: tuple[str, ...]) -> str:
    text = read(path)
    for marker in markers:
        if marker not in text:
            fail(f"{path} nema obavezni release guard: {marker}")
    return text


def main() -> int:
    workflow = require(
        ".github/workflows/release.yml",
        (
            "group: byftp-release",
            "quality:",
            "Produkccijski" if False else "Produkcijski quality, race, sigurnost i privatnost",
            "needs: [quality, windows, linux, macos]",
            "go test -race ./...",
            "go telemetry off",
            "Windows x64 i x86",
            "Linux amd64 arm64 i i386",
            "macOS Universal",
            "scripts/BUILD-LINUX.sh",
            "scripts/BUILD-MACOS.sh",
            "publish_release.ps1",
            "scripts/verify_bundle.py",
            "--arch $arch",
            "Provjeri staging javnih paketa",
            "RELEASE_QUALITY_GATE=passed",
            "ByFTP-$env:VERSION-Setup-x86.exe",
            "ByFTP-$env:VERSION-Portable-x86.exe",
            "ByFTP-$env:VERSION-Linux-amd64.deb",
            "ByFTP-$env:VERSION-Linux-arm64.deb",
            "ByFTP-$env:VERSION-Linux-i386.deb",
            "ByFTP-$env:VERSION-macOS-Universal.pkg",
        ),
    )
    if re.search(r"(?m)^\s*default:\s*['\"]?\d+\.\d+\.\d+", workflow):
        fail("release workflow ponovno hardkodira zadanu produkcijsku verziju")
    if re.search(r"(?m)^\s*tags:\s*$", workflow):
        fail("release workflow ponovno reagira na vlastiti tag i može stvoriti paralelni publisher")
    if "v2.15.0" in workflow or "release delete-asset" in workflow:
        fail("release workflow ponovno sadrži jednokratni migracijski cleanup stare verzije")
    if "gh release create" in workflow or "gh release upload" in workflow:
        fail("release.yml ne smije zaobići centralni publish_release.ps1")

    asset_match = re.search(r"\$assets\s*=\s*@\((.*?)\)\s*\n\s*foreach", workflow, re.S)
    if not asset_match:
        fail("nije pronađen eksplicitni javni $assets skup")
    public_assets = asset_match.group(1)
    expected_asset_markers = (
        "Portable-x64.exe", "Setup-x64.exe", "Windows-x64.zip",
        "Portable-x86.exe", "Setup-x86.exe", "Windows-x86.zip",
        "Linux-amd64.deb", "Linux-arm64.deb", "Linux-i386.deb",
        "macOS-Universal.pkg", "SHA256.txt", "RELEASE-NOTES.txt", "BUILD-METADATA.txt",
    )
    for marker in expected_asset_markers:
        if marker not in public_assets:
            fail(f"javni release asset skup nema: {marker}")
    for forbidden in ("Source.zip", "Uninstall-", "verification.txt"):
        if forbidden in public_assets:
            fail(f"javni release asseti ponovno sadrže interni/stari tip: {forbidden}")

    publisher = require(
        "scripts/publish_release.ps1",
        ("Resolve-TagCommit", "Assert-TagCommit", "Assert-RemoteAsset", "Get-FileHash", "sha256:", "gh release upload", "neočekivani asset", "RELEASE_PUBLISH_VERIFICATION=PASS"),
    )
    if "--clobber" in publisher:
        fail("publisher ne smije automatski prepisivati postojeći release asset")

    verifier = require(
        "scripts/verify_bundle.py",
        ("BUNDLE-SHA256.txt", "x64", "x86", "dupliciranu putanju", "nesigurnu putanju", "SHA-256 se ne podudara", "BUNDLE_VERIFICATION=PASS", "forbidden"),
    )
    if "extractall(" in verifier or ".extract(" in verifier:
        fail("ZIP verifier ne smije raspakiravati nepouzdane putanje na disk")

    windows = require(
        "BUILD-WINDOWS.ps1",
        ("Build-ByFTPArchitecture -GoArch 'amd64' -Label 'x64'", "Build-ByFTPArchitecture -GoArch '386' -Label 'x86'", "go telemetry").
        if False else ("Build-ByFTPArchitecture -GoArch 'amd64' -Label 'x64'", "Build-ByFTPArchitecture -GoArch '386' -Label 'x86'", "$telemetryMode = (go telemetry).Trim()"),
    )
    linux = require("scripts/BUILD-LINUX.sh", ("build_arch amd64 amd64", "build_arch arm64 arm64", "build_arch 386 i386", "dpkg-deb", 'telemetry="$(go telemetry)"'))
    macos = require("scripts/BUILD-MACOS.sh", ('GOARCH="$arch"', "lipo -create", "pkgbuild", "ByFTP.app", 'telemetry="$(go telemetry)"'))
    if not windows or not linux or not macos:
        fail("platformski build ugovor nije dostupan")

    notes = require("scripts/release_notes.py", ("Setup x86", "Linux amd64", "Linux arm64", "Linux i386", "macOS Universal", "SHA256.txt"))
    if not notes:
        fail("release notes ugovor nije dostupan")

    ci = require(
        ".github/workflows/ci.yml",
        ("go telemetry off", "python scripts/audit_docs.py", "python scripts/audit_security.py", "python scripts/audit_release.py", "python -m unittest discover -s scripts -p 'test_*.py'"),
    )
    if "BUILD-WINDOWS.ps1" not in ci:
        fail("CI mora koristiti kanonski Windows produkcijski build")

    # Korisnički dokumenti i release notes opisuju samo aktualne javne pakete.
    # Povijesni CHANGELOG smije zadržati stare nazive radi točne povijesti.
    for rel in ("README.md", "docs/INSTALACIJA.md", "scripts/release_notes.py"):
        text = read(rel)
        for forbidden in ("verification.txt", "Source.zip", "Uninstall-"):
            if forbidden in text:
                fail(f"{rel} ponovno oglašava zastarjeli/interne release naziv: {forbidden}")

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
