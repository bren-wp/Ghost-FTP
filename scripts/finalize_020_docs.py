#!/usr/bin/env python3
from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def read(rel: str) -> str:
    return (ROOT / rel).read_text(encoding="utf-8")


def write(rel: str, text: str) -> None:
    (ROOT / rel).write_text(text, encoding="utf-8", newline="\n")


def replace_once(rel: str, old: str, new: str) -> None:
    text = read(rel)
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"expected one match in {rel}, found {count}: {old[:100]!r}")
    write(rel, text.replace(old, new, 1))


def remove_section(rel: str, heading: str) -> None:
    text = read(rel)
    pattern = re.compile(rf"\n## {re.escape(heading)}\n.*?(?=\n## |\Z)", re.S)
    matches = list(pattern.finditer(text))
    if len(matches) != 1:
        raise SystemExit(f"expected one section {heading!r} in {rel}, found {len(matches)}")
    write(rel, text[:matches[0].start()] + "\n" + text[matches[0].end():])


# Architecture: one shared core, native Windows/Linux frontends, no retired companion boundary.
replace_once(
    "docs/ARCHITECTURE.md",
    "The active desktop development baseline is **0.1.0 Beta**. Version maturity does not change the architecture contract: all `0.x.y` builds use the same maintained Windows/Linux core and move toward the first stable **1.0.0** release.\n\nThe repository also retains the Ghost FTP Web companion as a separate shared-hosting/PWA source surface. The Web companion is not part of the Windows/Linux desktop application artifact contract.\n",
    "The current desktop development baseline is **0.2.0 Beta**. Version maturity does not change the architecture contract: all `0.x.y` builds use the same maintained Windows/Linux core and move toward the first stable **1.0.0** release.\n",
)
replace_once(
    "docs/ARCHITECTURE.md",
    "- Linux uses the terminal frontend in `other.go` with a `linux` build tag.\n",
    "- Linux uses the dependency-free native X11/XWayland-compatible graphical frontend and retains the hardened terminal frontend for headless or explicit fallback use.\n",
)
replace_once(
    "docs/ARCHITECTURE.md",
    "Windows buttons and Linux terminal queue commands invoke this same manager.\n",
    "Windows and Linux graphical/terminal queue actions invoke this same manager.\n",
)
remove_section("docs/ARCHITECTURE.md", "Web companion boundary")

# Roadmap: 0.2.0 is the active quality milestone; native Linux GUI already exists.
replace_once(
    "docs/ROADMAP.md",
    "Ghost FTP is currently on the **0.x Beta** development line, beginning at **0.1.0 Beta**. Product development remains focused on **Windows and Linux**, with reliability, security, protocol correctness, parity and professional usability taking priority over expanding the number of application platforms.\n",
    "Ghost FTP is currently on **0.2.0 Beta**, within the `0.x` development line that began at 0.1.0. Product development remains focused on **Windows and Linux**, with reliability, security, protocol correctness, parity and professional usability taking priority over expanding the number of application platforms.\n",
)
replace_once(
    "docs/ROADMAP.md",
    "- consider a native/lightweight Linux graphical frontend only if it meets dependency, security, reproducibility and maintenance requirements without forking protocol logic.\n",
    "- continue refining the existing dependency-free native Linux graphical frontend without forking protocol logic or weakening the terminal fallback.\n",
)
remove_section("docs/ROADMAP.md", "Web companion")

# Contributing: only the two maintained desktop frontends belong to active source.
replace_once(
    "docs/CONTRIBUTING.md",
    "- Linux terminal/platform behavior belongs in Linux build-tagged files and `linux/` packaging;\n- the separate Web companion remains under `GhostFTP WEB/` and must not be used as a hidden replacement for the Windows or Linux desktop frontend.\n",
    "- Linux graphical/terminal platform behavior belongs in Linux build-tagged files and `linux/` packaging.\n",
)

# Platform parity: replace the removed reference-shell description with the canonical workspace contract.
replace_once(
    "docs/PLATFORM-PARITY.md",
    "The active maturity baseline is **0.1.0 Beta**. Functional parity work completed before the version reset is preserved; changing the active version line does not remove or downgrade existing capabilities.\n",
    "The active maturity baseline is **0.2.0 Beta**. Functional parity work completed in earlier Beta milestones is preserved unless a duplicated or presentation-only surface is intentionally removed in favor of the canonical shared workflow.\n",
)
replace_once(
    "docs/PLATFORM-PARITY.md",
    "## Windows reference-shell boundary\n\nWindows Setup and Windows Portable package the same application executable/source. Once the app starts, both expose the same:\n\n- deep navy reference shell;\n- left profile/navigation sidebar;\n- menu and top action toolbar;\n- Connection Log and Quick Connect cards;\n- Local/Remote file cards;\n- remote search;\n- transfer queue;\n- live localization;\n- command/action-state validation;\n- Site Manager and settings surfaces.\n\nThe graphical shell is presentation only. Toolbar/menu actions route to the same canonical connection, file-operation and transfer code paths used elsewhere in the application.\n",
    "## Windows desktop boundary\n\nWindows Setup and Windows Portable package the same application executable/source. Once the app starts, both expose the same canonical desktop workflow:\n\n- brand, connection state and language header;\n- saved Sites/profile controls;\n- Quick Connect;\n- balanced Local/Remote file panes;\n- local and remote file-operation controls;\n- transfer queue and queue actions;\n- live localization;\n- command/action-state validation;\n- Site Manager and Settings surfaces.\n\nThe Win32 layer is presentation/input orchestration only. Menu/button actions route to the same canonical connection, file-operation and transfer code paths used by the shared engine.\n",
)
remove_section("docs/PLATFORM-PARITY.md", "Remote-search parity boundary")
replace_once(
    "docs/PLATFORM-PARITY.md",
    "- persistent left sidebar;\n- Connection Log and Quick Connect cards;\n- balanced local/server dual-pane workspace;\n- remote in-memory search;\n",
    "- compact brand/state/profile controls;\n- Quick Connect;\n- balanced local/server dual-pane workspace;\n",
)
replace_once(
    "docs/PLATFORM-PARITY.md",
    "The existing Web companion remains in the repository as a separate source surface. It is not counted as a Windows/Linux desktop application platform artifact in current releases.\n",
    "Only Windows and Linux are part of the maintained application source/build/release matrix.\n",
)
replace_once(
    "docs/PLATFORM-PARITY.md",
    "Windows/reference regressions verify:\n\n- canonical sidebar/toolbar/cards remain wired;\n- toolbar actions mirror canonical action state;\n- remote search retains a full unfiltered model;\n- local file columns remain four-column reference order;\n- remote file columns remain five-column reference order with Permissions;\n",
    "Windows regressions verify:\n\n- the canonical dual-pane workspace remains wired across resize and DPI changes;\n- visible actions mirror canonical action state instead of bypassing Engine validation;\n- local file columns remain four-column order;\n- remote file columns remain five-column order with Permissions;\n",
)

# GitHub Releases: one GitHub Release publication surface, current 0.2.0, no package registry.
gh = read("docs/GITHUB-RELEASES.md")
gh = gh.replace("Current source version: **0.1.1**.", "Current source version: **0.2.0**.")
gh = gh.replace("Ghost FTP 0.1.1 Beta", "Ghost FTP 0.2.0 Beta")
gh = gh.replace("Ghost-FTP-0.1.1-", "Ghost-FTP-0.2.0-")
gh = gh.replace("the active release prepared here is **0.1.1 Beta**", "the active release prepared here is **0.2.0 Beta**")
gh = gh.replace("The existing Web companion source is maintained separately and is not counted as a desktop/platform artifact in this release contract.\n\n", "")
gh = gh.replace("- Web companion source/runtime audit.\n", "- retired-surface/desktop-source audit.\n")
write("docs/GITHUB-RELEASES.md", gh)
remove_section("docs/GITHUB-RELEASES.md", "NuGet/GitHub Package")

# Versioning: retain historical 0.1.0 milestone, identify the current 0.2.0 baseline and remove retired package-registry wording.
replace_once(
    "docs/VERSIONING.md",
    "The active development line starts at **0.1.0**.\n",
    "The active development line began at **0.1.0** and the current source baseline is **0.2.0 Beta**.\n",
)
replace_once(
    "docs/VERSIONING.md",
    "- NuGet/GitHub Package versions;\n",
    "",
)
replace_once(
    "docs/VERSIONING.md",
    "For the current baseline this means:\n\n```text\nGhost-FTP-0.1.0-Setup-x64.exe\nGhost-FTP-0.1.0-Setup-x86.exe\nGhost-FTP-0.1.0-Portable-x64.exe\nGhost-FTP-0.1.0-Portable-x86.exe\n```",
    "For the current baseline this means:\n\n```text\nGhost-FTP-0.2.0-Setup-x64.exe\nGhost-FTP-0.2.0-Setup-x86.exe\nGhost-FTP-0.2.0-Portable-x64.exe\nGhost-FTP-0.2.0-Portable-x86.exe\n```",
)

# Release verification: current examples must describe the release being prepared.
rv = read("docs/RELEASE-VERIFICATION.md")
rv = rv.replace("The active baseline is **0.1.0 Beta**.", "The active baseline is **0.2.0 Beta**.")
rv = rv.replace("Ghost-FTP-0.1.0-", "Ghost-FTP-0.2.0-")
rv = rv.replace("Version: 0.1.0", "Version: 0.2.0")
rv = rv.replace("`0.1.0 Beta` or later current Windows/Linux releases", "the current 0.2.0 Beta Windows/Linux release")
write("docs/RELEASE-VERIFICATION.md", rv)

# Active guidance must not contain the retired-source markers enforced by audit_docs.py.
active = (
    "README.md",
    "docs/README.md",
    "docs/INSTALLATION.md",
    "docs/ARCHITECTURE.md",
    "docs/ROADMAP.md",
    "docs/GITHUB-RELEASES.md",
    "docs/RELEASE-VERIFICATION.md",
    "docs/CONTRIBUTING.md",
    "docs/PLATFORM-PARITY.md",
    "docs/VERSIONING.md",
)
for rel in active:
    lowered = read(rel).lower()
    for marker in ("android/", "ios/", "macos/", "ghostftp web/", "web companion", "pwa"):
        if marker in lowered:
            raise SystemExit(f"retired active-doc marker remains in {rel}: {marker}")

# Current release guidance must no longer advertise NuGet/package-registry publication.
for rel in ("docs/GITHUB-RELEASES.md", "docs/VERSIONING.md", "docs/RELEASE-VERIFICATION.md"):
    lowered = read(rel).lower()
    for marker in ("nuget", "github package"):
        if marker in lowered:
            raise SystemExit(f"retired publication marker remains in {rel}: {marker}")

# Remove this one-shot migration before the enclosing release cleanup stages the tree.
Path(__file__).unlink()
