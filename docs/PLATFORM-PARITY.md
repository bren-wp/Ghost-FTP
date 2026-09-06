# Windows and Linux platform parity

Ghost FTP **1.0.0 Stable** is one desktop product with native Windows and Linux frontends. Both platforms use the **same typed `internal/api.Engine`** and the same protocol, transfer, profile, settings, localization and security layers.

Parity means equivalent user-facing capability and security semantics, not pixel-identical native widgets.

## Shared protocol contract

Both platforms support:

- FTP;
- FTPS;
- SFTP password authentication;
- SFTP private-key authentication;
- SFTP key passphrase handling;
- SFTP host-key fingerprint trust;
- local/remote directory navigation;
- upload/download and tree transfers;
- remote rename/delete/create/chmod where supported;
- connection timeout policy and privacy-safe diagnostics.

A secure transport failure is not silently retried as a weaker protocol on either platform.

## Shared profile/settings model

Windows and Linux use the same typed configuration/profile model. Platform-specific secret protection is intentionally different, but the persistence rule is the same: saved credentials are opt-in and remain local.

Settings normalization, conflict policy, retry behavior, parallelism, timeout and language selection are shared contracts.

## 24-language parity

Ghost FTP ships one **24-language** local registry. English is the default/fallback. Both desktop frontends consume normalized locale codes from the same catalog, and missing optional text falls back safely rather than creating an online translation dependency.

## Transfer parity

Both platforms route transfers through the same transfer manager and remote abstraction. Shared behavior includes:

- explicit queued/running/terminal states;
- pause/resume/cancel/retry/clear lifecycle;
- connection-generation binding;
- truthful progress, speed and ETA snapshots;
- retry classification;
- local containment checks;
- upload-source snapshot validation;
- staged/rollback-oriented remote operations;
- cleanup and terminal-state correctness.

Renderer timing may differ, but it must not alter transfer semantics.

## File-management parity

Both frontends expose the same core file-management workflow: local and remote panes, navigation, selection, refresh, create, rename, delete, upload and download.

Windows provides richer native keyboard/sorting behavior; Linux maintains equivalent selectable/navigation actions through its native input model. Improvements should converge behavior without forcing one platform's widget technology onto the other.

## Windows-specific implementation

Windows uses native Win32 UI, per-monitor layout handling, native dialogs and the current-user Windows saved-secret protection boundary. Production packages include x64/x86 Setup and Portable binaries.

Production Authenticode is optional. When a trusted protected signing identity is configured, every generated Windows artifact must verify successfully; when it is absent, official Stable publication remains explicitly unsigned and records `WINDOWS_AUTHENTICODE=unsigned`. A generated/self-signed development certificate is never substituted for the trusted production publisher identity.

## Linux-specific implementation

Linux uses the maintained native X11/XWayland-compatible frontend and local authenticated encryption for opt-in saved secrets. Production packages include amd64, arm64 and i386 DEB builds.

Idle rendering is event/state driven so the application does not continuously redraw the complete workspace when nothing relevant changed.

## Connection diagnostics parity

Both platforms consume the shared privacy-safe connection diagnostic classification. Low-level errors are mapped to actionable categories without intentionally reproducing credentials or secret payloads.

## Release parity

The production workflow independently builds/verifies both platform families before any public stable release. A successful Windows build cannot substitute for a failed Linux build, and vice versa.

The GitHub Release contains **9 platform artifacts** across Windows/Linux and **12 public files** total. The stable GHCR package mirrors the verified assembled release directory and does not introduce a third application implementation.

Signing parity is represented truthfully rather than forced: a configured Windows production signature is verified fail-closed, while an unsigned Windows release is allowed only when release metadata states that unsigned status explicitly. Linux integrity continues to use exact build provenance, package metadata and SHA-256 verification rather than Windows Authenticode.

## Definition of parity complete

A cross-platform feature is complete when:

1. shared Core/API semantics are implemented once where appropriate;
2. Windows and Linux can invoke the behavior through their native UI;
3. security/privacy boundaries are equivalent;
4. localization/fallback works;
5. platform tests and production builds pass;
6. documentation describes only capabilities actually maintained on both platforms or clearly labels a platform-specific difference.

See [Architecture](ARCHITECTURE.md), [Testing](TESTING.md), [Signing](SIGNING.md) and [Security](SECURITY.md).
