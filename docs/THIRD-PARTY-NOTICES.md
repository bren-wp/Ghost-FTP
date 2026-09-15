# Third-party notices

Ghost FTP **0.0.6** keeps the maintained desktop/core Go module intentionally free of external Go module requirements. The desktop engine uses operating-system/platform facilities for protocol execution and UI rather than silently downloading an untracked runtime stack.

The published 0.0.6 release includes **Windows, Linux and Android** applications plus **Chrome, Edge, Firefox and Opera** extension packages. macOS is an active native development/source frontend and remains outside the public **16-file** release until real Developer ID signing and Apple notarization succeed.

## Desktop runtime transport tools

### FTP / FTPS

The shared desktop transport uses operating-system `curl` for FTP and explicit FTPS. Ghost FTP constrains configuration/environment so ambient proxy/config state cannot silently redirect a selected transfer. Secure FTPS failure is not downgraded to plain FTP.

### SFTP

The shared desktop SFTP transport uses operating-system OpenSSH `ssh` / `sftp` tooling with strict host-key trust and constrained credential handoff. Windows normally uses the Windows OpenSSH Client capability and Linux the distribution OpenSSH client package.

## Linux prerequisites

Debian/Ubuntu packages declare `ca-certificates`, `curl` and `openssh-client`; Fedora declares `ca-certificates`, `curl` and `openssh-clients`. Portable archives do not vendor private copies of those tools or CA stores.

## Android public dependency boundary

Android 0.0.6 is a public native application through the protected production-signing path. Java/Android SDK and Gradle components are build/platform dependencies. Android exposes FTP and strict explicit FTPS; **SFTP remains hidden** until strict maintained host-key identity support exists.

## Browser extensions

Browser packages use browser-provided extension APIs and repository-local JavaScript/CSS/assets. The published 0.0.6 helper does not itself contain an FTP/SFTP socket stack. For 0.0.7, real protocol parity may use a local Ghost FTP native-messaging companion because the browser sandbox cannot open arbitrary raw protocol sockets.

A native companion is an explicit local product component. It must use browser-supported native messaging, least privilege and local credential handling. It must not become an unauthenticated localhost service or a Ghost FTP remote credential/transfer relay.

## macOS dependency boundary

The AppKit frontend is built from repository source and shared product logic. Public distribution requires a real Developer ID Application identity and successful Apple notarization. Ad-hoc/development signing is not production evidence.

## GitHub Actions

Build workflows use pinned GitHub Actions revisions for checkout, toolchain setup, security analysis and artifact transfer. Runner packaging/signing tools are CI dependencies, not Ghost FTP runtime components.

## Signing material

Code-signing certificates, Android publisher keystores, Apple Developer credentials and private keys are not runtime libraries and must never be committed to this repository.

Official Windows publication requires the protected trusted production Authenticode identity and `WINDOWS_AUTHENTICODE=signed`; there is no supported unsigned continuation. Android publication independently requires the protected production keystore and exact signer certificate SHA-256 match. macOS production distribution has its own fail-closed Developer ID + notarization boundary.

## Tracking prohibition

Ghost FTP does not add third-party runtime dependencies for telemetry, analytics, advertising, fingerprinting, session replay or automatic remote crash collection.

See [Dependencies](DEPENDENCIES.md), [Security](SECURITY.md), [Privacy](PRIVACY.md), [Signing](SIGNING.md) and [Platform parity](PLATFORM-PARITY.md).
