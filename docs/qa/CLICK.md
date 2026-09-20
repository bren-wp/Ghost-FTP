# Ghost FTP Click / Interaction QA — RC10

## Verified in source and CI

The Ghost FTP desktop source contains real controls for the Main File Manager, Site Manager, New Connection, Preferences, Transfer Center, File Properties, About/Updates and the custom window chrome.

Current automated checks verify that the frontend typechecks and builds successfully and that Rust/Go unit and workspace checks execute in CI.

The production UI is built from real components. Reference screenshots are never used as runtime backgrounds or click maps.

## Interaction areas that exist in RC10

- Main menu and toolbar.
- Quick Connect.
- Site Manager list/search/details actions.
- New Connection protocol/host/port/credential controls.
- Preferences categories, Apply, Cancel and Reset.
- Transfer Queue and Transfer Center actions.
- File Properties General/Checksums/permissions workflows.
- About/Updates actions.
- Native minimize, maximize/restore, close and titlebar drag paths.
- Keyboard shortcut infrastructure and command palette.

## Remaining target-OS acceptance

Before FINAL, execute a complete Windows/Linux pointer and keyboard sweep of every visible action, including repeated modal open/close cycles, focus traversal, destructive confirmations, disabled states and all advertised language layouts.

Status: **component/source coverage exists; full native click-by-click acceptance remains a FINAL gate.**
