# Ghost FTP QA Index

This directory tracks evidence and acceptance criteria for the current Ghost FTP development/release line.

## Automated release gates

Every release source must pass the exact-head quality, real FTP/explicit FTPS/SFTP E2E, canonical Windows/Linux native build, Android and Windows hardening workflows before publication. The Android gate includes an emulator click-through smoke on the same tested source.

The quality gate also verifies that local README/documentation images are byte-identical to the newest published release tag.

Windows native QA covers seven critical surfaces:

- Main File Manager
- Site Manager
- New Connection
- Preferences
- Transfer Center
- File Properties
- Help & About

Each is captured at canonical, compact and near-minimum viewport sizes by the canonical native build.

## Interaction and functionality

- [Click / interaction QA](CLICK.md)
- [Transfer QA](TRANSFERS.md)
- [Installer QA](INSTALLER.md)
- [Protocol E2E](PROTOCOL_E2E.md)

## Visual acceptance

- [Pixel parity](PIXEL_PARITY.md)
- [Responsive behavior](RESPONSIVE.md)
- [Native titlebar](TITLEBAR.md)

## Reference integrity

- [Reference checksums](REFERENCE_SHA256.txt)

## Release truth

The authoritative current source/build state is [Build Status](../build/STATUS.md). Historical QA documents are not proof for a newer release unless the corresponding exact-head workflow produced fresh evidence.

## FINAL rule

Ghost FTP must not be labelled FINAL solely because source compiles or packages. Stable promotion still requires the documented target-OS installer, visual, security and accessibility acceptance appropriate to that release.
