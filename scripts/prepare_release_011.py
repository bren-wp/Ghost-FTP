#!/usr/bin/env python3
from __future__ import annotations

from pathlib import Path
import json
import re

ROOT = Path(__file__).resolve().parents[1]
VERSION = "0.1.1"


def read(path: str) -> str:
    return (ROOT / path).read_text(encoding="utf-8")


def write(path: str, text: str) -> None:
    (ROOT / path).write_text(text, encoding="utf-8")


def replace(path: str, old: str, new: str, *, count: int | None = None) -> None:
    text = read(path)
    actual = text.count(old)
    if actual == 0:
        raise SystemExit(f"{path}: required text not found: {old[:100]!r}")
    if count is not None and actual != count:
        raise SystemExit(f"{path}: expected {count} matches, found {actual}: {old[:100]!r}")
    write(path, text.replace(old, new))


def regex_replace(path: str, pattern: str, repl: str, *, count: int = 0, min_count: int = 1, flags: int = 0) -> None:
    text = read(path)
    updated, n = re.subn(pattern, repl, text, count=count, flags=flags)
    if n < min_count:
        raise SystemExit(f"{path}: regex did not match: {pattern}")
    write(path, updated)


# Canonical version surfaces.
write("VERSION", VERSION + "\n")
write("GhostFTP WEB/VERSION", VERSION + "\n")
composer = json.loads(read("GhostFTP WEB/composer.json"))
composer["version"] = VERSION
write("GhostFTP WEB/composer.json", json.dumps(composer, indent=2, ensure_ascii=False) + "\n")
replace("GhostFTP WEB/service-worker.js", "ghostftp-static-v0.1.0", "ghostftp-static-v0.1.1", count=1)

# Root product documentation.
replace("README.md", "Current Ghost FTP version: **0.1.0**", "Current Ghost FTP version: **0.1.1**", count=1)
replace(
    "README.md",
    "- **Linux** — the same shared transfer/security engine, packaged for amd64, arm64 and i386, with a hardened terminal interface.",
    "- **Linux** — the same shared transfer/security engine, packaged for amd64, arm64 and i386, with a native dependency-free X11/XWayland GUI and hardened terminal fallback.",
    count=1,
)
replace(
    "README.md",
    "Explicit FTPS is supported with certificate validation enabled. Ghost FTP does not silently disable TLS certificate revocation checks.",
    "Explicit FTPS and implicit FTPS are supported with certificate validation enabled. Ghost FTP does not silently disable TLS certificate revocation checks.",
    count=1,
)
replace(
    "README.md",
    "\nImplicit FTPS is not part of the supported desktop protocol contract.\n",
    "\nImplicit FTPS uses the dedicated `ftpsi` transport mode and the conventional port 990 while retaining certificate validation.\n",
    count=1,
)
replace("README.md", "- native dark graphite/navy interface;", "- native near-black Ghost FTP interface aligned with the maintained Web brand tokens;", count=1)
replace(
    "README.md",
    "Windows and Linux share the functional core contract, but their presentation layers intentionally differ: Windows is a native Win32 GUI and Linux is a terminal UI. A pixel-identical cross-platform GUI would require introducing or maintaining an additional GUI/runtime dependency, which the current dependency policy deliberately avoids.",
    "Windows and Linux now both provide graphical desktop frontends over the same functional core. Windows uses native Win32; Linux uses a dependency-free raw X11/XWayland-compatible frontend and retains the hardened terminal interface for headless or explicit fallback use. Their native control implementations are not claimed to be pixel-identical, but they share the same Ghost FTP palette, action model and security boundaries without adding a third-party GUI framework.",
    count=1,
)
replace(
    "README.md",
    "Ghost-FTP-0.1.0-Setup-x64.exe\nGhost-FTP-0.1.0-Setup-x86.exe\nGhost-FTP-0.1.0-Portable-x64.exe\nGhost-FTP-0.1.0-Portable-x86.exe",
    "Ghost-FTP-0.1.1-Setup-x64.exe\nGhost-FTP-0.1.1-Setup-x86.exe\nGhost-FTP-0.1.1-Portable-x64.exe\nGhost-FTP-0.1.1-Portable-x86.exe",
    count=1,
)
regex_replace("README.md", r"- main workspace: `1080x700`, SHA-256 `[0-9a-f]{64}`;", "- main workspace: `1080x700`, SHA-256 recorded by the current authentic capture workflow;", count=1)
regex_replace("README.md", r"- Site Manager: `920x610`, SHA-256 `[0-9a-f]{64}`\.", "- Site Manager: `920x610`, SHA-256 recorded by the current authentic capture workflow.", count=1)

# Documentation index and active platform truth.
replace("docs/README.md", "**Current Ghost FTP release: 0.1.0**", "**Current Ghost FTP release: 0.1.1**", count=1)
replace(
    "docs/README.md",
    "- **Linux** — shared core with hardened terminal frontend and DEB packages for amd64/arm64/i386. Functional parity is substantially broader than visual parity; Linux is not currently described as pixel-identical to the Windows GUI.",
    "- **Linux** — shared core with a native dependency-free X11/XWayland graphical frontend, hardened terminal fallback and DEB packages for amd64/arm64/i386. Native Win32 and X11 rendering are not claimed to be pixel-identical.",
    count=1,
)
replace(
    "docs/README.md",
    "- Linux functional parity must not be mislabeled as pixel-identical GUI parity while the maintained Linux frontend remains terminal-based;",
    "- Linux graphical parity must not be mislabeled as pixel-identical Win32 parity; the native X11 frontend and terminal fallback must continue to use the shared Engine boundary;",
    count=1,
)

# Platform parity.
replacements = [
    (
        "Both editions use the same transfer, profile, settings, localization and security core. The presentation layer still differs: Windows is the native graphical reference frontend, while Linux currently uses a hardened terminal frontend.",
        "Both editions use the same transfer, profile, settings, localization and security core. Windows uses the native Win32 reference frontend; Linux now uses a dependency-free native X11/XWayland graphical frontend and retains the hardened terminal frontend for headless or explicit fallback use.",
    ),
    (
        "Linux exposes the same underlying profile/connection capabilities through terminal commands rather than reproducing the Windows dialog.",
        "Linux exposes the same underlying profile/connection capabilities through its graphical profile controls and through terminal fallback commands; neither path creates a second connection stack.",
    ),
    (
        "Linux users can navigate/list through the terminal command surface. A future Linux GUI search field must reuse the same item model and must not introduce a separate server/indexing service simply to mimic the Windows control.",
        "Linux users can navigate/list through the graphical file panel or terminal fallback. Any graphical search enhancement must reuse the same item model and must not introduce a separate server/indexing service.",
    ),
    (
        "Windows exposes queue actions as buttons and the transfer list. Linux exposes the same transfer-manager actions as commands:",
        "Windows exposes queue actions as buttons and the transfer list. Linux exposes the same transfer-manager actions in its GUI and preserves these terminal fallback commands:",
    ),
    (
        "Linux profile commands use the shared protected profile store:",
        "Linux graphical profile controls and terminal profile commands use the shared protected profile store:",
    ),
    (
        "Linux can inspect settings using `settings` and modify supported values with:",
        "Linux can edit the same validated settings through its graphical Settings overlay and can also inspect/modify them through the terminal fallback:",
    ),
    (
        "The canonical registry contains 24 languages and English is the default/fallback. Windows supports live language switching and localized Setup primary flows. Linux resolves the same stored locale, can change it at runtime and uses the same translation catalogs for terminal operations.",
        "The canonical registry contains 24 languages and English is the default/fallback. Windows supports live language switching and localized Setup primary flows. Linux resolves the same stored locale for its graphical and terminal surfaces and uses the same translation catalogs/fallback normalization.",
    ),
]
for old, new in replacements:
    replace("docs/PLATFORM-PARITY.md", old, new, count=1)

old_linux_visual = """### Linux

Linux is currently a terminal presentation over the same core. The prompt exposes both remote and local working directories, and command groups are separated into Remote, Local, Files, Queue, Profiles and Options.

This keeps the current package free of a bundled cross-platform GUI framework and allows amd64, arm64 and i386 package targets to remain small and auditable.

**Linux is not currently pixel-identical to the Windows graphical reference.** That gap is explicitly documented. A future Linux graphical frontend is acceptable only if it preserves the dependency, security and reproducibility requirements and does not fork the transfer/security engine.

If such a GUI requires an OS toolkit/display runtime, that prerequisite must be documented accurately; it must not be called “no dependency” merely because distributions commonly ship it.
"""
new_linux_visual = """### Linux

Linux now provides a real graphical desktop presentation over the same core. The frontend speaks directly to the local X11/XWayland display protocol using the Go standard library and the existing typed `api.Engine`; it does not bundle GTK, Qt, Electron, a webview, a tracking runtime or another transfer engine. The hardened terminal frontend remains available for headless systems or explicit `GHOSTFTP_UI=terminal` use.

The GUI exposes Quick Connect, SFTP host-key trust, profiles, local/server panes, file and tree transfers, transfer queue controls, local/remote file operations, remote permissions and validated transfer settings. Destructive actions retain the shared confirm-delete policy.

**Linux is not claimed to be pixel-identical to native Win32.** It follows the same Ghost FTP brand palette and interaction hierarchy while preserving native platform/runtime constraints and small auditable amd64, arm64 and i386 packages.

The Linux desktop requires a local X11-compatible display (native X11 or XWayland). Protocol transport prerequisites remain the system `curl` and OpenSSH tools documented elsewhere.
"""
replace("docs/PLATFORM-PARITY.md", old_linux_visual, new_linux_visual, count=1)

# Linux usage: GUI-first, terminal fallback.
linux_readme = read("linux/README.md")
marker = "Runtime dependencies declared by the package are `ca-certificates`, `curl` and `openssh-client`. Ghost FTP does not bundle those projects or add external Go modules to the desktop/core module.\n"
if marker not in linux_readme:
    raise SystemExit("linux/README.md: GUI insertion marker missing")
gui_section = """
## Graphical desktop

When a local `DISPLAY` is available, `ghostftp` starts the native Ghost FTP graphical frontend by default. The GUI is implemented directly against X11/XWayland-compatible display transport without GTK, Qt, Electron, a webview or an external Go GUI module.

The graphical workspace includes Quick Connect, FTP/FTPS/implicit-FTPS/SFTP selection, SFTP host-key trust, saved profiles, dual local/server file panes, single-file and tree transfers, queue controls, local/remote file operations, remote permissions and validated transfer settings.

For a headless session, or to explicitly use the hardened command interface, set:

```text
GHOSTFTP_UI=terminal ghostftp
```

A graphical session requires a local X11-compatible display (native X11 or XWayland). The file-transfer protocols continue to use the system transport prerequisites documented below.
"""
linux_readme = linux_readme.replace(marker, marker + gui_section, 1)
linux_readme = linux_readme.replace("## Remote file commands", "## Terminal fallback: remote file commands", 1)
linux_readme = linux_readme.replace("## Local file commands", "## Terminal fallback: local file commands", 1)
linux_readme = linux_readme.replace("The Linux terminal now exposes a real local working directory rather than resolving relative transfer paths from the process working directory:", "The terminal fallback exposes a real local working directory rather than resolving relative transfer paths from the process working directory:", 1)
write("linux/README.md", linux_readme)

# Installation examples and GUI behavior.
for old, new in [
    ("Ghost-FTP-0.1.0-Setup-x64.exe", "Ghost-FTP-0.1.1-Setup-x64.exe"),
    ("Ghost-FTP-0.1.0-Setup-x86.exe", "Ghost-FTP-0.1.1-Setup-x86.exe"),
    ("Ghost-FTP-0.1.0-Portable-x64.exe", "Ghost-FTP-0.1.1-Portable-x64.exe"),
    ("Ghost-FTP-0.1.0-Portable-x86.exe", "Ghost-FTP-0.1.1-Portable-x86.exe"),
]:
    replace("docs/INSTALLATION.md", old, new, count=1)
replace("docs/INSTALLATION.md", "For the current Beta baseline, replace `X.Y.Z` with `0.1.0`.", "For the current Beta release, replace `X.Y.Z` with `0.1.1`.", count=1)
replace("docs/INSTALLATION.md", "sudo apt install ./Ghost-FTP-0.1.0-Linux-amd64.deb", "sudo apt install ./Ghost-FTP-0.1.1-Linux-amd64.deb", count=1)
replace(
    "docs/INSTALLATION.md",
    "The package installs the `ghostftp` executable and Linux desktop/package metadata. The current Linux frontend is terminal-based but uses the same transfer/security engine as Windows.",
    "The package installs the `ghostftp` executable and Linux desktop/package metadata. With a local X11-compatible display it starts the native graphical frontend by default; headless systems can use the hardened terminal fallback. Both use the same transfer/security Engine as Windows.",
    count=1,
)

# Reference UI and canonical Web brand tokens.
replace(
    "docs/REFERENCE-UI.md",
    "The intended visual character is a dense professional desktop file manager using deep navy surfaces, cool blue borders, muted secondary text and a blue-violet primary accent. The UI remains native Win32 and does not load a web UI, tracking runtime or third-party GUI framework.",
    "The intended visual character is a dense professional desktop file manager using the maintained Ghost FTP near-black surfaces, cool neutral borders, muted secondary text and blue primary accent. These tokens mirror the maintained Web companion brand source while the UI remains native Win32 and does not load a web UI, tracking runtime or third-party GUI framework.",
    count=1,
)
old_palette = """| Window | `5, 17, 29` |
| Panel | `7, 25, 39` |
| List | `8, 28, 43` |
| Border | `28, 62, 86` |
| Primary text | `224, 237, 255` |
| Muted text | `126, 161, 201` |
| Accent | `96, 126, 255` |
| Strong accent | `110, 84, 255` |
| Success | `57, 216, 166` |
| Warning | `247, 190, 72` |"""
new_palette = """| Window | `8, 10, 15` (`#080A0F`) |
| Panel | `15, 19, 28` (`#0F131C`) |
| List | `21, 26, 37` (`#151A25`) |
| Border | `37, 45, 60` (`#252D3C`) |
| Primary text | `244, 247, 255` (`#F4F7FF`) |
| Muted text | `142, 153, 173` (`#8E99AD`) |
| Accent | `82, 119, 245` (`#5277F5`) |
| Strong accent | `114, 147, 255` (`#7293FF`) |
| Success | `74, 215, 155` (`#4AD79B`) |
| Warning | `242, 186, 85` (`#F2BA55`) |"""
replace("docs/REFERENCE-UI.md", old_palette, new_palette, count=1)
regex_replace(
    "docs/REFERENCE-UI.md",
    r"## Linux presentation boundary\n\nLinux currently uses the same transfer/security/settings/profile engine with a hardened terminal frontend\..*?reviewed under the dependency policy\.\n",
    """## Linux presentation boundary

Linux now uses the same transfer/security/settings/profile engine with a real dependency-free X11/XWayland graphical frontend. It remains **native-platform different** from Win32 rather than pretending to be pixel-identical.

The Linux GUI was accepted only after satisfying the same boundary rules: no fork of the connection/transfer/security engine, no telemetry, no hidden web/service runtime, reproducible amd64/arm64/i386 packaging, shared destructive-operation safeguards and a real production-binary X11 runtime smoke test. The hardened terminal frontend remains available for headless or explicit fallback use.

No cross-platform GUI toolkit is bundled. A local X11-compatible display is the graphical runtime prerequisite; FTP/FTPS and SFTP continue to use the documented system `curl` and OpenSSH prerequisites.
""",
    count=1,
    flags=re.S,
)

# Other active docs: remove only stale current-presentation phrases.
for path in ["docs/ARCHITECTURE.md", "docs/ROADMAP.md", "docs/TESTING.md", "docs/DEPENDENCIES.md", "docs/SECURITY.md"]:
    text = read(path)
    text = text.replace("hardened terminal frontend", "native X11 GUI with hardened terminal fallback")
    text = text.replace("hardened terminal interface", "native X11 GUI with hardened terminal fallback")
    text = text.replace("Linux terminal frontend", "Linux native X11 GUI and terminal fallback")
    text = text.replace("Linux terminal interface", "Linux native X11 GUI and terminal fallback")
    write(path, text)

# Changelog and release history.
changelog = read("CHANGELOG.md")
if "## 0.1.1" not in changelog:
    heading_end = changelog.find("\n", changelog.find("#")) + 1
    entry = """
## 0.1.1 - 2026-09-06 Beta

### Added

- Native dependency-free Linux X11/XWayland graphical desktop with Quick Connect, SFTP host trust, profiles, dual file panes, file/tree transfer actions and transfer queue controls.
- Linux graphical local/remote New folder, Rename and Delete actions plus remote Permissions/chmod.
- Linux graphical transfer Settings overlay for parallelism, conflict policy, retry count/delay, connection timeout and delete confirmation.
- Production X11 runtime smoke coverage that verifies protocol setup, a real mapped Ghost FTP window and stable process lifetime.

### Changed

- Desktop palette now mirrors the maintained Ghost FTP Web brand tokens (`#080A0F`, `#0F131C`, `#151A25`, `#5277F5`, `#7293FF`) while retaining native Win32/X11 rendering.
- Linux desktop packaging launches the graphical application by default when a display is present; the hardened terminal remains an explicit/headless fallback.
- Documentation, package examples and authentic Windows runtime screenshots are refreshed for the 0.1.1 Beta line.
- Implicit FTPS (`ftpsi`, conventional port 990) is documented as a maintained desktop protocol option.

### Fixed / hardened

- Corrected raw X11 CreateGC value-mask handling and GetKeyboardMapping wire offsets found by real runtime testing.
- Destructive Linux GUI actions honor the canonical confirm-delete setting and retain typed Engine validation.
- No Linux password/passphrase persistence was introduced; existing secret-lifetime boundaries remain unchanged.

"""
    write("CHANGELOG.md", changelog[:heading_end] + entry + changelog[heading_end:])

history = read("docs/RELEASE-HISTORY.md")
if "## 0.1.1" not in history:
    pos = history.find("\n") + 1
    entry = """
## 0.1.1 — 2026-09-06 Beta

Ghost FTP 0.1.1 advances the pre-1.0 Windows/Linux line with a real Linux graphical frontend, stronger runtime verification and a unified Ghost FTP visual system. Windows remains native Win32; Linux uses a dependency-free raw X11/XWayland-compatible frontend over the same typed Engine and retains the terminal for headless/explicit fallback.

The release also refreshes native Windows runtime screenshots, documents implicit FTPS, aligns desktop palette tokens with the maintained Web brand source, and preserves the existing no-telemetry, SFTP host-trust, credential-lifetime, filesystem and release-readback protections.

"""
    write("docs/RELEASE-HISTORY.md", history[:pos] + entry + history[pos:])

# Release workflow controlled branch-create trigger. Build the expression without
# a literal ${{...}} in this helper so an Actions caller never pre-expands it.
release = read(".github/workflows/release.yml")
if "\n  create:\n" not in release:
    release = release.replace("        type: string\n\npermissions:", "        type: string\n  create:\n\npermissions:", 1)
gate = "$" + "{{ github.event_name == 'workflow_dispatch' || (github.event_name == 'create' && github.event.ref_type == 'branch' && startsWith(github.event.ref, 'release/ghostftp-v')) }}"
for job in ["quality", "windows", "linux"]:
    needle = f"  {job}:\n    name:"
    if needle not in release:
        raise SystemExit(f"release workflow job marker missing: {job}")
    release = release.replace(needle, f"  {job}:\n    if: {gate}\n    name:", 1)
needle = "  publish:\n    name:"
if needle not in release:
    raise SystemExit("release workflow publish marker missing")
release = release.replace(needle, f"  publish:\n    if: {gate}\n    name:", 1)
write(".github/workflows/release.yml", release)

ghrel = read("docs/GITHUB-RELEASES.md")
if "release/ghostftp-vX.Y.Z" not in ghrel:
    ghrel += """
## Automated production trigger

After the release candidate is merged and the exact `main` commit has passed all required gates, maintainers may create `release/ghostftp-vX.Y.Z` at that exact `main` SHA. The release workflow accepts only that branch namespace (or an explicit manual dispatch), re-runs the full production gates, rechecks that `main` still equals the release commit, and then creates the immutable `ghostftp-vX.Y.Z` release tag.
"""
    write("docs/GITHUB-RELEASES.md", ghrel)

print("RELEASE_PREP_TRANSFORMS=PASS")
