# Ghost FTP architecture

Ghost FTP **0.0.6** is a multi-platform file-transfer product built around a shared typed Go desktop engine, native Windows and Linux frontends, a native Android application, a native macOS source surface, and four privacy-minimal browser helper packages.

The retired repository website and Web FTP runtime are not part of the supported product architecture and are intentionally absent from the source tree. The root `VERSION` file remains authoritative and stays at **0.0.6**.

## Release identity

The current candidate identity is `ghostftp-v0.0.6`, Current channel, `prerelease=false`. Publication must originate from the exact verified `main` commit and must not rewrite an existing tag or release.

The canonical public release contains **13 platform artifacts / 16 public files**:

- Windows: Setup and Portable.
- Linux: Installer and Portable bundles for Debian, Ubuntu and Fedora.
- Android: one production-signed APK.
- Browser helpers: Chrome, Edge, Firefox and Opera packages.
- Release integrity files required by the publication contract.

macOS remains an active native source surface. It is not advertised as a public release artifact until the maintained release pipeline has real Developer ID signing, Apple notarization, stapling validation and Gatekeeper verification.

## Shared desktop engine

- `internal/api` — typed application engine and connection, navigation and file-operation orchestration.
- `internal/remote` — FTP, FTPS and SFTP protocol execution, trust boundaries and remote staging semantics.
- `internal/transfer` — transfer queue lifecycle, progress, retry, cancellation and priority ordering.
- `internal/config` — profiles, settings and protected saved-secret references.
- `internal/security` — host, path, fingerprint and secret validation plus destructive-operation guards.
- `internal/desktop` — native Windows and Linux frontends over the shared engine.

The desktop runtime has no product telemetry, advertising SDK, mandatory Ghost FTP account, hidden relay service or Ghost FTP cloud credential store.

## Windows

Windows is the reference desktop UI. Public 0.0.6 output is exactly:

```text
Ghost-FTP-0.0.6-Setup.exe
Ghost-FTP-0.0.6-Portable.exe
```

Each user-facing package embeds verified x64, x86 and ARM64 payloads. The bootstrap selects the local payload from native system architecture information and performs no architecture download. Official publication requires trusted Authenticode signing; isolated CI smoke identities never substitute for production signing.

Canonical Windows evidence remains explicit:

```text
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```

The ARM64 marker is intentionally conservative: cross-build, resource and package verification are not represented as native ARM64 runtime execution when maintained Windows CI does not run natively on ARM64.

Installed Windows builds also own the `ghostftp:` custom protocol registration used by the official browser helpers. Setup registers an exact quoted command for the installed executable and uninstall removes the handler only when ownership can still be proven. Browser launch metadata is parsed through an allowlist before it reaches native connection controls; credentials and private-key material are outside this contract.

## Linux

`linux/BUILD-DISTROS.sh` builds amd64, arm64 and i386 payloads. Those payloads are bundled into exactly six user-facing 0.0.6 files: Installer and Portable for Debian, Ubuntu and Fedora. Each bundle selects the matching payload locally.

Native installer, runtime and GUI evidence is maintained on supported amd64 runners. ARM64 and i386 remain cross-build/package evidence unless native runtime evidence is available.

## Android

Android publishes one production-signed 0.0.6 APK. The client uses Android Storage Access Framework capabilities, FTP plus strict explicit FTPS, bounded parsing, staged transfer semantics and lifecycle generation ownership.

Android SFTP is not exposed as a supported public capability until strict maintained host-key verification is implemented. Release validation is fail-closed: source contracts, tests, lint, APK structure and signing identity must pass before publication.

## macOS

The AppKit frontend is an active native source surface over the shared engine. It is maintained as production-quality source, but it is not represented as a public macOS distribution until the release pipeline has real Developer ID Application signing, Apple notarization, stapling and Gatekeeper assessment.

CI validation artifacts are engineering evidence only and are never presented as public production-signed packages.

## Browser helpers

Chrome, Edge, Firefox and Opera packages are built from one shared local runtime plus browser-specific manifests. They request zero browser permissions and zero host permissions and perform local parsing only. On supported installed Windows builds, **Open in Ghost FTP** creates a sanitized `ghostftp://connect` launch request containing only protocol, host, optional port, optional username and optional remote path. Passwords, passphrases, private keys, source query values and fragments are never included, and the handoff does not auto-connect or introduce a hidden network relay.

The helpers are release packages, not a replacement website and not a Web FTP client.

## Transfer and Remote Edit integrity

Core invariants include bounded and cancellable work, source/destination identity checks, staged activation, cleanup on failure, connection-generation ownership, truthful progress and bounded UTF-8/binary-safe Remote Edit behavior.

User-facing errors are sanitized so internal application paths, private storage locations, credentials and raw transport diagnostics are not exposed.

## State and privacy

Profiles, settings and bookmarks stay local to the relevant application platform. Ghost FTP has no product analytics backend. Credentials are not written to arbitrary runtime files, and platform-specific secret handling must follow the maintained security contract.

Runtime transport connections are initiated for the destination explicitly configured by the user. There is no retired web proxy or Web FTP server in the supported source tree.

## Release architecture

The 0.0.6 publication sequence is:

1. exact-head Go format, race tests and vet plus repository/platform/security/privacy/documentation/release audits;
2. maintained regression and platform contract tests;
3. trusted-signed Windows universal Setup and Portable packages;
4. six Linux universal distro bundles;
5. one production-signed Android APK with exact certificate fingerprint verification;
6. four deterministic browser helper ZIPs;
7. exact **13 platform artifacts / 16 public files** allow-list assembly and SHA-256 manifest;
8. non-prerelease GitHub Release creation and exact remote asset/digest readback;
9. exact-version GHCR distribution-bundle publication and readback;
10. latest-only cleanup only after integrity success.

See [Security](SECURITY.md), [Privacy](PRIVACY.md), [Platform parity](PLATFORM-PARITY.md), [Signing](SIGNING.md) and [Release verification](RELEASE-VERIFICATION.md).
