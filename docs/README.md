# Ghost FTP Documentation

This directory is the documentation hub for Ghost FTP. The repository root stays intentionally product-focused; engineering evidence, audits, release notes and operational guidance live here.

## Product

- [Implemented features](product/FEATURES.md) — what currently exists in RC9.
- [Recommended next work](product/ROADMAP.md) — what is worth adding or finishing next.
- [UI / UX contract](product/UI_UX.md) — canonical geometry, interaction and pixel-parity rules.

## Architecture and repository

- [Repository structure](architecture/STRUCTURE.md) — naming policy and why framework-required internal paths remain stable.

## Build and release status

- [Build status](build/STATUS.md) — authoritative build/release state.
- [Reference runtime notes](build/REFERENCE_RUNTIME.md) — compatibility/runtime context.
- [RC9 release notes](releases/2.1.1-rc.9.md) — current release candidate.
- GitHub Releases is the canonical source for downloadable binaries.

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
- [Titlebar QA](qa/TITLEBAR.md)
- [Transfer QA](qa/TRANSFERS.md)
- [Reference checksums](qa/REFERENCE_SHA256.txt)

## Legal and notices

- [Privacy](legal/PRIVACY.md)
- [Third-party notices](legal/THIRD_PARTY_NOTICES.md)
- Commercial license/EULA remain at repository root as `LICENSE.txt` and `EULA.txt`.
- `SECURITY.md` remains at repository root for GitHub security discovery.

## Visual references

The files under `assets/screenshots/` are the approved Ghost FTP design/QA reference set and are also used by the product README.

They are documentation and acceptance references only. Production UI must be implemented with real Ghost FTP components and controls; screenshot-as-UI or click-hotspot implementations are not accepted.

## Naming

User-facing names use **Ghost FTP**. Filenames, package names and technical product identifiers use **GhostFTP**.

Framework-specific internal directory names are retained only where renaming would create unnecessary build risk. Public release archives and documentation use GhostFTP-first names.
