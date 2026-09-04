# Architecture

Ghost FTP is a multi-platform file-transfer product with separate native/runtime implementations that share product, security, versioning and release policy.

## Runtime surfaces

### Desktop core — Windows, Linux and macOS

The desktop core is written in Go:

- `cmd/byftp/` is the legacy-named desktop entry point retained for source compatibility.
- `cmd/installer/` owns the Windows installation transaction.
- `internal/api/` exposes typed application operations.
- `internal/desktop/` contains platform presentation and desktop interaction.
- `internal/remote/` owns FTP, FTPS and SFTP connection boundaries.
- `internal/transfer/` owns transfer state and queue lifecycle.
- `internal/config/` owns durable settings and profiles.
- `internal/security/` centralizes path, secret and process hardening.
- `internal/i18n/` owns runtime localization.
- `internal/platform/` isolates operating-system primitives.

Linux packaging lives under `linux/`; macOS packaging lives under `macos/`. Both build the shared Go core rather than carrying forked protocol implementations.

The maintained Go toolchain is pinned by CI and version audits. Production build scripts disable Go telemetry before building and keep external module fetching disabled for the dependency-free Go module graph.

### Windows installation boundary

The Windows installer uses an application-only verified payload. Legacy identifiers such as `ByFTP.exe`, old App Paths entries and old uninstall registry keys may remain where required to upgrade or clean existing installations safely. They are compatibility identifiers, not public branding.

The installer validates its embedded payload, stages verified bytes, protects against redirected/reparse installation paths and uses rollback-aware file/registry transactions. Ghost FTP does not publish a standalone uninstaller binary from this pipeline.

### Android

`android/` is a native Android application. It contains its own connection, remote-browser and lifecycle implementation and does not depend on a project-controlled backend service.

The package/application identifiers may retain `byftp` for update identity compatibility, while the visible application name is **Ghost FTP**. Android release CI tests, lints and assembles an installable APK. A production store signing identity is intentionally external to the repository.

### iOS

`ios/` is a native Swift/Xcode application. The existing Xcode project/bundle identifiers may retain legacy identifiers where changing them would break application identity. User-visible naming is **Ghost FTP**.

The iOS build derives its marketing version from root `VERSION`, builds a real arm64 iPhoneOS application and packages an unsigned IPA. Normal device/TestFlight/App Store distribution requires an externally managed Apple signing identity and provisioning profile.

### Web/PWA

`ByFTP WEB/` is the legacy-named source directory for the Ghost FTP PHP/shared-hosting application. The directory and internal PHP namespace are retained to avoid unnecessary migration churn; public application naming is **Ghost FTP**.

The web runtime contains:

- `app/Remote/` — FTP/SFTP transport and remote-path boundaries.
- `app/Security/` — authentication, rate limiting, host validation, encryption and security logging.
- `app/Storage/` — durable users, preferences and encrypted connection profiles.
- `app/Operations/` — higher-level remote operations.
- `tests/` — executable PHP regression tests.
- `assets/` plus `manifest.webmanifest` and `service-worker.js` — PWA presentation/runtime assets.

Sensitive navigation/API/download responses are never cached by the service worker. Ghost FTP uses a `ghostftp-static-vX.Y.Z` cache namespace and removes legacy `byftp-static-*` caches during activation.

## Security boundaries

Security-sensitive design principles are shared across implementations:

- user-selected server destinations only;
- no application telemetry or advertising SDKs;
- strict host/path/port validation;
- fail-closed traversal and noncanonical path handling;
- SFTP host-key/fingerprint verification where SFTP is supported;
- platform TLS verification for FTPS;
- bounded temporary-file and transfer handling;
- minimized credential lifetime and no persistent plaintext secret logging;
- rollback/cleanup paths for interrupted operations;
- no hidden network probes or background scanners.

The web application additionally enforces strict session cookies, CSRF protection, cross-site POST rejection, security headers, rate limiting and runtime-storage exclusion from release archives.

## Version source

Root `VERSION` is the canonical production version source. Ghost FTP starts at **1.0.0**. Platform build metadata is derived from that version and CI rejects drift between desktop, Android, iOS, Linux, macOS and web packaging.

Historical Git tags are not rewritten. Ghost FTP releases use the separate namespace `ghostftp-vX.Y.Z`.

## Release architecture

`.github/workflows/ci.yml` validates pull requests and `main`. `.github/workflows/release.yml` is the single production publication path.

The release workflow builds independent platform stages and assembles exactly eight public platform packages:

1. Windows x64 Setup
2. Windows x86 Setup
3. Windows x32 alias of the x86 Setup
4. Linux multiarch archive containing amd64/arm64/i386 DEBs
5. macOS Universal PKG
6. Android APK
7. iOS arm64 unsigned IPA
8. Web shared-hosting ZIP

It also creates `SHA256.txt`, `RELEASE-NOTES.txt` and `BUILD-METADATA.txt`, for exactly **11 public release files**.

Publication verifies that `main` still points to the release commit and refuses to move an existing `ghostftp-vX.Y.Z` tag to a different commit.
