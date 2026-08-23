# Architecture

ByFTP has two intentionally separate runtime implementations that share product/release policy rather than process memory.

## Desktop core

The Windows/Linux/macOS implementation is written in Go and split into small typed packages. `cmd/byftp` starts the client, `cmd/installer` and `cmd/uninstaller` own the Windows lifecycle, `internal/api` exposes the engine, `internal/desktop` contains native/terminal presentation, `internal/remote` owns protocol sessions, `internal/transfer` owns queue state, `internal/config` owns durable settings/profiles, `internal/security` centralizes hardening, `internal/i18n` owns localization and `internal/platform` isolates operating-system primitives.

## Android

`android/` is a native Android application using Java 17 and Android platform APIs. It does not embed the Go desktop executable, start a localhost bridge or send credentials to a ByFTP service.

The Android module is separated into:

- `model/`: validated connection configuration, remote entries and remote path helpers.
- `remote/`: a small `RemoteClient` boundary with FTP/FTPS and SFTP adapters.
- `MainActivity`: Android lifecycle/UI orchestration and Storage Access Framework interaction.
- unit tests under `android/app/src/test`.

Apache Commons Net provides FTP/FTPS protocol primitives; SSHJ provides SFTP/SSH protocol primitives and its native SHA-256 fingerprint verifier is used for host-key pinning.

Android connection passwords are not persisted. Local file access is delegated to Android document providers through content URIs.

## Shared invariants

Both implementations use the root `VERSION` as the product version, keep project telemetry/advertising absent, contact only endpoints selected by the user and are required to pass repository release gates. The Android implementation does not weaken or bypass desktop security controls; it has its own mobile-specific audit surface.
