# Ghost FTP security

Ghost FTP **0.0.6** uses explicit transport, identity, path, secret, process and release boundaries. Security-sensitive behavior is tested in shared product code, platform adapters and protected release workflows.

## Transport policy

- **FTP** is an intentional unencrypted compatibility mode.
- **Explicit FTPS** validates certificate trust and server hostname identity; secure failure is never silently retried as plain FTP.
- **Desktop SFTP** requires strict SSH host-key trust/pinning.
- **Android SFTP** remains hidden until strict host-key verification has a maintained Android implementation.
- **Browser extensions** do not fake raw FTP/SFTP sockets. The 0.0.7 development surface uses a local trusted Ghost FTP Native Messaging bridge backed by the existing Engine.

## Input and path boundaries

Hosts, ports, control characters, remote paths and destructive filesystem operations are validated before use. Desktop local destructive operations preserve root/symlink/reparse protections. Android local authority remains Storage Access Framework scoped.

Remote server replies are treated as untrusted input. User-facing errors must not expose raw stack traces, internal exception types, source paths or secrets.

## Transfer lifecycle

Transfers are lifecycle-owned operations rather than blind copies. Maintained behavior covers staged commit/rollback, upload source snapshots, cancellation before final activation, connection-generation ownership, stale-completion rejection and cleanup after failure.

Expected DNS, authentication, permission, timeout, disconnect, disk-full and server-unavailable conditions must be handled as controlled failures instead of process crashes.

## Credential boundary

Never place passwords in URLs. Never log plaintext passwords/passphrases. Never store passwords in plaintext files. Saved credential persistence is opt-in and platform-local, using protected OS facilities where available.

Endpoint/account/key identity changes must not silently carry a protected credential into a different identity.

## Browser extension security

Official Chrome, Edge, Firefox and Opera packages request exactly one browser permission: `nativeMessaging`. They do not request extension storage, tabs, host permissions, content scripts, externally-connectable web origins or remote code.

The browser runtime is required to remain free of remote network APIs such as `fetch`, `XMLHttpRequest`, WebSocket and `sendBeacon`. It also rejects dynamic code execution and server-controlled HTML insertion in the maintained contract.

The Native Messaging companion:

- is registered as `com.ghostftp.bridge` and accepts requests only through the browser's native-messaging mechanism;
- uses bounded little-endian length-prefixed messages and strict JSON decoding with unknown fields rejected;
- routes only a fixed operation set into the existing typed Ghost FTP Engine;
- validates request type, protocol, host, port, IDs, local paths, remote paths and file/folder names;
- rejects local traversal and symlink escape outside the operating-system-selected local root;
- never exposes an unauthenticated localhost HTTP or WebSocket control port;
- never forwards credentials, directory listings or file data to a Ghost FTP remote service;
- returns privacy-safe errors through the existing `internal/usererror` mapping rather than raw curl/OpenSSH/OS diagnostics;
- preserves the existing SFTP first-contact fingerprint confirmation and changed-key blocking behavior;
- keeps the native port open only while a UI client, pending request or active transfer requires it.

The browser extension does not maintain a second transfer implementation. Upload/download lifecycle, retry, cancel, progress and queue state remain owned by `internal/transfer` and are observed through the Engine event stream.

Native-host registration is fail-closed. Firefox uses the fixed signing identity `ghostftp-connection-helper@ghostftp.com`. Chromium-family manifests require explicit valid extension IDs and produce exact `chrome-extension://<id>/` origins. Wildcard origins and invented store IDs are prohibited.

## Linux external-tool trust

Linux accepts protocol tooling only through maintained executable/directory provenance rules. Credential-bearing AskPass flows require trusted Ghost FTP helper/process identity and fail closed where that provenance cannot be established.

## Windows package security

Published Windows output is `Ghost-FTP-0.0.6-Setup.exe` and `Ghost-FTP-0.0.6-Portable.exe`, each carrying internal x64, x86 and ARM64 payloads selected locally. Staged payload bytes are verified before execution and no architecture payload is fetched from the network.

```text
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```

Official publication is signed-only. Production PFX/password material is protected outside the repository and final public executables must pass trusted Authenticode verification.

## Android signing security

The published APK is signed with the protected production keystore. `apksigner verify --verbose --print-certs` must succeed and the signer certificate SHA-256 must match `GHOSTFTP_ANDROID_CERT_SHA256` exactly. CI-only identities are not production substitutes.

## Release supply chain

Release workflows pin third-party Actions revisions, disable Go telemetry/external module resolution, run exact-head quality gates, require protected production signing, enforce a canonical release allow-list, generate SHA-256 metadata and refuse to rewrite an existing release identity.

The published `ghostftp-v0.0.6` release is immutable history. The 0.0.7 cleanup branch does not republish or retag it.

## Security testing

```text
go test -race ./...
go vet ./...
python scripts/audit_security.py
python scripts/audit_privacy.py
python scripts/audit_dependencies.py
python scripts/audit_repository.py
python scripts/audit_release.py
python -m unittest discover -s scripts -p 'test_*.py'
```

The dedicated browser workflow additionally tests and vets `cmd/ghostftp-native-host`, validates exact browser permissions/runtime restrictions, builds deterministic extension ZIPs and tests fail-closed Native Messaging manifest generation.

## Reporting a vulnerability

Never put real passwords, private keys, passphrases, server private data, signing credentials or notarization secrets into a public issue. Use synthetic reproduction data and privacy-safe logs.

See [Privacy](PRIVACY.md), [Signing](SIGNING.md), [Testing](TESTING.md) and [Support](SUPPORT.md).
