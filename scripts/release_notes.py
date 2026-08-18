#!/usr/bin/env python3
"""Generira hrvatske ByFTP bilješke iz točno odgovarajućeg CHANGELOG odjeljka."""

from __future__ import annotations

import argparse
from pathlib import Path
import re
import sys


def extract_section(changelog: str, version: str) -> str:
    header = re.compile(rf"^##\s+{re.escape(version)}(?:\s|$).*$", re.MULTILINE)
    match = header.search(changelog)
    if not match:
        raise ValueError(f"CHANGELOG odjeljak za verziju {version} nije pronađen")
    start = changelog.find("\n", match.end())
    if start < 0:
        return ""
    start += 1
    next_header = re.search(r"^##\s+", changelog[start:], re.MULTILINE)
    end = start + next_header.start() if next_header else len(changelog)
    return changelog[start:end].strip()


def build_notes(version: str, section: str) -> str:
    return f"""ByFTP {version}

FTP / FTPS / SFTP klijent tvrtke Brendigo za Windows, Linux i macOS.

Najvažnije promjene
-------------------
{section}

Datoteke za preuzimanje
-----------------------
Windows:
- Setup x64 EXE: preporučena 64-bitna instalacija
- Portable x64 EXE: 64-bitno pokretanje bez instalacije
- Windows x64 ZIP: provjereni 64-bitni paket s dokumentacijom
- Setup x86 EXE: 32-bitna instalacija za kompatibilne Windows sustave
- Portable x86 EXE: 32-bitno pokretanje bez instalacije
- Windows x86 ZIP: provjereni 32-bitni paket s dokumentacijom

Linux:
- Linux amd64 DEB
- Linux arm64 DEB
- Linux i386 DEB

macOS:
- macOS Universal PKG: Intel + Apple Silicon

Zajedničko:
- SHA256.txt: kontrolni sažeci objavljenih paketa
- RELEASE-NOTES.txt: ove bilješke izdanja
- BUILD-METADATA.txt: podrijetlo commita i buildova

Standalone Uninstaller, interni verification izvještaj i dodatni ByFTP Source ZIP nisu javni release asseti. GitHub i dalje automatski prikazuje vlastite Source code ZIP/TAR poveznice za svaki tag.

Windows binariji nisu Authenticode potpisani bez stvarnog Brendigo code-signing certifikata, a macOS paket nije Developer ID potpisan bez stvarnog Apple certifikata. Provjerite SHA-256 prije distribucije.
"""


def main() -> int:
    parser = argparse.ArgumentParser(description="Generiranje hrvatskih ByFTP bilješki iz CHANGELOG-a")
    parser.add_argument("--version", required=True)
    parser.add_argument("--changelog", default="CHANGELOG.md")
    parser.add_argument("--output", required=True)
    args = parser.parse_args()

    version = args.version.strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        print(f"neispravna verzija izdanja: {version}", file=sys.stderr)
        return 2
    try:
        changelog = Path(args.changelog).read_text(encoding="utf-8")
        section = extract_section(changelog, version)
        if not section:
            raise ValueError(f"CHANGELOG odjeljak za verziju {version} je prazan")
        Path(args.output).write_text(build_notes(version, section), encoding="utf-8", newline="\n")
    except (OSError, ValueError) as exc:
        print(str(exc), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
