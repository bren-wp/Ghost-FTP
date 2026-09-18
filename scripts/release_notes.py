#!/usr/bin/env python3
"""Generate Ghost FTP release notes from the matching CHANGELOG section."""

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
    parts = version.split(".")
    major = int(parts[0])
    minor_alias = ".".join(parts[:2])

    return f"""Ghost FTP {version}

Privacy-first FTP/FTPS/SFTP workspace with public Windows, Linux and Android applications plus privacy-minimal browser helper packages.
Release channel: Current.
GitHub prerelease flag: false.

Highlights
----------
{section}

Release tag
-----------
ghostftp-v{version}

Public platform packages
------------------------
Windows:
- Ghost-FTP-{version}-Setup.exe — one self-contained universal Windows Setup with verified native x64/x86/ARM64 payloads and required trusted Authenticode.
- Ghost-FTP-{version}-Portable.exe — one self-contained universal Windows Portable with verified native x64/x86/ARM64 payloads and required trusted Authenticode.

Linux / Debian:
- Ghost-FTP-{version}-Linux-Debian-Installer.run — one architecture-selecting installer carrying amd64, arm64 and i386 payloads.
- Ghost-FTP-{version}-Linux-Debian-Portable.tar.gz — one architecture-selecting portable bundle carrying amd64, arm64 and i386 payloads.

Linux / Ubuntu:
- Ghost-FTP-{version}-Linux-Ubuntu-Installer.run — one architecture-selecting installer carrying amd64, arm64 and i386 payloads.
- Ghost-FTP-{version}-Linux-Ubuntu-Portable.tar.gz — one architecture-selecting portable bundle carrying amd64, arm64 and i386 payloads.

Linux / Fedora:
- Ghost-FTP-{version}-Linux-Fedora-Installer.run — one architecture-selecting installer carrying amd64, arm64 and i386 payloads.
- Ghost-FTP-{version}-Linux-Fedora-Portable.tar.gz — one architecture-selecting portable bundle carrying amd64, arm64 and i386 payloads.

Android:
- Ghost-FTP-{version}-Android.apk — one protected production-signed APK. Android SFTP remains hidden until strict maintained host-key verification exists.

Browser helper packages:
- Ghost-FTP-{version}-Chrome-Extension.zip
- Ghost-FTP-{version}-Edge-Extension.zip
- Ghost-FTP-{version}-Firefox-Extension.zip
- Ghost-FTP-{version}-Opera-Extension.zip
- These remain zero-permission local helpers. On supported Windows installs, the explicit Open in Ghost FTP action uses a sanitized ghostftp://connect handoff that excludes secrets, query data and fragments and never auto-connects.

GitHub Packages
---------------
- Package: ghcr.io/bren-wp/ghost-ftp:{version}
- Type: verified OCI distribution bundle, not a runtime container.
- Contents: the same verified release directory under /ghostftp-release/.
- Current aliases: {major}, {minor_alias}, latest.
- The workflow verifies the exact-version registry readback before completing publication.

Verification files
------------------
- SHA256.txt — SHA-256 checksums for every public release file except SHA256.txt itself.
- RELEASE-NOTES.txt — these notes generated from CHANGELOG.md.
- BUILD-METADATA.txt — version, release tag, exact source commit, signing/evidence state and distribution metadata.

Release contract
----------------
- Current Ghost FTP releases are not inferred to be prereleases from semantic-version major zero.
- 13 platform artifacts.
- 16 public release files total, including BUILD-METADATA.txt, RELEASE-NOTES.txt and SHA256.txt.
- Public application platforms: Windows, Linux and Android.
- Public browser-helper packages: Chrome, Edge, Firefox and Opera.
- macOS remains a separately validated development/source frontend until real Developer ID signing and Apple notarization succeed.
- Local language catalog: 24 selectable desktop languages with English default/fallback.
- Application telemetry: disabled.
- Each public Linux distro gets one Installer and one Portable archive. Both carry amd64, arm64 and i386 payloads and select the native payload locally; native CI runtime evidence is reported separately from build/package evidence.
- Publication is bound to the exact verified main commit and followed by canonical latest-only retention verification.

Signing and trust
-----------------
- Official Windows publication is signed-only and requires trusted Authenticode; there is no supported unsigned official-publication continuation.
- Official Android publication requires the protected production publisher identity and an exact signer-certificate SHA-256 fingerprint match.
- The production workflow never generates a replacement long-lived Windows or Android publisher identity.
- Android production signing does not expose SFTP without strict maintained host-key verification.
- Always verify SHA256.txt and the official GitHub release location before installation or deployment.

Privacy
-------
Release bundles contain only the explicit verified artifact allow-list. They do not contain saved profiles, FTP/SFTP passwords, private-key passphrases, signing private keys, local application data or user files.
"""


def main() -> int:
    parser = argparse.ArgumentParser(description="Generate Ghost FTP release notes from CHANGELOG.md")
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
