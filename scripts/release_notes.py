#!/usr/bin/env python3
"""Generate English ByFTP release notes from the exact matching CHANGELOG section."""

from __future__ import annotations

import argparse
from pathlib import Path
import re
import sys


def extract_section(changelog: str, version: str) -> str:
    header = re.compile(rf"^##\s+{re.escape(version)}(?:\s|$).*$", re.MULTILINE)
    match = header.search(changelog)
    if not match:
        raise ValueError(f"CHANGELOG section for version {version} was not found")
    start = changelog.find("\n", match.end())
    if start < 0:
        return ""
    start += 1
    next_header = re.search(r"^##\s+", changelog[start:], re.MULTILINE)
    end = start + next_header.start() if next_header else len(changelog)
    return changelog[start:end].strip()


def build_notes(version: str, section: str) -> str:
    return f"""ByFTP {version}

Privacy-focused file-transfer client for Windows, Linux, macOS, Android and iOS.

Highlights
----------
{section}

Official packages
-----------------
Windows:
- Setup x64 EXE: recommended installer for 64-bit Windows 10/11
- Portable x64 EXE: 64-bit build that runs without installation
- Windows x64 ZIP: Setup + Portable + documentation + bundle checksums
- Setup x86 EXE: installer for supported 32-bit Windows systems
- Portable x86 EXE: 32-bit build that runs without installation
- Windows x86 ZIP: Setup + Portable + documentation + bundle checksums

Linux:
- Linux amd64 DEB
- Linux arm64 DEB
- Linux i386 DEB

macOS:
- macOS Universal PKG: Intel x86_64 + Apple Silicon arm64

Android:
- Android debug APK: installable development/test build signed with the standard Android debug identity
- Android release-unsigned APK: optimized/minified release build intended for verification or external production signing

iOS:
- iOS arm64 unsigned IPA: real iPhoneOS device application in the normal Payload/ByFTP.app IPA structure
- iOS arm64 unsigned app ZIP: the same unsigned ByFTP.app bundle packaged for verification or an external Apple signing workflow

Android and iOS artifacts are generated from the same canonical VERSION and gated source as the desktop packages. Android production distribution still requires a stable private Android signing identity. The iOS IPA/app ZIP require a valid Apple signing identity and provisioning configuration before normal device/App Store/TestFlight distribution. Unsigned/debug files are never represented as production store-signed builds.

Release verification
--------------------
- SHA256.txt: SHA-256 checksums for every public package and shared release metadata file
- RELEASE-NOTES.txt: these release notes generated from the matching CHANGELOG section
- BUILD-METADATA.txt: version, release commit and production workflow provenance

Before installing
-----------------
1. Download the package that matches your operating system and architecture.
2. Compare the downloaded package SHA-256 hash with the official SHA256.txt file.
3. On Windows, use Setup for a normal installation; use Portable when installation is not required.
4. On Android, use the debug APK only for testing/development installs. The release-unsigned APK must be signed with a trusted external production identity before production distribution.
5. On iOS, treat the unsigned IPA/app ZIP as reproducible pre-signing evidence. Apply a valid Apple identity/provisioning profile externally before normal device or store distribution.

Signing status
--------------
Windows binaries do not show Verified Publisher until a real ByFTP Authenticode certificate is available. The macOS PKG is not Developer ID signed/notarized without a real Apple signing identity. Android production distribution requires a stable private signing identity. iOS production distribution requires an Apple signing identity and provisioning profile outside this repository. The workflow never fabricates publisher identities; SHA-256 verification and release provenance remain mandatory parts of public releases.
"""


def main() -> int:
    parser = argparse.ArgumentParser(description="Generate ByFTP release notes from CHANGELOG.md")
    parser.add_argument("--version", required=True)
    parser.add_argument("--changelog", default="CHANGELOG.md")
    parser.add_argument("--output", required=True)
    args = parser.parse_args()
    version = args.version.strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        print(f"invalid release version: {version}", file=sys.stderr)
        return 2
    try:
        changelog = Path(args.changelog).read_text(encoding="utf-8")
        section = extract_section(changelog, version)
        if not section:
            raise ValueError(f"CHANGELOG section for version {version} is empty")
        Path(args.output).write_text(build_notes(version, section), encoding="utf-8", newline="\n")
    except (OSError, ValueError) as exc:
        print(str(exc), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
