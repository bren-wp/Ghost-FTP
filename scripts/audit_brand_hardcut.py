#!/usr/bin/env python3
"""Fail closed on retired branding and author-identity leakage outside About."""

from __future__ import annotations

import re
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RETIRED = re.compile(r"by[\s_-]?ftp", re.IGNORECASE)
AUTHOR_BRAND = re.compile(r"brendigo", re.IGNORECASE)

# Legal/historical records retain their original ownership/release history. Tests
# and this audit may name the author identity while proving that active product
# surfaces do not. The only runtime source allowed to expose it is About.
ALLOWED_AUTHOR_IDENTITY = {
    "LICENSE",
    "CHANGELOG.md",
    "docs/RELEASE-HISTORY.md",
    "internal/desktop/about_identity_windows.go",
    "macos/Bridge/about_identity.go",
    "internal/brand/runtime_metadata_test.go",
    "scripts/audit_brand_hardcut.py",
    "scripts/test_about_card_release.py",
    "scripts/test_official_destinations_contract.py",
    "scripts/test_linux_distro_packaging_contract.py",
}


def fail(message: str) -> None:
    raise SystemExit("BRAND_HARDCUT_FAILED: " + message)


def main() -> int:
    raw = subprocess.check_output(["git", "ls-files", "-z"], cwd=ROOT)
    violations: list[str] = []

    for item in raw.split(b"\0"):
        if not item:
            continue
        rel = item.decode("utf-8", "strict")
        if RETIRED.search(rel):
            violations.append("retired-path:" + rel)
            continue

        path = ROOT / rel
        if not path.is_file():
            continue
        try:
            text = path.read_text(encoding="utf-8")
        except (UnicodeDecodeError, OSError):
            continue

        if RETIRED.search(text):
            violations.append("retired-content:" + rel)
        if rel not in ALLOWED_AUTHOR_IDENTITY and AUTHOR_BRAND.search(text):
            violations.append("author-identity-outside-about:" + rel)

    about = ROOT / "internal" / "desktop" / "about_identity_windows.go"
    if not about.is_file():
        violations.append("missing-about-identity-source")
    else:
        about_text = about.read_text(encoding="utf-8")
        for marker in (
            'aboutPublisher     = "BRENDIGO LTD"',
            'aboutAuthorWebsite = "brendigo.com"',
            'aboutSupport       = "brendigo.com/kontakt"',
        ):
            if marker not in about_text:
                violations.append("about-identity-contract:" + marker)

    mac_about = ROOT / "macos" / "Bridge" / "about_identity.go"
    if not mac_about.is_file():
        violations.append("missing-macos-about-identity-source")
    else:
        mac_about_text = mac_about.read_text(encoding="utf-8")
        for marker in (
            'macAboutPublisher     = "BRENDIGO LTD"',
            'macAboutAuthorWebsite = "brendigo.com"',
            'macAboutSupport       = "brendigo.com/kontakt"',
        ):
            if marker not in mac_about_text:
                violations.append("macos-about-identity-contract:" + marker)

    if violations:
        fail("branding contract violation: " + ", ".join(violations))

    print("BRAND_HARDCUT=PASS")
    print("PUBLIC_BRAND=Ghost FTP")
    print("AUTHOR_IDENTITY_SURFACE=ABOUT_ONLY")
    print("TECHNICAL_IDENTITY=GhostFTP")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
