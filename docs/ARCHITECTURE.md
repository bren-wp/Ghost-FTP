# Ghost FTP architecture

Ghost FTP **0.0.6** is a multi-platform file-transfer product built around a shared typed Go desktop engine, native Windows/Linux frontends, a native Android application, a maintained native macOS development frontend and four privacy-minimal browser helper packages.

The canonical public release contains **13 platform artifacts / 16 public files**. Repository-hosted website and browser protocol-client source are retired and are no longer part of the application architecture.

## Release identity

The root `VERSION` file is authoritative. The active source identity is `0.0.6`; publication must originate from the exact verified source commit and must not silently rewrite an existing release identity.

## Shared desktop engine

- `internal/api` — typed application engine and connection/navigation/file-operation orchestration.
- `internal/remote` — FTP/FTPS/SFTP protocol execution, trust boundaries and remote staging semantics.
- `internal/transfer` — transfer lifecycle, progress, retry/cancel and priority ordering.
- `internal/config` — profiles, settings and protected saved-secret references.
- `internal/security` — host/path/fingerprint/secret validation and local destructive-operation guards.
- `internal/desktop` — native Windows and Linux frontends over the shared engine.

The shared engine intentionally avoids a generic JSON command dispatcher. Native frontends call typed application capabilities and keep security-sensitive validation close to the engine boundary.

## Windows

Windows is the reference desktop experience. Public output is:

```text
Ghost-FTP-0.0.6-Setup.exe
Ghost-FTP-0.0.6-Portable.exe
```

Each package carries x64, x86 and ARM64 payloads internally and selects the matching payload locally. Official publication requires trusted Authenticode signing; CI smoke certificates are development evidence only.

## Linux

`linux/BUILD-DISTROS.sh` builds amd64, arm64 and i386 payloads and packages them into Installer and Portable bundles for Debian, Ubuntu and Fedora. Runtime evidence must distinguish native execution from cross-build/package verification.

## Android

Android publishes one production-signed APK and uses Android-native Storage Access Framework authority for local files. Maintained protocol support is FTP plus strict explicit FTPS. Android SFTP remains hidden until strict maintained host-key verification exists.

Android network/file operations are lifecycle-owned and use bounded parsing, staged transfer semantics and cancellation/generation guards so stale work cannot silently overwrite newer application state.

## macOS

The AppKit frontend remains active development source over the shared engine. Build success is not public distribution evidence. Public packaging remains blocked until real Developer ID Application signing, Apple notarization, stapling/validation and Gatekeeper assessment succeed.

## Browser helpers

Chrome, Edge, Firefox and Opera packages share a local helper runtime with browser-specific manifests. They are intentionally narrow companion tools rather than protocol engines: no broad browser/host permissions, no credential persistence, no telemetry backend and no supported browser-to-desktop launch/handoff.

## Transfer and Remote Edit integrity

Core invariants include bounded/cancellable work, source/destination identity checks, staged activation, cleanup on failure, connection-generation ownership, truthful progress and bounded UTF-8/binary-safe Remote Edit behavior.

## State and privacy

Profiles, settings and bookmarks remain local to their platform. Ghost FTP has no product analytics backend, advertising SDK, hidden synchronization service or mandatory product account. Runtime network access is limited to user-directed transfer/protocol operations and platform/release infrastructure documented elsewhere.

## Release architecture

The 0.0.6 publication sequence is:

1. exact-head Go format/race/vet and repository/platform/security/privacy/docs/release audits;
2. complete Python regression suite;
3. trusted-signed Windows universal Setup/Portable;
4. six Linux distro bundles;
5. one production-signed Android APK with exact certificate fingerprint verification;
6. four deterministic browser helper ZIPs;
7. exact **13 platform artifacts / 16 public files** allow-list assembly and SHA-256 manifest;
8. release publication and remote asset/digest readback;
9. exact-version GHCR distribution-bundle verification;
10. retention cleanup only after integrity succeeds.

See [Security](SECURITY.md), [Privacy](PRIVACY.md), [Platform parity](PLATFORM-PARITY.md), [Signing](SIGNING.md) and [Release verification](RELEASE-VERIFICATION.md).
