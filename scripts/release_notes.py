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

Izvorni Windows FTP / FTPS / SFTP klijent tvrtke Brendigo.

Najvažnije promjene
-------------------
{section}

Datoteke za preuzimanje
-----------------------
- Setup x64 EXE: preporučena instalacija
- Portable x64 EXE: pokretanje bez instalacije
- Uninstaller x64 EXE: samostalni program za uklanjanje
- Windows x64 ZIP: kompletan spreman Windows paket
- Source ZIP: točna snimka praćenog izvornog koda ovog izdanja
- SHA256.txt: kontrolni sažeci javnih artefakata
- verification.txt: izvještaj PE/sigurnosne provjere
- BUILD-METADATA.txt: podrijetlo izvornog commita i build alata

Prije distribucije provjerite SHA-256 vrijednosti. Javni produkcijski binariji trebaju biti Authenticode potpisani stvarnim Brendigo potpisnim identitetom.
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
