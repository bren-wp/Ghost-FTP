# Architecture

Ghost FTP is a multi-platform file-transfer product with three active native applications: **Windows, Linux and Android**.

## Shared desktop engine

Windows and Linux share typed Go application, model, transfer, profile and protocol boundaries. Native frontends own rendering and platform lifecycle; they do not implement separate protocol stacks.

The desktop architecture separates:

- connection/profile state;
- local/remote filesystem operations;
- transfer queue lifecycle;
- navigation/bookmarks;
- file filters and recursive search;
- directory comparison;
- remote permissions/editing;
- settings and localization;
- release/security verification.

## Windows

Windows uses the native desktop implementation under `internal/desktop` with Windows-specific files selected by build tags. The Files workspace follows the canonical Ghost FTP layout and preserves native keyboard, DPI, resize and accessibility semantics.

## Linux

Linux uses the maintained native X11/XWayland-compatible surface backed by the same shared engine contracts. The master Files layout exposes four primary rail destinations and keeps advanced file/connection tools behind the real **More** surface.

## Android

Android is a separate native mobile application under `android/`. It uses Android lifecycle/storage boundaries and does not pretend to have desktop-only capabilities. Android SFTP remains hidden until strict maintained host-key verification exists.

## Retired macOS implementation

macOS is no longer an active source or release platform. AppKit source, Darwin-only implementation files, macOS signing/build workflows and macOS-specific tests are removed from the current codebase.

## Security architecture

Protocol and credential boundaries are independent of UI parity work. Visual changes must not bypass TLS verification, SFTP host trust, credential protection, path safety or transfer ownership.

See [Security](SECURITY.md), [Privacy](PRIVACY.md) and [Platform parity](PLATFORM-PARITY.md).


## Windows architecture evidence

Ghost FTP Windows Setup and Portable are universal launchers with native **x64, x86 and ARM64** application payloads.

```text
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```

The ARM64 payload is built and structurally verified in CI. The metadata value above is intentionally explicit: current hosted CI does not claim native Windows-on-ARM runtime execution evidence.
