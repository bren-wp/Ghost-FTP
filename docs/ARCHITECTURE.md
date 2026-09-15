# Ghost FTP architecture

Ghost FTP **0.0.6** is a multi-platform file-transfer product with a shared typed Go desktop engine, native Windows/Linux frontends, a native Android application, an active native macOS source surface and browser extension packages.

The former repository website and Web FTP application have been removed. Protocol credentials and file-transfer traffic must not be routed through a Ghost FTP-operated remote proxy.

## Shared desktop engine

- `internal/api` — typed application engine and connection/navigation/file-operation orchestration.
- `internal/remote` — FTP/FTPS/SFTP execution and trust boundaries.
- `internal/transfer` — queue lifecycle, progress, retry, cancellation and ordering.
- `internal/config` — profiles, settings and protected saved-secret references.
- `internal/security` — host/path/fingerprint/secret validation and destructive-operation guards.
- `internal/desktop` — Windows and Linux native frontends over the same engine.

Shared validation and error mapping should live in common layers where platform architecture permits. UI surfaces must not duplicate protocol policy or weaken trust rules.

## Windows

Windows is the reference desktop UX. Published 0.0.6 output is:

```text
Ghost-FTP-0.0.6-Setup.exe
Ghost-FTP-0.0.6-Portable.exe
```

Each package embeds verified x64, x86 and ARM64 payloads and selects locally. Official publication requires trusted Authenticode. CI smoke certificates are never production substitutes.

```text
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```

## Linux

`linux/BUILD-DISTROS.sh` builds amd64, arm64 and i386 payloads and packages Installer + Portable bundles for Debian, Ubuntu and Fedora. Native runtime evidence is maintained only where actual maintained runners execute the product; cross-built payloads are not mislabeled as native runtime evidence.

## Android

Android publishes one production-signed APK. Local file authority uses Android's Storage Access Framework. FTP and strict explicit FTPS are exposed. Android SFTP remains hidden until strict maintained host-key verification is implemented.

Lifecycle ownership, cancellation, network loss and staged transfer semantics must prevent expected transport failures from crashing the application.

## macOS

The AppKit frontend remains active source over shared product logic. Public macOS distribution remains gated on real Developer ID Application signing, notarization, stapling/validation and Gatekeeper assessment.

## Browser extensions

Chrome, Edge, Firefox and Opera share packaged browser UI/runtime code with browser-specific Manifest V3 metadata. Browser JavaScript does not open arbitrary raw FTP, FTPS or SFTP sockets.

The 0.0.7 development architecture implements this local path:

```text
Browser popup
    ↓ extension runtime messaging
Background relay
    ↓ Native Messaging: com.ghostftp.bridge
Ghost FTP native host
    ↓ typed internal/api Engine
internal/remote + internal/transfer
    ↓ direct FTP / FTPS / SFTP
User-selected server
```

The browser packages request exactly `nativeMessaging`. They do not request host permissions, tab access, extension storage, content scripts or externally-connectable web origins. Password and passphrase inputs are not persisted in browser storage; optional saved credentials remain in the native protected profile store.

`cmd/ghostftp-native-host` accepts bounded length-prefixed messages, uses strict JSON schemas and maps a fixed operation set into the existing Engine. It does not expose a localhost HTTP/WebSocket service. Local file operations require a user-selected root and reject traversal or symlink escape outside that root. SFTP uses the existing fail-closed host-key verification path.

The background relay keeps the native port open only while the popup is connected, a request is pending or a real transfer remains queued/running. Transfer state comes from the existing Engine event stream rather than a second browser-side queue.

Native host manifests are generated with `scripts/build_native_host_manifests.py`. Firefox uses the fixed signed extension identity `ghostftp-connection-helper@ghostftp.com`. Chromium-family registration requires explicit 32-character extension IDs; wildcard origins and invented store IDs are prohibited and the generator can fail closed when an ID is required but unavailable.

Fake, simulated or remote-proxy FTP/SFTP behavior remains prohibited. Credentials and transfer bytes never need a Ghost FTP-operated server.

## Transfer integrity

Core invariants include bounded/cancellable work, source/destination identity checks, staged activation, cleanup on failure, connection-generation ownership, truthful progress and privacy-safe error mapping. Expected server/network failures return controlled application errors rather than crashes.

## State and privacy

Profiles, bookmarks and settings remain local. Passwords must never be written to plaintext configuration, URLs or logs. Where persistence is supported, secrets use the platform's protected credential facility. Ghost FTP has no telemetry or analytics backend.

## Published release architecture

The published 0.0.6 release contains **13 platform artifacts / 16 public files** across Windows, Linux, Android and four browser packages plus release metadata. `ghostftp-v0.0.6` is published release history and is not rewritten by 0.0.7 cleanup work.

See [Security](SECURITY.md), [Privacy](PRIVACY.md), [Platform parity](PLATFORM-PARITY.md), [Signing](SIGNING.md), [Testing](TESTING.md) and [Release verification](RELEASE-VERIFICATION.md).
