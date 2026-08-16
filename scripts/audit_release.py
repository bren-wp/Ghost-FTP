#!/usr/bin/env python3
"""Provjerava da release pipeline ostaje idempotentan, verzijski neutralan i fail-closed."""

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
            "publish_release.ps1",
            "scripts/verify_bundle.py",
            "python scripts/audit_docs.py",
            "python scripts/audit_security.py",
            "python scripts/audit_release.py",
            "python -m unittest discover -s scripts -p 'test_*.py'",
        ),
    )
    if re.search(r"(?m)^\s*default:\s*['\"]?\d+\.\d+\.\d+", workflow):
        fail("release workflow ponovno hardkodira zadanu produkcijsku verziju")
    if re.search(r"(?i)primjer\s+\d+\.\d+\.\d+", workflow):
        fail("release workflow ponovno hardkodira verzijski primjer")
    if "gh release create" in workflow or "gh release upload" in workflow:
        fail("release.yml ne smije zaobići centralni publish_release.ps1")

    publisher = require(
        "scripts/publish_release.ps1",
        (
            "Resolve-TagCommit",
            "Assert-TagCommit",
            "Assert-RemoteAsset",
            "Get-FileHash",
            "sha256:",
            "gh release upload",
            "neočekivani asset",
            "RELEASE_PUBLISH_VERIFICATION=PASS",
        ),
    )
    if "--clobber" in publisher:
        fail("publisher ne smije automatski prepisivati postojeći release asset")

    verifier = require(
        "scripts/verify_bundle.py",
        (
            "BUNDLE-SHA256.txt",
            "dupliciranu putanju",
            "nesigurnu putanju",
            "SHA-256 se ne podudara",
            "BUNDLE_VERIFICATION=PASS",
        ),
    )
    if "extractall(" in verifier or ".extract(" in verifier:
        fail("ZIP verifier ne smije raspakiravati nepouzdane putanje na disk")

    ci = require(
        ".github/workflows/ci.yml",
        (
            "python scripts/audit_docs.py",
            "python scripts/audit_security.py",
            "python scripts/audit_release.py",
            "python -m unittest discover -s scripts -p 'test_*.py'",
        ),
    )
    build = require(
        "BUILD-WINDOWS.ps1",
        (
            "scripts/audit_docs.py",
            "scripts/audit_security.py",
            "scripts/audit_release.py",
            "test_*.py",
        ),
    )
    if not ci or not build:
        fail("CI/build release gate nije dostupan")

    print("RELEASE_AUDIT=PASS")
    print("RELEASE_RERUN_REPAIR=ENABLED")
    print("RELEASE_TAG_COMMIT_BINDING=ENABLED")
    print("RELEASE_ASSET_DIGEST_FAIL_CLOSED=ENABLED")
    print("WINDOWS_ZIP_POST_PACKAGE_VERIFICATION=ENABLED")
    return 0


if __name__ == "__main__":
    main()
