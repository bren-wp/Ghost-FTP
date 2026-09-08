#!/usr/bin/env python3
"""Validate Ghost FTP active documentation against the current product/release contract."""

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
CURRENT_RELEASE_RE = re.compile(r"\*\*Current Ghost FTP release:\s*(\d+\.\d+\.\d+)\*\*")
IGNORED_PREFIXES = ("http://", "https://", "mailto:", "data:", "//", "#")
RETIRED_ACTIVE_MARKERS = ("android/", "ios/", "macos/", "ghostftp web/", "web companion", "pwa")
STALE_SIGNING_POLICY_MARKERS = (
    "trusted authenticode requirement for stable windows publication",
    "a stable windows release is blocked unless",
    "stable release whose windows signing state is not trusted/configured",
    "stable windows authenticode gate",
    "stable windows signing gate",
    "stable windows publication requires",
    "stable windows publication additionally requires",
)
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
    "linux/README.md",
    "scripts/README.md",
)


def fail(message: str) -> None:
    raise SystemExit("DOCS_AUDIT_FAILED: " + message)


def clean_destination(raw: str) -> str:
    value = raw.strip()
    if value.startswith("<") and ">" in value:
        value = value[1:value.index(">")]
    elif value:
        value = value.split(maxsplit=1)[0]
    value = unquote(value).split("#", 1)[0].split("?", 1)[0]
    return value.strip()


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


def require_markers(label: str, text: str, markers: tuple[str, ...]) -> None:
    for marker in markers:
        if marker not in text:
            fail(f"{label} missing marker: {marker}")


def main() -> int:
    version_path = ROOT / "VERSION"
    if not version_path.is_file():
        fail("VERSION is missing")
    version = version_path.read_text(encoding="utf-8").strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        fail(f"invalid VERSION: {version!r}")
    major = int(version.split(".", 1)[0])

    files = sorted(path for path in ROOT.rglob("*.md") if ".git" not in path.parts)
    if not files or not INDEX.is_file():
        fail("documentation set is incomplete")

    for path in files:
        text = path.read_text(encoding="utf-8")
        for match in MARKDOWN_LINK_RE.finditer(text):
            check_link(path, match.group(1))
        for match in HTML_LINK_RE.finditer(text):
            check_link(path, match.group(1))
        for match in CURRENT_RELEASE_RE.finditer(text):
            if match.group(1) != version:
                fail(f"stale release marker in {path.relative_to(ROOT)}: {match.group(1)}")

    readme = (ROOT / "README.md").read_text(encoding="utf-8")
    index = INDEX.read_text(encoding="utf-8")
    if not readme.startswith("# Ghost FTP\n"):
        fail("README public title must be Ghost FTP")
    if not index.startswith("# Ghost FTP documentation\n"):
        fail("documentation index title is invalid")
    if f"Current Ghost FTP version: **{version}**" not in readme:
        fail("README VERSION marker is stale")
    if f"**Current Ghost FTP release: {version}**" not in index:
        fail("docs/README release marker is stale")

    if major == 0:
        for rel, text in (("README.md", readme), ("docs/README.md", index)):
            if "Development status: **Beta**" not in text:
                fail(f"{rel} must mark pre-1.0 releases as Beta")
    else:
        for rel, text in (("README.md", readme), ("docs/README.md", index)):
            if "Development status: **Stable**" not in text:
                fail(f"{rel} must mark 1.x+ releases as Stable")
        if "prerelease=false" not in index:
            fail("stable documentation index must state prerelease=false")

    for rel in ACTIVE_DOCS:
        path = ROOT / rel
        if not path.is_file():
            fail(f"missing active document: {rel}")
        text = path.read_text(encoding="utf-8")
        lowered = text.lower()
        for marker in RETIRED_ACTIVE_MARKERS:
            if marker in lowered:
                fail(f"retired application surface appears in active guidance: {rel} -> {marker}")
        for marker in STALE_SIGNING_POLICY_MARKERS:
            if marker in lowered:
                fail(f"stale mandatory-signing policy appears in active guidance: {rel} -> {marker}")

    require_markers("README", readme, ("Windows", "Linux", "24", "FTP", "FTPS", "SFTP"))

    # Published 1.1.6 remains the current stable release contract. These values
    # intentionally do not become the future canonical 12/15 release shape.
    for marker in ("9 platform artifacts", "12 public files"):
        if marker not in readme or marker not in index:
            fail(f"release contract marker missing: {marker}")

    # Source/CI documentation must clearly distinguish historical publication,
    # the canonical next-release workflow and supplemental distro CI artifacts.
    require_markers(
        "README Linux distro contract",
        readme,
        (
            "supplemental distro-specific CI packages",
            "Debian 13 amd64",
            "Ubuntu 26.04 LTS amd64",
            "Fedora 44 x86_64",
            "not yet part of the canonical release allow-list",
            "12 platform artifacts / 15 public files",
            "linux/BUILD-DISTROS.sh",
        ),
    )

    for required_path in (
        ROOT / "linux" / "BUILD-DISTROS.sh",
        ROOT / ".github" / "workflows" / "linux-distro-packages.yml",
        ROOT / ".github" / "workflows" / "linux-distro-install.yml",
    ):
        if not required_path.is_file():
            fail(f"documented Linux distro contract implementation missing: {required_path.relative_to(ROOT)}")

    linux_readme = (ROOT / "linux" / "README.md").read_text(encoding="utf-8")
    require_markers(
        "linux/README distro contract",
        linux_readme,
        (
            "linux/BUILD-DISTROS.sh",
            "Linux-Debian-amd64.deb",
            "Linux-Ubuntu-amd64.deb",
            "Linux-Fedora-x86_64.rpm",
            "Linux-Portable-amd64.tar.gz",
            "`amd64` | `x86_64`",
            "`arm64` | `aarch64`",
            "`i386` | `i686`",
            ".github/workflows/linux-distro-packages.yml",
            ".github/workflows/linux-distro-install.yml",
            "Debian 13 amd64",
            "Ubuntu 26.04 LTS amd64",
            "Fedora 44 x86_64",
            "x86-64 only",
            "current canonical release workflow does not yet publish them as release assets",
            "12 platform artifacts / 15 public files",
        ),
    )
    for stale in (
        "does not pretend to be an RPM",
        "unverified RPM",
        "an unverified RPM/AppImage/Flatpak/Snap lifecycle",
    ):
        if stale in linux_readme:
            fail(f"linux/README contains stale distro verification claim: {stale}")

    installation = (DOCS / "INSTALLATION.md").read_text(encoding="utf-8")
    require_markers(
        "installation distro contract",
        installation,
        (
            "Published 1.1.6 packages",
            "Canonical next-release source packages",
            "Supplemental distro-specific source/CI packages",
            "Linux-Debian-amd64.deb",
            "Linux-Ubuntu-amd64.deb",
            "Linux-Fedora-x86_64.rpm",
            "Debian 13 amd64",
            "Ubuntu 26.04 LTS amd64",
            "Fedora 44 x86_64",
            "not yet part of the canonical release allow-list",
            "x86-64 only",
        ),
    )

    parity = (DOCS / "PLATFORM-PARITY.md").read_text(encoding="utf-8")
    require_markers(
        "platform parity documentation",
        parity,
        (
            "Windows and Linux platform parity",
            "SFTP password",
            "SFTP key passphrase",
            "24-language",
            "same typed `internal/api.Engine`",
            "Production Authenticode is optional.",
            "WINDOWS_AUTHENTICODE=unsigned",
            "linux/BUILD-DISTROS.sh",
            ".github/workflows/linux-distro-packages.yml",
            ".github/workflows/linux-distro-install.yml",
            "Debian 13 amd64",
            "Ubuntu 26.04 LTS amd64",
            "Fedora 44 x86_64",
            "x86-64 only",
            "byte-identical compatibility alias of the verified x86 Setup artifact",
            "not yet part of the canonical release allow-list",
            "12 platform artifacts / 15 public files",
        ),
    )

    testing = (DOCS / "TESTING.md").read_text(encoding="utf-8")
    require_markers(
        "testing documentation",
        testing,
        (
            f"Ghost FTP **{version} Stable**",
            ".github/workflows/linux-distro-packages.yml",
            ".github/workflows/linux-distro-install.yml",
            "linux/BUILD-DISTROS.sh",
            "Debian 13 amd64",
            "Ubuntu 26.04 LTS amd64",
            "Fedora 44 x86_64",
            "Native package-manager/runtime coverage is deliberately limited to x86-64.",
            "9 platform artifacts / 12 public files",
            "12 platform artifacts / 15 public files",
            "Exact-head and post-merge rule",
        ),
    )
    for stale_version in ("Ghost FTP 1.1.1", "**1.1.1 Stable**"):
        if stale_version in testing:
            fail(f"testing documentation contains stale current-version claim: {stale_version}")

    versioning = (DOCS / "VERSIONING.md").read_text(encoding="utf-8")
    require_markers(
        "versioning documentation",
        versioning,
        (
            "0.1.0",
            "0.x.y",
            "Beta",
            "Stable",
            "1.0.0",
            "Setup",
            "Portable",
            "optional production hardening layer",
            "WINDOWS_AUTHENTICODE=unsigned",
            "Absence of a production Authenticode certificate by itself is not a versioning failure.",
        ),
    )

    packages = (DOCS / "PACKAGES.md").read_text(encoding="utf-8")
    require_markers(
        "packages documentation",
        packages,
        (
            "ghcr.io/bren-wp/ghost-ftp",
            "distribution bundle",
            "not a runtime container",
            "/ghostftp-release/",
            "SHA256.txt",
            "Authenticode verification **when a trusted production certificate is configured**",
            "WINDOWS_AUTHENTICODE=unsigned",
        ),
    )

    contributing = (DOCS / "CONTRIBUTING.md").read_text(encoding="utf-8")
    require_markers(
        "contributing documentation",
        contributing,
        (
            "truthful Windows signing state",
            "WINDOWS_AUTHENTICODE=unsigned",
            "A missing production code-signing certificate alone is not a release failure.",
        ),
    )

    roadmap = (DOCS / "ROADMAP.md").read_text(encoding="utf-8")
    require_markers(
        "roadmap documentation",
        roadmap,
        (
            "truthful Windows signing-state metadata",
            "WINDOWS_AUTHENTICODE=unsigned",
            "no generated/self-signed production identity represented as a trusted publisher",
        ),
    )

    support = (DOCS / "SUPPORT.md").read_text(encoding="utf-8")
    require_markers(
        "support documentation",
        support,
        (
            "inspect `WINDOWS_AUTHENTICODE` in `BUILD-METADATA.txt`",
            "official file is explicitly `unsigned`",
            "if metadata says `signed` and Windows signature verification fails",
        ),
    )

    scripts_readme = (ROOT / "scripts/README.md").read_text(encoding="utf-8")
    require_markers(
        "scripts README",
        scripts_readme,
        (
            "Windows production signing is optional.",
            "WINDOWS_AUTHENTICODE=unsigned",
            "self-signed development certificate must never be substituted",
        ),
    )

    signing = (DOCS / "SIGNING.md").read_text(encoding="utf-8")
    require_markers(
        "signing documentation",
        signing,
        (
            "supports Windows Authenticode signing as an optional production hardening layer",
            "WINDOWS_AUTHENTICODE=signed",
            "WINDOWS_AUTHENTICODE=unsigned",
            "production workflow never creates its own long-lived publisher key",
        ),
    )

    release_verification = (DOCS / "RELEASE-VERIFICATION.md").read_text(encoding="utf-8")
    require_markers(
        "release verification documentation",
        release_verification,
        (
            "truthful supported publication state",
            "does not create a self-signed production identity",
            "explicit unsigned metadata when no production certificate is configured",
        ),
    )

    print(f"DOCS_AUDIT=PASS ({version}; {len(files)} Markdown files)")
    print("PUBLIC_BRAND=Ghost FTP")
    print("ACTIVE_APPLICATION_PLATFORMS=WINDOWS,LINUX")
    print("PRE_1_0_CHANNEL=BETA")
    print("FIRST_STABLE_VERSION=1.0.0")
    print("STABLE_GITHUB_RELEASE_PRERELEASE=FALSE")
    print("STABLE_WINDOWS_RELEASE_REQUIRES_TRUSTED_AUTHENTICODE=NO")
    print("TRUSTED_AUTHENTICODE_WHEN_CONFIGURED=VERIFIED")
    print("SELF_SIGNED_PRODUCTION_IDENTITY=BLOCKED")
    print("PUBLIC_PLATFORM_ARTIFACTS=9")
    print("PUBLIC_RELEASE_FILES=12")
    print("NEXT_RELEASE_PLATFORM_ARTIFACTS=12")
    print("NEXT_RELEASE_PUBLIC_FILES=15")
    print("SUPPLEMENTAL_DISTRO_PACKAGING=DEBIAN,UBUNTU,FEDORA,PORTABLE")
    print("NATIVE_DISTRO_INSTALL_COVERAGE=DEBIAN13_AMD64,UBUNTU26.04_AMD64,FEDORA44_X86_64")
    print("SUPPLEMENTAL_DISTRO_RELEASE_ASSETS=NO")
    print("STABLE_GITHUB_PACKAGE=ghcr.io/bren-wp/ghost-ftp")
    return 0


if __name__ == "__main__":
    sys.exit(main())
