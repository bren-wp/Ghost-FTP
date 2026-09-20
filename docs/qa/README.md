# Ghost FTP QA Index — RC11

This directory tracks the evidence required before Ghost FTP can move from release candidate to FINAL.

## Interaction and functionality

- [Click / interaction QA](CLICK.md)
- [Transfer QA](TRANSFERS.md)
- [Installer QA](INSTALLER.md)

## Visual acceptance

- [Pixel parity](PIXEL_PARITY.md)
- [Responsive behavior](RESPONSIVE.md)
- [Native titlebar](TITLEBAR.md)

## Reference integrity

- [Reference checksums](REFERENCE_SHA256.txt)

## Release truth

The authoritative release/build state is [Build Status](../build/STATUS.md).

A source implementation, successful compile or compatibility-host test is **not** treated as proof of a target-OS acceptance gate unless that exact gate was executed.

## FINAL rule

Ghost FTP must not be labelled FINAL until the documented protocol, installer, titlebar and pixel/responsive acceptance gates have real evidence.

## RC11 UI regression checks

- Application header must remain below semantic modal surfaces.
- File/Tools/Help and other top menus must render as floating popovers and never push the Quick Connect or toolbar rows.
- About, Preferences, Site Manager, Transfer Center and New Connection must fully cover the underlying application when open.
- Default native geometry remains 1290×852 with adaptive checks at the documented smaller sizes.
