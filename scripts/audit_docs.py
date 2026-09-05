#!/usr/bin/env python3
"""Validate Ghost FTP documentation links and the active Windows/Linux release contract."""

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
VERSIONED_DOC_TITLE_RE = re.compile(r"(?m)^#\s+(?:Ghost FTP|GhostFTP)\s+\d+\.\d+\.\d+\s+—")
CURRENT_RELEASE_RE = re.compile(r"\*\*Current Ghost FTP release:\s*(\d+\.\d+\.\d+)\*\*")
IGNORED_PREFIXES = ("http://", "https://", "mailto:", "data:", "//", "#")
RETIRED_ACTIVE_LINKS = ("../android/", "../ios/", "../macos/")


def fail(message: str) -> None:
    raise SystemExit("DOCS_AUDIT_FAILED: " + message)


def markdown_files() -> list[Path]:
    return sorted(path for path in ROOT.rglob("*.md") if ".git" not in path.parts)


def clean_destination(raw: str) -> str:
    value = raw.strip()
    if value.startswith("<") and ">" in value:
        value = value[1 : value.index(">")]
    else:
        value = value.split(maxsplit=1)[0] if value else value
    value = unquote(value)
    value = value.split("#", 1)[0].split("?", 1)[0]
    return value.strip()


def check_link(source: Path, raw: str) -> None:
    destination = clean_destination(raw)
    if not destination or destination.lower().startswith(IGNORED_PREFIXES):
        return
    target = (source.parent / destination).resolve()
    try:
        target.relative_to(ROOT.resolve())
    except ValueError:
        fail(f"link escapes the repository: {source.relative_to(ROOT)} -> {raw}")
    if not target.exists():
        fail(f"missing local link: {source.relative_to(ROOT)} -> {raw}")


def main() -> int:
    files = markdown_files()
    if not files:
        fail("no Markdown documents were found")
    if not INDEX.is_file():
        fail("docs/README.md is missing")

    for path in files:
        text = path.read_text(encoding="utf-8")
        if path.parent == DOCS and VERSIONED_DOC_TITLE_RE.search(text):
            fail(f"long-lived document has a versioned title: {path.relative_to(ROOT)}")
        for match in MARKDOWN_LINK_RE.finditer(text):
            check_link(path, match.group(1))
        for match in HTML_LINK_RE.finditer(text):
            check_link(path, match.group(1))

    version = (ROOT / "VERSION").read_text(encoding="utf-8").strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        fail(f"invalid canonical VERSION: {version!r}")
    if int(version.split(".", 1)[0]) < 2:
        fail("active documentation contract requires the 2.x Windows/Linux line")

    index_text = INDEX.read_text(encoding="utf-8")
    root_readme = (ROOT / "README.md").read_text(encoding="utf-8")
    if not root_readme.startswith("# Ghost FTP\n"):
        fail("root README does not use the Ghost FTP public title")
    if f"Current Ghost FTP version: **{version}**" not in root_readme:
        fail("root README current-version marker does not match VERSION")
    if f"**Current Ghost FTP release: {version}**" not in index_text:
        fail("documentation index current-release marker does not match VERSION")

    for path in files:
        text = path.read_text(encoding="utf-8")
        for match in CURRENT_RELEASE_RE.finditer(text):
            if match.group(1) != version:
                fail(f"stale current-release marker in {path.relative_to(ROOT)}: {match.group(1)} != {version}")

    for marker in ("9 platform artifacts", "12 public files"):
        if marker not in root_readme or marker not in index_text:
            fail(f"release-contract documentation is missing current marker: {marker}")

    if not index_text.startswith("# Ghost FTP documentation\n"):
        fail("documentation index does not use the Ghost FTP public title")
    for marker in ("Windows", "Linux", "PLATFORM-PARITY.md", "Web companion"):
        if marker not in index_text:
            fail(f"documentation index is missing platform marker: {marker}")

    detailed_docs = sorted(path for path in DOCS.glob("*.md") if path.name != "README.md")
    for path in detailed_docs:
        if path.name not in index_text:
            fail(f"docs/README.md does not index {path.name}")
        if f"docs/{path.name}" not in root_readme:
            fail(f"root README does not link docs/{path.name}")

    for required in ("docs/PLATFORM-PARITY.md", "linux/README.md", "GhostFTP WEB/README.md"):
        if not (ROOT / required).is_file():
            fail(f"missing required active-platform/companion documentation: {required}")

    for rel in ("README.md", "docs/README.md", "docs/INSTALLATION.md", "docs/ARCHITECTURE.md", "docs/ROADMAP.md", "docs/GITHUB-RELEASES.md"):
        text = (ROOT / rel).read_text(encoding="utf-8")
        lowered = text.lower()
        for retired in RETIRED_ACTIVE_LINKS:
            if retired in lowered:
                fail(f"active documentation links to retired platform path: {rel} -> {retired}")

    parity = (DOCS / "PLATFORM-PARITY.md").read_text(encoding="utf-8")
    for marker in (
        "Windows and Linux platform parity",
        "SFTP password",
        "SFTP key passphrase",
        "24-language",
        "same typed `internal/api.Engine`",
    ):
        if marker not in parity:
            fail(f"platform parity documentation is missing marker: {marker}")

    print(f"DOCS_AUDIT=PASS ({version}; {len(files)} Markdown files, {len(detailed_docs)} detailed documents)")
    print("PUBLIC_BRAND=Ghost FTP")
    print("ACTIVE_APPLICATION_PLATFORMS=WINDOWS,LINUX")
    print("PUBLIC_PLATFORM_ARTIFACTS=9")
    print("PUBLIC_RELEASE_FILES=12")
    return 0


if __name__ == "__main__":
    sys.exit(main())
