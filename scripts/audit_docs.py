#!/usr/bin/env python3
"""Provjerava lokalne poveznice, indeks i verzijsku neutralnost ByFTP dokumentacije."""

from __future__ import annotations

import re
import sys
from pathlib import Path
from urllib.parse import unquote

ROOT = Path(__file__).resolve().parents[1]
DOCS = ROOT / "docs"
INDEX = DOCS / "README.md"

MARKDOWN_LINK_RE = re.compile(r"!?\[[^\]]*\]\(([^)\n]+)\)")
HTML_LINK_RE = re.compile(r"\b(?:href|src)\s*=\s*[\"']([^\"']+)[\"']", re.IGNORECASE)
VERSIONED_DOC_TITLE_RE = re.compile(r"(?m)^#\s+ByFTP\s+\d+\.\d+\.\d+\s+—")
IGNORED_PREFIXES = ("http://", "https://", "mailto:", "data:", "//", "#")


def fail(message: str) -> None:
    raise SystemExit("DOCS_AUDIT_NIJE_PROSAO: " + message)


def markdown_files() -> list[Path]:
    return sorted(path for path in ROOT.rglob("*.md") if ".git" not in path.parts)


def clean_destination(raw: str) -> str:
    value = raw.strip()
    if value.startswith("<") and ">" in value:
        value = value[1 : value.index(">")]
    else:
        # Markdown naslov nakon odredišta nije dio putanje. ByFTP lokalne putanje
        # nemaju razmake, pa je prvi token dovoljan i ne mijenja stvarne linkove.
        value = value.split(maxsplit=1)[0] if value else value
    value = unquote(value)
    value = value.split("#", 1)[0].split("?", 1)[0]
    return value.strip()


def check_link(source: Path, raw: str) -> None:
    destination = clean_destination(raw)
    if not destination or destination.lower().startswith(IGNORED_PREFIXES):
        return
    # Root README koristi GitHub UI-relativnu Actions poveznicu koja namjerno
    # izlazi iz repozitorijskog filesystema; ona nije lokalni dokument.
    if source == ROOT / "README.md" and destination.startswith("../../actions/"):
        return
    target = (source.parent / destination).resolve()
    try:
        target.relative_to(ROOT.resolve())
    except ValueError:
        fail(f"poveznica izlazi iz repozitorija: {source.relative_to(ROOT)} -> {raw}")
    if not target.exists():
        fail(f"nepostojeća lokalna poveznica: {source.relative_to(ROOT)} -> {raw}")


def main() -> int:
    files = markdown_files()
    if not files:
        fail("nisu pronađeni Markdown dokumenti")
    if not INDEX.is_file():
        fail("nedostaje docs/README.md")

    for path in files:
        text = path.read_text(encoding="utf-8")
        if path.parent == DOCS and VERSIONED_DOC_TITLE_RE.search(text):
            fail(f"dokument ima zastarjeli verzionirani naslov: {path.relative_to(ROOT)}")
        for match in MARKDOWN_LINK_RE.finditer(text):
            check_link(path, match.group(1))
        for match in HTML_LINK_RE.finditer(text):
            check_link(path, match.group(1))

    index_text = INDEX.read_text(encoding="utf-8")
    root_readme = (ROOT / "README.md").read_text(encoding="utf-8")
    detailed_docs = sorted(path for path in DOCS.glob("*.md") if path.name != "README.md")
    for path in detailed_docs:
        if path.name not in index_text:
            fail(f"docs/README.md ne indeksira {path.name}")
        if f"docs/{path.name}" not in root_readme:
            fail(f"glavni README ne povezuje dokument docs/{path.name}")

    print(f"DOCS_AUDIT=PROSAO ({len(files)} Markdown datoteka, {len(detailed_docs)} detaljnih dokumenata)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
