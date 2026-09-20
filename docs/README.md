# Ghost FTP Documentation

This directory is the documentation hub for **Ghost FTP**. The repository root is intentionally product-focused; detailed engineering evidence, release state, audits, QA and operational guidance live here.

## Start here

- [Project status & next work](product/STATUS_AND_NEXT.md) — what is implemented, what was improved in RC9, what still blocks FINAL, and what is recommended next.
- [Implemented features](product/FEATURES.md) — current functionality only.
- [Roadmap](product/ROADMAP.md) — recommended future improvements, clearly separated from implemented features.
- [UI / UX contract](product/UI_UX.md) — canonical 1:1 geometry, interaction and responsive rules.

## Repository architecture and naming

- [Repository structure](architecture/STRUCTURE.md)
- [GhostFTP naming policy](architecture/NAMING.md)

The repository uses branded top-level source paths:

- `ghostftp-desktop/`
- `ghostftp-runtime/`
- `ghostftp-installer/`
- `ghostftp-web/`
- `ghostftp-updates/`

Framework-specific terms remain only where technically required by the build ecosystem.

## Build and release

- [Build status](build/STATUS.md) — authoritative build and release state.
- [Reference runtime notes](build/REFERENCE_RUNTIME.md)
- [RC9 release notes](releases/2.1.1-rc.9.md)
- GitHub Releases is the canonical source for published binaries and checksums.

## Guides

- [Installation](guides/INSTALLATION.md)
- [Uninstall](guides/UNINSTALL.md)
- [Update policy](guides/UPDATES.md)
- [Support](guides/SUPPORT.md)

## Audits

- [Project audit](audits/PROJECT.md)
- [Code audit](audits/CODE.md)
- [Branding audit](audits/BRANDING.md)
- [Language audit](audits/LANGUAGE.md)
- [Security audit](audits/SECURITY.md)

## QA

- [QA overview](qa/README.md)
- [Click/interaction QA](qa/CLICK.md)
- [Installer QA](qa/INSTALLER.md)
- [Pixel parity](qa/PIXEL_PARITY.md)
- [Responsive QA](qa/RESPONSIVE.md)
- [Native titlebar QA](qa/TITLEBAR.md)
- [Transfer QA](qa/TRANSFERS.md)
- [Reference checksums](qa/REFERENCE_SHA256.txt)

## Legal and notices

- [Privacy](legal/PRIVACY.md)
- [Third-party notices](legal/THIRD_PARTY_NOTICES.md)
- Commercial license/EULA remain at repository root as `LICENSE.txt` and `EULA.txt`.
- `SECURITY.md` remains at repository root for GitHub security discovery.

## Visual references

The approved reference media under `assets/screenshots/` is used by the README and release QA.

Current branded reference filenames include:

- `ghostftp-main-file-manager.webp`
- `ghostftp-site-manager.webp`
- `ghostftp-new-connection.webp`
- `ghostftp-preferences.webp`
- `ghostftp-transfer-center.webp`
- `ghostftp-file-properties.webp`
- `ghostftp-about.webp`
- `ghostftp-platforms.webp`
- `ghostftp-brand-board.webp`

These files are **documentation and acceptance references only**. Production UI is implemented with real Ghost FTP components and controls; screenshot-as-UI and click-hotspot implementations are not accepted.

## Naming summary

- **Ghost FTP** — user-facing product name.
- **GhostFTP** — filenames, packages, archives and technical product identifiers.
- Internal framework names remain only where technically required.
