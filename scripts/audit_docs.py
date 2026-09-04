#!/usr/bin/env python3
"""Validate Ghost FTP documentation links, indexing and durable titles."""

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
VERSIONED_DOC_TITLE_RE = re.compile(r"(?m)^#\s+(?:Ghost FTP|ByFTP)\s+\d+\.\d+\.\d+\s+—")
IGNORED_PREFIXES = ("http://", "https://", "mailto:", "data:", "//", "#")


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

    index_text = INDEX.read_text(encoding="utf-8")
    root_readme = (ROOT / "README.md").read_text(encoding="utf-8")
    if not root_readme.startswith("# Ghost FTP\n"):
        fail("root README does not use the Ghost FTP public title")
    if not index_text.startswith("# Ghost FTP documentation\n"):
        fail("documentation index does not use the Ghost FTP public title")

    detailed_docs = sorted(path for path in DOCS.glob("*.md") if path.name != "README.md")
    for path in detailed_docs:
        if path.name not in index_text:
            fail(f"docs/README.md does not index {path.name}")
        if f"docs/{path.name}" not in root_readme:
            fail(f"root README does not link docs/{path.name}")

    for platform_readme in ("linux/README.md", "macos/README.md", "android/README.md", "ios/README.md"):
        if not (ROOT / platform_readme).is_file():
            fail(f"missing platform documentation: {platform_readme}")
        if platform_readme not in root_readme:
            fail(f"root README does not link {platform_readme}")

    print(f"DOCS_AUDIT=PASS ({len(files)} Markdown files, {len(detailed_docs)} detailed documents)")
    print("PUBLIC_BRAND=Ghost FTP")
    print("PLATFORM_DOCS=LINUX,MACOS,ANDROID,IOS,WEB")
    return 0


if __name__ == "__main__":
    sys.exit(main())
