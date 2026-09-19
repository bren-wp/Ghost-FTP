# Contributing

Ghost FTP changes must preserve the active Windows, Linux and Android product contract.

## Before changing code

Identify which maintained surface is affected:

- shared Go engine;
- Windows desktop;
- Linux desktop;
- Android;
- browser helpers;
- packaging / release tooling;
- documentation.

## Required quality

Run the relevant automated gates and keep the changed source exact-head clean:

- gofmt;
- Go tests;
- Go vet;
- Python regression tests;
- security/privacy audits;
- packaging checks;
- CodeQL / Govulncheck where applicable.

## UI changes

Windows and Linux changes must preserve the supplied dual-pane Ghost FTP hierarchy, readable labels, keyboard focus and engine-backed controls.

Android changes must preserve readable touch targets, Android lifecycle behavior and the Files / Sites / Bookmarks / Transfers / Settings navigation model.

Do not add developer-only placeholder text to shipping UI.

## Security

Do not weaken:

- strict desktop SFTP host-key trust;
- FTPS certificate and hostname validation;
- protected local secret handling;
- release signing boundaries;
- the Android rule that SFTP stays hidden until strict host-key verification is implemented there.

## Runtime evidence

Reference images define the visual target. Runtime claims require authentic screenshots from the exact tested source SHA.

## Retired platform

Do not restore the retired macOS application, macOS build workflows or macOS-specific tests without an explicit product-scope decision.
