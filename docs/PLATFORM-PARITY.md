# Windows and Linux platform parity

Ghost FTP **1.1.1 Stable** is one desktop product with native Windows and Linux frontends. Both platforms use the **same typed `internal/api.Engine`** and the same protocol, transfer, profile, settings, localization and security layers.

Parity means equivalent protocol/security semantics and honest native-platform UX, not pixel-identical widgets or a requirement to expose a control before its backend lifecycle is complete.

## Shared protocol contract

Both platforms support:

- FTP;
- FTPS;
- SFTP password authentication;
- SFTP private-key authentication;
- SFTP key passphrase handling;
- SFTP host-key fingerprint trust;
- local/remote navigation;
- upload/download/tree transfers;
- remote file operations where supported;
- connection timeouts;
- privacy-safe diagnostics.

Fresh/quick-connect state resolves to **explicit FTPS on port 21** on both platforms. Plain FTP remains an explicit compatibility option. A secure transport failure is never silently retried as a weaker protocol.

## Shared profile/settings model

Windows and Linux use the same typed configuration/profile model. Platform-specific secret protection is intentionally different, but saved credentials remain opt-in and local.

Settings normalization, conflict policy, retry behavior, parallelism, timeout, language and appearance defaults remain shared contracts. Compatibility JSON fields are migration state, not justification for duplicate UI controls.

## Appearance in 1.1.1

**Classic Light is the primary fresh/fallback appearance.**

- **Windows** provides one `Classic Light / Dark` appearance choice. Fresh, missing or invalid appearance state resolves to Classic Light. An explicitly persisted Dark selection remains respected and is applied on the next process start so native title-bar/control theming is initialized coherently.
- **Linux GUI** uses Classic Light as its canonical palette. It intentionally does not expose a runtime theme switch until complete native switching can be delivered without mixed-state rendering or race-prone global palette changes.

This is an explicitly documented native-frontend difference rather than a fake parity control. Both implementations use local source-defined colors and add no theme service, browser runtime, telemetry or network dependency.

## 24-language parity

Ghost FTP ships one **24-language** local registry. English is the default/fallback. Both frontends consume normalized locale codes from the same catalog, and missing optional text falls back safely without online translation.

Security/privacy-sensitive credential-persistence prompts are catalog-backed rather than hardcoded into one Windows path.

## Transfer parity

Both platforms route transfers through the same transfer manager and remote abstraction. Shared behavior includes queued/running/terminal states, pause/resume/cancel/retry/clear lifecycle, connection-generation binding, truthful progress/speed/ETA snapshots, retry classification, local containment, upload-source snapshot validation, staged/rollback-oriented remote operations, cleanup and terminal-state correctness.

Renderer timing must not alter transfer semantics.

## Connection-manager parity

The frontends use the shared remote manager rather than independent session implementations. Regression coverage exercises successful manager connect/list/operation/disconnect behavior, invalid FTP credentials and FTPS-to-plaintext failure. Connection identity/generation prevents stale work from silently attaching to a later connection.

## File-management parity

Both frontends expose local/remote panes, navigation, selection, refresh, create, rename, delete, upload and download. Windows has richer native keyboard/sorting integration; Linux maintains equivalent selectable/navigation actions through its native input model.

## Security parity

Both platforms preserve:

- FTPS certificate and hostname validation;
- SFTP host-key trust;
- no silent secure-to-plain downgrade;
- validated paths and bounded recursive operations;
- symlink/reparse-aware local safety appropriate to the platform;
- protected credential handling with bounded secret lifetime;
- privacy-safe diagnostics without intentional credential reproduction.

SFTP protected-secret ownership distinguishes transient/session-owned material from credentials borrowed from stored profiles. Cancel/expiry/mismatch and failed setup paths must not retain newly owned secret blobs longer than necessary, and cleanup must not invalidate borrowed profile credentials.

## Windows-specific implementation

Windows uses native Win32 UI, DPI-aware layout, native dialogs and the current-user Windows saved-secret protection boundary. Production packages include x64/x86 Setup and Portable binaries.

The main profile-save flow and Site Manager use the same explicit credential-save consent semantics. Profile identity/path data can be saved without persisting entered credentials when the user declines consent.

Production Authenticode is optional. When a trusted protected signing identity is configured, every generated Windows artifact must verify successfully; when it is absent, official stable publication remains explicitly unsigned and records:

```text
WINDOWS_AUTHENTICODE=unsigned
```

A generated/self-signed development certificate is never substituted for trusted production publisher identity.

## Linux-specific implementation

Linux uses the maintained native X11/XWayland-compatible frontend and platform-local saved-secret/storage protections. Production packages include amd64, arm64 and i386 DEB builds.

Idle rendering is state/event driven so the complete workspace is not continuously repainted while nothing relevant changes.

## Release parity

The production workflow independently builds and verifies both platform families before publication. A successful Windows build cannot substitute for a failed Linux build, and vice versa.

The GitHub Release contract remains **9 platform artifacts / 12 public files**. GHCR mirrors the verified assembled release directory and is a distribution bundle, not a third application implementation.

## Definition of parity complete

A cross-platform feature is complete when:

1. shared Core/API semantics are implemented once where appropriate;
2. each platform exposes only behavior its native frontend can implement truthfully;
3. platform-specific differences are documented rather than hidden behind dead controls;
4. security/privacy boundaries remain equivalent;
5. localization/fallback works;
6. platform tests and production builds pass;
7. documentation and authentic UI evidence match the shipped implementation.

See [Architecture](ARCHITECTURE.md), [Settings](SETTINGS.md), [Reference UI](REFERENCE-UI.md), [Testing](TESTING.md), [Signing](SIGNING.md) and [Security](SECURITY.md).
