#!/usr/bin/env python3
from __future__ import annotations

import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def read(rel: str) -> str:
    return (ROOT / rel).read_text(encoding="utf-8")


def write(rel: str, text: str) -> None:
    path = ROOT / rel
    path.write_text(text, encoding="utf-8", newline="\n")


def replace_once(rel: str, old: str, new: str) -> None:
    text = read(rel)
    if text.count(old) != 1:
        raise SystemExit(f"expected one match in {rel}, found {text.count(old)}")
    write(rel, text.replace(old, new, 1))


# Make uninstall registration part of the install transaction rather than dead code.
replace_once(
    "cmd/installer/main.go",
    '''\tif err := platform.DeleteRegistryKey(uninstallKey); err != nil {\n\t\twarnings = append(warnings, "The legacy Windows uninstall registry entry could not be removed.")\n\t}\n\n''',
    "",
)

replace_once(
    "cmd/installer/main.go",
    '''\tif err := register(appPath); err != nil {\n\t\textra := rollbackMessage()\n\t\tshowInstallError(\n\t\t\t"Setup did not finish",\n\t\t\t"Windows could not finish registering the application. Try again."+extra,\n\t\t)\n\t\treturn 1\n\t}\n\n\ttransactionCommitted = true\n''',
    '''\tif err := register(appPath); err != nil {\n\t\textra := rollbackMessage()\n\t\tshowInstallError(\n\t\t\t"Setup did not finish",\n\t\t\t"Windows could not finish registering the application. Try again."+extra,\n\t\t)\n\t\treturn 1\n\t}\n\n\tif err := registerIntegratedUninstall(appPath, version); err != nil {\n\t\textra := rollbackMessage()\n\t\tshowInstallError(\n\t\t\t"Setup did not finish",\n\t\t\t"Windows could not register the Installed Apps uninstall entry. Try again."+extra,\n\t\t)\n\t\treturn 1\n\t}\n\n\ttransactionCommitted = true\n''',
)

# Snapshot every registry value that the installer changes so a failed upgrade can restore it.
write(
    "cmd/installer/registry_snapshot.go",
    '''package main

import (
\t"errors"

\t"github.com/bren-wp/Ghost-FTP/internal/platform"
)

type registryStringSnapshot struct {
\tkey     string
\tname    string
\tvalue   string
\texisted bool
}

type registryDWORDSnapshot struct {
\tkey     string
\tname    string
\tvalue   uint32
\texisted bool
}

type registrySnapshot struct {
\tstrings []registryStringSnapshot
\tdwords  []registryDWORDSnapshot
}

var installerStringRegistryValues = []struct{ key, name string }{
\t{appPathsKey, ""},
\t{uninstallKey, "DisplayName"},
\t{uninstallKey, "DisplayVersion"},
\t{uninstallKey, "Publisher"},
\t{uninstallKey, "InstallLocation"},
\t{uninstallKey, "DisplayIcon"},
\t{uninstallKey, "UninstallString"},
\t{uninstallKey, "QuietUninstallString"},
\t{uninstallKey, "URLInfoAbout"},
}

var installerDWORDRegistryValues = []struct{ key, name string }{
\t{uninstallKey, "NoModify"},
\t{uninstallKey, "NoRepair"},
}

func captureRegistrySnapshot() (registrySnapshot, error) {
\tvar out registrySnapshot
\tfor _, item := range installerStringRegistryValues {
\t\tvalue, existed, err := platform.GetRegistryString(item.key, item.name)
\t\tif err != nil {
\t\t\treturn registrySnapshot{}, err
\t\t}
\t\tout.strings = append(out.strings, registryStringSnapshot{
\t\t\tkey: item.key, name: item.name, value: value, existed: existed,
\t\t})
\t}
\tfor _, item := range installerDWORDRegistryValues {
\t\tvalue, existed, err := platform.GetRegistryDWORD(item.key, item.name)
\t\tif err != nil {
\t\t\treturn registrySnapshot{}, err
\t\t}
\t\tout.dwords = append(out.dwords, registryDWORDSnapshot{
\t\t\tkey: item.key, name: item.name, value: value, existed: existed,
\t\t})
\t}
\treturn out, nil
}

func (s registrySnapshot) restore() error {
\tvar errs []error
\tfor _, item := range s.strings {
\t\tvar err error
\t\tif item.existed {
\t\t\terr = platform.SetRegistryString(item.key, item.name, item.value)
\t\t} else {
\t\t\terr = platform.DeleteRegistryValue(item.key, item.name)
\t\t}
\t\tif err != nil {
\t\t\terrs = append(errs, err)
\t\t}
\t}
\tfor _, item := range s.dwords {
\t\tvar err error
\t\tif item.existed {
\t\t\terr = platform.SetRegistryDWORD(item.key, item.name, item.value)
\t\t} else {
\t\t\terr = platform.DeleteRegistryValue(item.key, item.name)
\t\t}
\t\tif err != nil {
\t\t\terrs = append(errs, err)
\t\t}
\t}
\treturn errors.Join(errs...)
}
''',
)

# Add the missing detailed 0.2.0 entry while preserving all prior version history.
changelog = read("CHANGELOG.md")
marker = "# Changelog\n\n"
if "## 0.2.0 - 2026-09-06 Beta" not in changelog:
    if not changelog.startswith(marker):
        raise SystemExit("unexpected CHANGELOG header")
    section = '''## 0.2.0 - 2026-09-06 Beta

### Desktop product scope

- Restricted the active application source and release contract to Windows and Linux; retired Web/PWA, Android, iOS and macOS application surfaces are not part of the 0.2.0 product.
- Kept one shared FTP/FTPS/SFTP Engine, settings model, profile model and transfer queue across Windows and Linux.
- Preserved English as the default/fallback language and the canonical 24-language registry for desktop and Setup surfaces.

### Windows UX and Setup

- Simplified the Windows workspace toward one canonical dual-pane file manager instead of presenting overlapping duplicate action layers.
- Kept native resize, minimize, maximize, restore and DPI-aware relayout behavior and improved compact-width geometry.
- Made protocol-specific SFTP credential controls contextual so FTP/FTPS sessions do not show irrelevant key/passphrase fields.
- Added an integrated Installed Apps uninstall path through the installed GhostFTP.exe using `--uninstall`, without shipping a separate Uninstall.exe.
- Portable builds reject the integrated uninstall command because only the canonical installed executable may remove installation-owned artifacts.
- Setup now writes a real Windows Installed Apps registration with display metadata and uninstall commands, and the registration participates in rollback-safe registry snapshots.
- Uninstall removes application-owned shortcuts/registrations and schedules the running executable for final deletion while preserving saved profiles/settings by default.

### Linux parity and localization

- Added a real 24-language selector to the Linux graphical Settings surface using the same normalized locale registry as Windows.
- Moved shared desktop geometry constants out of Windows-only source so Linux compiles independently and uses the same minimum/start workspace contract.
- Kept the native Linux X11/XWayland graphical client, headless terminal fallback and the same FTP/FTPS/SFTP operations exposed through the shared Engine.

### Security, privacy and stability

- Kept telemetry, analytics, advertising and external crash-reporting SDKs prohibited by fail-closed audits.
- Preserved SFTP host-key trust/pinning, transfer path validation, local symlink/reparse protections, atomic staging/rollback protections and bounded retry/timeout settings.
- Retained zero external Go modules and no third-party GUI framework; OS curl and OpenSSH remain explicit audited transport prerequisites.
- Strengthened Setup rollback so App Paths and every Installed Apps value modified by 0.2.0 can be restored if installation fails before commit.

### Repository, CI and release quality

- Removed the retired GhostFTP WEB application tree and continued blocking Web/PWA/mobile application roots from returning to active desktop source.
- Updated README/documentation around the Windows/Linux-only product contract, 24 languages, security/privacy boundaries and Setup/Portable behavior.
- Continued Windows x64/x86 Setup + Portable builds and Linux amd64/arm64/i386 DEB builds from one numeric VERSION source.
- Preserved detailed historical changelog entries instead of rewriting older release provenance.

'''
    changelog = marker + section + changelog[len(marker):]
    write("CHANGELOG.md", changelog)

# Remove the one-shot migration mechanism from the production branch after it has served its purpose.
for rel in (
    "scripts/apply_020_desktop_polish.py",
    ".github/workflows/apply-020-desktop-polish.yml",
):
    path = ROOT / rel
    if path.exists():
        path.unlink()

# Format all maintained Go source. This fixes the current CI gofmt blocker and
# keeps generated changes deterministic.
subprocess.run(
    ["gofmt", "-w", *[str(p) for root in (ROOT / "cmd", ROOT / "internal") for p in root.rglob("*.go")]],
    cwd=ROOT,
    check=True,
)

# Self-remove the migration files so the resulting branch contains only product changes.
for rel in ("scripts/finalize_020_core.py", ".github/workflows/finalize-020-core.yml"):
    path = ROOT / rel
    if path.exists():
        path.unlink()
