# Ghost FTP Documentation

This directory is the documentation hub for Ghost FTP. The root README is the product landing page; detailed engineering, QA, release and planning material lives here.

## Start here

- [Project status](PROJECT-STATUS.md) — what is finished, what is verified and what still blocks FINAL.
- [Feature matrix](roadmap/FEATURE-MATRIX.md) — implemented options and current support status.
- [Recommended roadmap](roadmap/ROADMAP.md) — improvements worth doing next, without presenting unfinished work as complete.
- [Build status](build/STATUS.md) — authoritative build/release evidence.
- [Installation](guides/INSTALLATION.md) — platform installation guidance.
- [Uninstall](guides/UNINSTALL.md) — removal and cleanup.
- [Update policy](guides/UPDATES.md) — signed update expectations.
- [Support](guides/SUPPORT.md) — diagnostics and support routes.
- [Privacy](legal/PRIVACY.md) — local-data and privacy model.

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

## Build and architecture

- [Build status](build/STATUS.md)
- [Reference runtime notes](build/REFERENCE_RUNTIME.md)

## Legal and notices

- [Privacy](legal/PRIVACY.md)
- [Third-party notices](legal/THIRD_PARTY_NOTICES.md)
- Commercial license/EULA remain at repository root as `LICENSE.txt` and `EULA.txt`.

## Releases

- [Ghost FTP 2.1.1 RC10](releases/2.1.1-rc.10.md) — current release candidate.
- [Ghost FTP 2.1.1 RC9](releases/2.1.1-rc.9.md) — previous release candidate.
- Older release notes are retained under `releases/archive/`.
- GitHub Releases is the canonical location for downloadable binaries.

## Repository naming

Product-facing paths use GhostFTP names:

- `ghostftp-desktop/`
- `ghostftp-web/`
- `ghostftp-runtime/`
- `ghostftp-installer/`

The internal `ghostftp-desktop/src-tauri/` folder is intentionally retained because the desktop framework expects that convention. It is treated as an implementation detail rather than a product-facing name.

## Visual references

Images under `assets/screenshots/` are the approved Ghost FTP design and QA reference set. They are documentation/acceptance references only. Production UI must be implemented with real controls and components; screenshot-as-UI implementations are not accepted.
