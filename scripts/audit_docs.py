#!/usr/bin/env python3
"""Validate active Ghost FTP documentation against the current release contract."""

from __future__ import annotations

import re
import sys
from pathlib import Path
from urllib.parse import unquote

ROOT = Path(__file__).resolve().parents[1]
DOCS = ROOT / "docs"
INDEX = DOCS / "README.md"
MARKDOWN_LINK_RE = re.compile(r"!?\[[^\]]*\]\(([^)\n]+)\)")
MARKDOWN_IMAGE_RE = re.compile(r"!\[[^\]]*\]\(([^)\n]+)\)")
HTML_LINK_RE = re.compile(r"\b(?:href|src)\s*=\s*[\"']([^\"']+)[\"']", re.IGNORECASE)
HTML_IMAGE_RE = re.compile(r"<img\b[^>]*\bsrc\s*=\s*[\"']([^\"']+)[\"']", re.IGNORECASE)
CURRENT_RELEASE_RE = re.compile(r"\*\*Current Ghost FTP release:\s*(\d+\.\d+\.\d+)\*\*")
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
    "ekstenzije/README.md",
    "ekstenzije/PRIVACY.md",
    "scripts/README.md",
)

CURRENT_VERSION_DOCS = (
    "README.md",
    "docs/README.md",
    "docs/INSTALLATION.md",
    "docs/GITHUB-RELEASES.md",
    "docs/PACKAGES.md",
    "docs/RELEASE-VERIFICATION.md",
    "docs/VERSIONING.md",
    "docs/SUPPORT.md",
    "docs/ARCHITECTURE.md",
    "docs/PLATFORM-PARITY.md",
    "docs/SIGNING.md",
    "docs/TESTING.md",
    "linux/README.md",
    "android/README.md",
)

VISUAL_ASSETS = (
    "build/icon.png",
    "docs/images/ghost-ftp-main-workspace.png",
    "docs/images/ghost-ftp-site-manager.png",
    "docs/images/ghost-ftp-settings.png",
    "docs/images/ghost-ftp-about.png",
)

STALE_RELEASE_SHAPES = (
    "12 platform artifacts / 15 public files",
    "9 platform artifacts / 12 public files",
    "6 platform artifacts / 9 public files",
)

STALE_SIGNING_MARKERS = (
    "production authenticode is optional",
    "official file is explicitly `unsigned`",
    "publication remains truthfully unsigned",
    "supports windows authenticode signing as an optional production hardening layer",
)


def fail(message: str) -> None:
    raise SystemExit("DOCS_AUDIT_FAILED: " + message)


def read(relative: str) -> str:
    path = ROOT / relative
    if not path.is_file():
        fail(f"missing active document: {relative}")
    return path.read_text(encoding="utf-8")


def require_markers(label: str, text: str, markers: tuple[str, ...]) -> None:
    for marker in markers:
        if marker not in text:
            fail(f"{label} missing marker: {marker}")


def clean_destination(raw: str) -> str:
    value = raw.strip()
    if value.startswith("<") and ">" in value:
        value = value[1:value.index(">")]
    elif value:
        value = value.split(maxsplit=1)[0]
    return unquote(value).split("#", 1)[0].split("?", 1)[0].strip()


def check_link(source: Path, raw: str) -> None:
    destination = clean_destination(raw)
    if not destination or destination.lower().startswith(IGNORED_PREFIXES):
        return
    target = (source.parent / destination).resolve()
    try:
        target.relative_to(ROOT.resolve())
    except ValueError:
        fail(f"link escapes repository: {source.relative_to(ROOT)} -> {raw}")
    if not target.exists():
        fail(f"missing local link: {source.relative_to(ROOT)} -> {raw}")


def check_media(source: Path, raw: str) -> None:
    destination = clean_destination(raw)
    if not destination:
        fail(f"empty documentation media source: {source.relative_to(ROOT)}")
    if destination.lower().startswith(REMOTE_MEDIA_PREFIXES):
        fail(f"remote/data documentation media is blocked: {source.relative_to(ROOT)} -> {raw}")
    check_link(source, raw)


def main() -> int:
    version = read("VERSION").strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        fail(f"invalid VERSION: {version!r}")
    if tuple(int(part) for part in version.split(".")) < (0, 0, 1):
        fail("documentation public version must be 0.0.1 or newer")

    if not INDEX.is_file():
        fail("documentation index is missing")
    for relative in ACTIVE_DOCS:
        read(relative)

    for path in sorted(path for path in ROOT.rglob("*.md") if ".git" not in path.parts):
        text = path.read_text(encoding="utf-8")
        for match in MARKDOWN_LINK_RE.finditer(text):
            check_link(path, match.group(1))
        for match in HTML_LINK_RE.finditer(text):
            check_link(path, match.group(1))
        for match in MARKDOWN_IMAGE_RE.finditer(text):
            check_media(path, match.group(1))
        for match in HTML_IMAGE_RE.finditer(text):
            check_media(path, match.group(1))
        for match in CURRENT_RELEASE_RE.finditer(text):
            if match.group(1) != version:
                fail(f"stale current-release marker in {path.relative_to(ROOT)}: {match.group(1)}")

    for relative in VISUAL_ASSETS:
        path = ROOT / relative
        if not path.is_file() or path.stat().st_size <= 0:
            fail(f"missing maintained local documentation visual: {relative}")

    for relative in ACTIVE_DOCS:
        lowered = read(relative).lower()
        for marker in STALE_RELEASE_SHAPES:
            if marker in lowered:
                fail(f"stale release shape in {relative}: {marker}")
        for marker in STALE_SIGNING_MARKERS:
            if marker in lowered:
                fail(f"stale signing policy in {relative}: {marker}")
        if "ghostftp web/" in lowered or "ios/" in lowered:
            fail(f"retired application surface appears in active guidance: {relative}")

    for relative in CURRENT_VERSION_DOCS:
        text = read(relative)
        if version not in text:
            fail(f"current release document does not mention VERSION {version}: {relative}")

    readme = read("README.md")
    index = read("docs/README.md")
    if not readme.startswith("# Ghost FTP\n"):
        fail("README public title must be Ghost FTP")
    if not index.startswith("# Ghost FTP documentation\n"):
        fail("documentation index title is invalid")

    common_release = (
        "18 platform artifacts / 21 public files",
        f"Ghost-FTP-{version}-Setup.exe",
        f"Ghost-FTP-{version}-Portable.exe",
        f"Ghost-FTP-{version}-Linux-Debian-amd64.deb",
        f"Ghost-FTP-{version}-Linux-Ubuntu-amd64.deb",
        f"Ghost-FTP-{version}-Linux-Fedora-x86_64.rpm",
        f"Ghost-FTP-{version}-Linux-Portable-amd64.tar.gz",
        f"Ghost-FTP-{version}-Android.apk",
    )
    require_markers("README release contract", readme, common_release)
    require_markers(
        "README browser contract",
        readme,
        (
            f"Ghost-FTP-{version}-Chrome-Extension.zip",
            f"Ghost-FTP-{version}-Edge-Extension.zip",
            f"Ghost-FTP-{version}-Firefox-Extension.zip",
            "no supported browser-to-desktop",
        ),
    )
    require_markers(
        "README identity",
        readme,
        (
            f"Current Ghost FTP version: **{version}**",
            "Development status: **Active**",
            "Release channel: **Current**",
            f"ghostftp-v{version}",
            "prerelease=false",
            f"ghcr.io/bren-wp/ghost-ftp:{version}",
            "Active native source platforms: **Windows, Linux, Android and macOS**",
            "24 selectable desktop languages",
            "repository-local",
            "exact-head",
        ),
    )
    require_markers(
        "documentation index",
        index,
        (
            f"**Current Ghost FTP release: {version}**",
            "Development status: **Active**",
            "Release channel: **Current**",
            "PRERELEASE=false",
            "latest release only",
            "18 platform artifacts / 21 public files",
            f"ghcr.io/bren-wp/ghost-ftp:{version}",
            "repository-local",
            "read-only verified cross-platform evidence bundle",
        ),
    )

    installation = read("docs/INSTALLATION.md")
    require_markers(
        "installation",
        installation,
        (
            f"Ghost FTP **{version}** is the current published release",
            "18 platform artifacts / 21 public files",
            f"Ghost-FTP-{version}-Android.apk",
            "Official Windows publication requires trusted Authenticode.",
            "WINDOWS_AUTHENTICODE=signed",
            "SFTP",
            "macOS development app",
        ),
    )

    releases = read("docs/GITHUB-RELEASES.md")
    require_markers(
        "GitHub release documentation",
        releases,
        (
            f"Ghost FTP **{version}** is the current published release contract",
            f"ghostftp-v{version}",
            "18 platform artifacts",
            "21 public files",
            f"Ghost-FTP-{version}-Android.apk",
            f"Ghost-FTP-{version}-Firefox-Extension.zip",
            "Prerelease: false",
            "release/ghostftp-vX.Y.Z",
            "does not publish a release directly",
        ),
    )

    verification = read("docs/RELEASE-VERIFICATION.md")
    require_markers(
        "release verification",
        verification,
        (
            f"current maintained release is **{version}**",
            f"## Published {version} release identity",
            f"VERSION={version}",
            f"TAG=ghostftp-v{version}",
            f"TITLE=Ghost FTP {version}",
            "CHANNEL=Current",
            "PRERELEASE=false",
            "PUBLIC_PLATFORM_ARTIFACTS=18",
            "PUBLIC_RELEASE_FILES=21",
            f"Ghost-FTP-{version}-Android.apk",
            "GHOSTFTP_ANDROID_SIGNER_SHA256",
            "release/ghostftp-vX.Y.Z",
            "A push to `main`, including a change to `VERSION`, must never publish a release directly.",
        ),
    )

    signing = read("docs/SIGNING.md")
    require_markers(
        "signing documentation",
        signing,
        (
            f"Ghost FTP **{version}**",
            "Official Windows publication is **signed-only**.",
            "GHOSTFTP_SIGNING_PFX_BASE64",
            "WINDOWS_AUTHENTICODE=signed",
            "GHOSTFTP_ANDROID_KEYSTORE_BASE64",
            "GHOSTFTP_ANDROID_SIGNER_SHA256",
            "production workflow never creates its own long-lived publisher key",
            "Developer ID Application",
            "macos/SIGN_AND_NOTARIZE.sh",
        ),
    )

    android = read("android/README.md")
    require_markers(
        "Android documentation",
        android,
        (
            f"Ghost FTP **{version}**",
            "production-signed public Android release",
            f"Ghost-FTP-{version}-Android.apk",
            "Ghost-FTP-Android-dev.apk",
            "FTP and explicit FTPS",
            "SFTP is intentionally not exposed",
            "Storage Access Framework",
        ),
    )

    browser = read("ekstenzije/README.md")
    require_markers(
        "browser helper documentation",
        browser,
        (
            "Chrome", "Microsoft Edge", "Firefox",
            "no supported browser-to-desktop",
        ),
    )

    macos = read("macos/README.md")
    require_markers(
        "macOS boundary",
        macos,
        (
            "Developer ID Application",
            "notarization",
            "development",
        ),
    )

    reference = read("docs/REFERENCE-UI.md")
    require_markers(
        "authentic UI evidence",
        reference,
        (
            "Windows — 5 images",
            "Linux — 3 images",
            "Android — 7 images",
            "exactly **15 runtime images**",
            "Mockups, image-generation output and manually composed approximations are not accepted",
        ),
    )

    print(f"DOCS_AUDIT=PASS ({version}; 18 platform artifacts / 21 public files)")
    print("PUBLIC_RELEASE_PLATFORMS=WINDOWS,LINUX,ANDROID,BROWSER_HELPER")
    print("ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID,MACOS")
    print("ANDROID_PUBLIC_RELEASE=PRODUCTION_SIGNED")
    print("ANDROID_SFTP=HIDDEN_UNTIL_STRICT_HOST_KEY_VERIFICATION")
    print("BROWSER_DESKTOP_HANDOFF=UNSUPPORTED")
    print("MACOS_PUBLIC_RELEASE=NO")
    return 0


if __name__ == "__main__":
    sys.exit(main())
