# Ghost FTP QA Index

This directory tracks the evidence and acceptance criteria used by the current Ghost FTP release-candidate line.

## Automated release gates

Every current RC must pass the exact-head quality workflow, real FTP / explicit FTPS / SFTP E2E and native Windows/Linux build workflow before publication.

Windows native QA covers seven critical surfaces:

- Main File Manager
- Site Manager
- New Connection
- Preferences
- Transfer Center
- File Properties
- Help & About

Each is captured at canonical, compact and near-minimum viewport sizes.

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

The authoritative current source/build state is [Build Status](../build/STATUS.md). Historical QA documents can describe older candidates; they must not be interpreted as proof for a newer release unless the corresponding exact-head workflow produced fresh evidence.

## FINAL rule

Ghost FTP must not be labelled FINAL solely because source compiles or packages. Stable promotion still requires the documented target-OS installer, visual, security and accessibility acceptance appropriate to that release.
