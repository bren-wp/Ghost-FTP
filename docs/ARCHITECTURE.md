# Architecture

ByFTP has two intentionally separate runtime implementations that share product, security and release policy rather than process memory.

## Desktop core

The Windows/Linux/macOS implementation is written in Go and split into small typed packages. `cmd/byftp` starts the client, `cmd/installer` and `cmd/uninstaller` own the Windows lifecycle, `internal/api` exposes the engine, `internal/desktop` contains presentation, `internal/remote` owns protocol sessions, `internal/transfer` owns queue state, `internal/config` owns durable settings/profiles, `internal/security` centralizes hardening, `internal/i18n` owns localization and `internal/platform` isolates operating-system primitives.

The Windows entrypoint also contains the constrained OpenSSH AskPass helper path. AskPass parent/token validation, protected-secret handling and environment clearing remain security boundaries separate from localized desktop UI text.

## Android runtime

`android/` is a native Android application using Java 17 and Android platform APIs. It does not embed the Go desktop executable, start a localhost bridge or send credentials to a ByFTP service.

The Android module is separated into:

- `model/`: validated connection configuration, remote entries and remote-path helpers.
- `remote/`: a small `RemoteClient` boundary with independent FTP/FTPS and SFTP adapters.
- `MainActivity`: Android UI/lifecycle orchestration and Storage Access Framework interaction.
- `res/`: UI resources plus network-security, backup and device-transfer policy.
- `android/app/src/test`: unit/security/path/version regressions.

Apache Commons Net provides FTP/FTPS primitives. The FTPS adapter explicitly uses platform trust, endpoint/hostname verification and `PROT P`. FTP/FTPS records the authenticated login working directory and maps the UI root to that account namespace, with a login-relative fallback when `PWD` is unavailable.

SSHJ provides SFTP/SSH primitives and its native SHA-256 fingerprint verifier is used for mandatory host-key pinning.

Android connection passwords and SSH secrets are not persisted. Local file access is delegated to Android document providers through content URIs; broad storage permissions are not requested. Application data is excluded from Android cloud-backup and device-transfer flows.

Blocking network work stays on a dedicated executor. Pending and active remote clients are tracked separately; destruction detaches/closes both, interrupts queued work and discards late UI callbacks. Download-picker state is consumed/cleared when a result arrives so stale remote targets do not survive an incomplete picker flow.

## Build and packaging boundary

Root `VERSION` is shared across desktop and Android product metadata. Android Gradle produces a debug APK and an optimized unsigned release APK. `scripts/package_android.py` validates both APK containers and stages versioned release names. This packaging step does not create a production publisher identity.

The public GitHub Release can therefore contain build-valid Android APK artifacts while production Android signing remains a separate external trust boundary. The debug APK uses the standard debug identity; the release APK remains unsigned until a real private production identity is applied outside the repository.

## Shared invariants

Both runtimes use the root `VERSION`, keep project telemetry/advertising absent, contact only endpoints selected by the user and must pass repository release gates.

Desktop and Android controls are platform-specific rather than simulated through a shared hidden runtime. Android has dedicated audits for SFTP host-key pinning, FTPS trust/hostname checks, FTP login-root paths, cleartext policy, storage/backup rules, password persistence, picker/lifecycle cleanup, APK packaging and version binding. Windows/Linux/macOS keep their existing independent build and security gates.
