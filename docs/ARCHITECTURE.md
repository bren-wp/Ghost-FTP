# Ghost FTP architecture

Ghost FTP **0.0.6** is a multi-surface file-transfer product with a shared typed Go desktop engine, native Windows/Linux frontends, a native Android application, a maintained native macOS development frontend, four privacy-minimal browser helpers, a self-contained product website and an explicit server-assisted Web FTP client.

The canonical GitHub Release contains **13 platform artifacts / 16 public files**. `web/` and `web/ftp` are active source/deployment surfaces but are not additional binary release assets.

## Release identity

The root `VERSION` file is authoritative. The current candidate identity is `ghostftp-v0.0.6`, Current channel, `prerelease=false`. Publication must originate from the exact verified `main` commit and must not rewrite an existing tag/release.

## Shared desktop engine

- `internal/api` — typed application engine and connection/navigation/file-operation orchestration.
- `internal/remote` — FTP/FTPS/SFTP protocol execution, trust boundaries and remote staging semantics.
- `internal/transfer` — queue lifecycle, progress, retry/cancel and Top/Up/Down/Bottom ordering.
- `internal/config` — profiles/settings and protected saved-secret references.
- `internal/security` — host/path/fingerprint/secret validation and local destructive-operation guards.
- `internal/desktop` — Windows and Linux native frontends over the same engine.

## Windows

Windows is the reference desktop UI. Public 0.0.6 output is exactly:

```text
Ghost-FTP-0.0.6-Setup.exe
Ghost-FTP-0.0.6-Portable.exe
```

Each user-facing package embeds verified x64, x86 and ARM64 payloads. The bootstrap selects locally using native system architecture information and performs no architecture download. Official publication requires trusted Authenticode; CI smoke certificates never substitute for production signing.

## Linux

`linux/BUILD-DISTROS.sh` builds three native payloads: amd64, arm64 and i386. Those payloads are bundled into exactly six user-facing 0.0.6 files: Installer + Portable for Debian, Ubuntu and Fedora. Each bundle selects the matching payload locally.

Native installer/runtime/GUI evidence is maintained on Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64. ARM64/i386 remain cross-build/package evidence unless native runtime evidence is available.

## Android

Android publishes one production-signed 0.0.6 APK. The client uses Android-native Storage Access Framework capabilities, FTP + strict explicit FTPS, bounded parsing, staged transfer semantics and lifecycle generation ownership. Android SFTP remains hidden until strict maintained host-key verification is implemented.

## macOS

The AppKit frontend remains active development source over the shared engine. Development build success is not public distribution evidence. Public macOS packaging remains blocked until real Developer ID Application signing, Apple notarization, stapling/validation and Gatekeeper assessment succeed.

## Browser helpers

Chrome, Edge, Firefox and Opera packages are built from one shared local runtime plus browser-specific manifests. They request zero broad browser/host permissions, perform local parsing/copying only and have no supported browser-to-desktop launch/handoff.

## Website

`web/` is a self-contained static product website intended for ghostftp.com. It owns marketing, authentic runtime screenshots, canonical download links, security/privacy/legal pages and the route into Web FTP. It loads no external analytics, ads, fonts, CSS or JavaScript.

## Web FTP

`web/ftp` is intentionally a separate architecture from native Ghost FTP because browser JavaScript cannot open raw FTP/FTPS/SFTP sockets.

```text
Browser UI
   │ HTTPS operation request
   ▼
web/ftp/api.php
   ├── CurlFtpTransport  → FTP / explicit FTPS
   └── SftpTransport     → SFTP with pinned SHA-256 host key
```

The supplied application has no account database and no durable profile/password/history store. Each request reconstructs a transport, validates the destination, performs one bounded operation and closes the connection.

FTP/FTPS DNS is resolved before cURL execution. Private/reserved addresses are blocked by default, and the accepted IP is pinned with `CURLOPT_RESOLVE`. Explicit FTPS keeps the original hostname in the URL so TLS certificate/hostname verification remains meaningful.

SFTP connects to the validated IP, reads the remote host key, compares the expected SHA-256 fingerprint before authentication and fails closed on mismatch.

## Transfer and Remote Edit integrity

Core invariants include bounded/cancellable work, source/destination identity checks, staged activation, cleanup on failure, connection-generation ownership, truthful progress and bounded UTF-8/binary-safe Remote Edit behavior. The Web FTP editor applies an independent small-file boundary and never claims desktop queue semantics it does not implement.

## State and privacy

Native profiles/settings/bookmarks stay local. Ghost FTP has no product analytics backend. Web FTP is different by necessity: the trusted web host performs the protocol request. The supplied web application avoids durable credentials/history, but the hosting infrastructure is part of the runtime trust boundary.

## Release architecture

The 0.0.6 publication sequence is:

1. exact-head Go format/race/vet and repository/platform/security/privacy/docs/release audits;
2. Python regression suite including Web FTP contract checks;
3. trusted-signed Windows universal Setup/Portable;
4. six Linux universal distro bundles;
5. one production-signed Android APK with exact certificate fingerprint verification;
6. four deterministic browser helper ZIPs;
7. exact **13 platform artifacts / 16 public files** allow-list assembly and SHA-256 manifest;
8. non-prerelease GitHub Release creation and exact remote asset/digest readback;
9. exact-version GHCR distribution-bundle publication/readback;
10. latest-only cleanup only after integrity success.

See [Web](WEB.md), [Security](SECURITY.md), [Privacy](PRIVACY.md), [Platform parity](PLATFORM-PARITY.md), [Signing](SIGNING.md) and [Release verification](RELEASE-VERIFICATION.md).
