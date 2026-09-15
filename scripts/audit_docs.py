#!/usr/bin/env python3
"""Validate maintained Ghost FTP documentation against the active repository contract."""
from __future__ import annotations

import re
import sys
from pathlib import Path
from urllib.parse import unquote

ROOT = Path(__file__).resolve().parents[1]
MARKDOWN_LINK_RE = re.compile(r"!?\[[^\]]*\]\(([^)\n]+)\)")
MARKDOWN_IMAGE_RE = re.compile(r"!\[[^\]]*\]\(([^)\n]+)\)")
HTML_LINK_RE = re.compile(r"\b(?:href|src)\s*=\s*[\"']([^\"']+)[\"']", re.I)
HTML_IMAGE_RE = re.compile(r"<img\b[^>]*\bsrc\s*=\s*[\"']([^\"']+)[\"']", re.I)
IGNORED_PREFIXES = ("http://", "https://", "mailto:", "data:", "//", "#")
REMOTE_MEDIA_PREFIXES = ("http://", "https://", "data:", "//")
ACTIVE_DOCS = (
    "README.md",
    "docs/README.md",
    "docs/INSTALLATION.md",
    "docs/ARCHITECTURE.md",
    "docs/ROADMAP.md",
    "docs/GITHUB-RELEASES.md",
    "docs/PACKAGES.md",
    "docs/RELEASE-VERIFICATION.md",
    "docs/CONTRIBUTING.md",
    "docs/PLATFORM-PARITY.md",
    "docs/VERSIONING.md",
    "docs/SECURITY.md",
    "docs/PRIVACY.md",
    "docs/SIGNING.md",
    "docs/LOCALIZATION.md",
    "docs/DEPENDENCIES.md",
    "docs/SETTINGS.md",
    "docs/TESTING.md",
    "docs/SUPPORT.md",
    "docs/REFERENCE-UI.md",
    "docs/NAVIGATION-BOOKMARKS.md",
    "docs/QUEUE-PRIORITY.md",
    "docs/THIRD-PARTY-NOTICES.md",
    "linux/README.md",
    "android/README.md",
    "android/UI-UX.md",
    "android/TRANSFER-PROGRESS.md",
    "macos/README.md",
    "macos/PARITY.md",
    "extensions/README.md",
    "extensions/PRIVACY.md",
    "scripts/README.md",
)
VISUAL_ASSETS = (
    "build/icon.png",
    "docs/images/0.0.6/ghost-ftp-main-workspace.png",
    "docs/images/0.0.6/ghost-ftp-site-manager.png",
    "docs/images/0.0.6/ghost-ftp-settings.png",
    "docs/images/0.0.6/ghost-ftp-about.png",
    "docs/images/0.0.6/ghost-ftp-linux-main-workspace.png",
    "docs/images/0.0.6/ghost-ftp-android-files.png",
)
RETIRED_LOCAL_REFERENCES = (
    "web/ftp",
    "web/README.md",
    "docs/WEB.md",
    "check_web_contract.py",
    "test_web_contract.py",
    ".github/workflows/web.yml",
)


def fail(message: str) -> None:
    raise SystemExit("DOCS_AUDIT_FAILED: " + message)


def read(relative: str) -> str:
    path = ROOT / relative
    if not path.is_file():
        fail(f"missing active document: {relative}")
    return path.read_text(encoding="utf-8")


def clean_destination(raw: str) -> str:
    value = raw.strip()
    if value.startswith("<") and ">" in value:
        value = value[1:value.index(">")]
    elif value:
        value = value.split(maxsplit=1)[0]
    return unquote(value).split("#", 1)[0].split("?", 1)[0].strip()


def check_link(source: Path, raw: str) -> None:
    dest = clean_destination(raw)
    if not dest or dest.lower().startswith(IGNORED_PREFIXES):
        return
    target = (source.parent / dest).resolve()
    try:
        target.relative_to(ROOT.resolve())
    except ValueError:
        fail(f"link escapes repository: {source.relative_to(ROOT)} -> {raw}")
    if not target.exists():
        fail(f"missing local link: {source.relative_to(ROOT)} -> {raw}")


def check_media(source: Path, raw: str) -> None:
    dest = clean_destination(raw)
    if not dest:
        fail(f"empty documentation media source: {source.relative_to(ROOT)}")
    if dest.lower().startswith(REMOTE_MEDIA_PREFIXES):
        fail(f"remote/data documentation media is blocked: {source.relative_to(ROOT)} -> {raw}")
    check_link(source, raw)


def main() -> int:
    version = read("VERSION").strip()
    if version != "0.0.6":
        fail(f"cleanup branch currently preserves published source VERSION 0.0.6, got {version!r}")

    for rel in ACTIVE_DOCS:
        read(rel)

    for path in sorted(x for x in ROOT.rglob("*.md") if ".git" not in x.parts):
        text = path.read_text(encoding="utf-8")
        for marker in RETIRED_LOCAL_REFERENCES:
            if marker in text:
                fail(f"retired web-project reference remains in {path.relative_to(ROOT)}: {marker}")
        for match in MARKDOWN_LINK_RE.finditer(text):
            check_link(path, match.group(1))
        for match in HTML_LINK_RE.finditer(text):
            check_link(path, match.group(1))
        for match in MARKDOWN_IMAGE_RE.finditer(text):
            check_media(path, match.group(1))
        for match in HTML_IMAGE_RE.finditer(text):
            check_media(path, match.group(1))

    for rel in VISUAL_ASSETS:
        path = ROOT / rel
        if not path.is_file() or path.stat().st_size <= 0:
            fail(f"missing maintained local documentation visual: {rel}")

    root = read("README.md")
    if not root.startswith("# Ghost FTP\n"):
        fail("README public title must be Ghost FTP")
    for marker in (
        "Latest published GitHub Release: `ghostftp-v0.0.6`",
        "13 platform artifacts / 16 public files",
        "Ghost-FTP-0.0.6-Android.apk",
        "Ghost-FTP-0.0.6-Opera-Extension.zip",
        "GHOSTFTP_ANDROID_CERT_SHA256",
        "Brendigo LTD",
        "repository-local",
        "exact-head",
    ):
        if marker not in root:
            fail(f"README missing marker: {marker}")

    index = read("docs/README.md")
    for marker in (
        "Latest published GitHub Release: **0.0.6**",
        "13 platform artifacts / 16 public files",
        "Chrome, Edge, Firefox and Opera",
        "proprietary commercial software",
        "../extensions/README.md",
    ):
        if marker not in index:
            fail(f"documentation index missing marker: {marker}")

    if (ROOT / "web").exists():
        fail("retired web directory still exists")
    if (ROOT / ".github/workflows/web.yml").exists():
        fail("retired web workflow still exists")
    if (ROOT / "docs/WEB.md").exists():
        fail("retired web documentation still exists")
    if (ROOT / "scripts/check_web_contract.py").exists() or (ROOT / "scripts/test_web_contract.py").exists():
        fail("retired web contract scripts still exist")

    print(f"DOCS_AUDIT=PASS ({version})")
    print("LATEST_PUBLISHED_GITHUB_RELEASE=0.0.6")
    print("PUBLIC_RELEASE_PLATFORMS=WINDOWS,LINUX,ANDROID,BROWSER_EXTENSIONS")
    print("ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID,MACOS,BROWSER_EXTENSIONS")
    print("RETIRED_WEB_PROJECT=ABSENT")
    return 0


if __name__ == "__main__":
    sys.exit(main())
