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

Privatan FTP / FTPS / SFTP klijent tvrtke Brendigo za Windows, Linux i macOS.

Najvažnije promjene
-------------------
{section}

Službeni paketi
---------------
Windows:
- Setup x64 EXE: preporučena instalacija za 64-bitni Windows 10/11
- Portable x64 EXE: 64-bitno pokretanje bez instalacije
- Windows x64 ZIP: Setup + Portable + dokumentacija i bundle checksumovi
- Setup x86 EXE: instalacija za podržani 32-bitni Windows
- Portable x86 EXE: 32-bitno pokretanje bez instalacije
- Windows x86 ZIP: Setup + Portable + dokumentacija i bundle checksumovi

Linux:
- Linux amd64 DEB
- Linux arm64 DEB
- Linux i386 DEB

macOS:
- macOS Universal PKG: Intel x86_64 + Apple Silicon arm64

Provjera izdanja:
- SHA256.txt: SHA-256 svih javnih paketa i zajedničkih release metapodataka
- RELEASE-NOTES.txt: ove bilješke izdanja
- BUILD-METADATA.txt: verzija, release commit i podaci produkcijskog build runa

Preporuka prije instalacije
---------------------------
1. Preuzmite paket koji odgovara operacijskom sustavu i arhitekturi.
2. Usporedite SHA-256 preuzetog paketa sa službenim SHA256.txt.
3. Windows korisnici za uobičajenu instalaciju trebaju odabrati Setup paket; Portable je namijenjen radu bez instalacije.
4. Linux i macOS izdanje koriste terminalno sučelje nad istim ByFTP engineom i sigurnosnim transfer slojem.

Sigurnost potpisa
-----------------
Windows binariji nemaju status Verified Publisher dok nije dostupan stvarni Brendigo Authenticode certifikat. macOS PKG nije Developer ID potpisan/notariziran bez stvarnog Apple certifikata. Workflow ne fabricira publisher identitet; SHA-256 i release provenance ostaju obavezni dio izdanja.
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
