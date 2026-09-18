# Contributing

Ghost FTP changes should preserve correctness, security, privacy and platform-native behavior across the active Windows, Linux and Android applications.

## Before changing code

- understand the shared engine/state ownership;
- keep protocol security separate from visual work;
- do not fabricate UI state to match a screenshot;
- do not add telemetry or hidden external dependencies;
- keep English product terminology consistent.

## Platform changes

Windows changes must preserve native DPI/resize, keyboard/focus and packaging behavior.

Linux changes must preserve the shared engine contract, distro packaging/install lifecycle and native modal behavior.

Android changes must preserve Android lifecycle/storage constraints and the stricter SFTP capability boundary.

macOS is retired; do not reintroduce AppKit/Darwin-only source or macOS release workflows without an explicit future product decision.

## Validation

Run affected unit/contract tests, security/privacy audits and required platform workflows. UI changes require authentic exact-head runtime evidence where the repository provides it.

Never merge because a different commit was green.
