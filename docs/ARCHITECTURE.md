# Architecture

ByFTP has two intentionally separate runtime implementations that share product, security and release policy rather than process memory.

## Desktop core

The Windows/Linux/macOS implementation is written in Go and split into small typed packages. `cmd/byftp` starts the client, `cmd/installer` and `cmd/uninstaller` own the Windows lifecycle, `internal/api` exposes the engine, `internal/desktop` contains native/terminal presentation, `internal/remote` owns protocol sessions, `internal/transfer` owns queue state, `internal/config` owns durable settings/profiles, `internal/security` centralizes hardening, `internal/i18n` owns localization and `internal/platform` isolates operating-system primitives.

## Android

`android/` is a native Android application using Java 17 and Android platform APIs. It does not embed the Go desktop executable, start a localhost bridge or send credentials to a ByFTP service.

The Android module is separated into:

- `model/`: validated connection configuration, remote entries and remote-path helpers.
- `remote/`: a small `RemoteClient` boundary with independent FTP/FTPS and SFTP adapters.
- `MainActivity`: Android UI/lifecycle orchestration and Storage Access Framework interaction.
- resource/security policy under `res/`, including network-security, backup and device-transfer rules.
- unit tests under `android/app/src/test`.

Apache Commons Net provides FTP/FTPS protocol primitives. The FTPS adapter explicitly selects the Android/JVM platform trust manager, enables endpoint/hostname verification and protects private data channels with `PROT P`. SSHJ provides SFTP/SSH protocol primitives and its native SHA-256 fingerprint verifier is used for mandatory host-key pinning.

Android connection passwords and SSH secrets are not persisted. Local file access is delegated to Android document providers through content URIs, so the app does not request broad storage permissions. Application data is excluded from Android cloud-backup and device-transfer extraction rules.

The Activity keeps blocking network work on a dedicated executor. A connection that is still being established is tracked separately from the active client. During Activity destruction, pending and active remote clients are detached and closed outside the UI thread, executor work is interrupted and queued main-thread callbacks are discarded/ignored. This prevents an obsolete Activity instance from adopting a late connection or mutating destroyed UI state.

## Shared invariants

Both implementations use the root `VERSION` as the product version, keep project telemetry/advertising absent, contact only endpoints selected by the user and are required to pass repository release gates.

Desktop and Android security controls are intentionally platform-specific rather than simulated through one runtime. Android has dedicated audits for SFTP host-key pinning, FTPS trust/hostname checking, cleartext policy, storage permissions, backup/device-transfer exclusions, password persistence, lifecycle cleanup and version binding. Android validation is a required production release dependency even though a public production APK remains withheld until a stable private signing identity exists outside the repository.
