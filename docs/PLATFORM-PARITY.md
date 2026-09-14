# Windows and Linux platform parity

Ghost FTP **0.0.6** keeps Windows and Linux as one desktop product. Both frontends use the **same typed `internal/api.Engine`** and shared protocol, transfer, profile, settings, localization, Remote Edit and security layers.

Parity means equivalent supported behavior and protocol/security semantics with honest native-platform UX. A supported action must not exist as a decorative or dead control on either maintained desktop frontend.

## Shared protocol contract

Windows and Linux support FTP, explicit FTPS, SFTP password authentication, SFTP private-key authentication, key-passphrase handling, strict SFTP host-key fingerprint trust, local/remote navigation, transfer trees, remote file operations, connection timeouts and privacy-safe diagnostics.

Fresh Quick Connect resolves to explicit FTPS on port 21. Plain FTP is an explicit compatibility choice. Secure-transport failure is never silently retried through a weaker protocol.

Linux automatic SFTP password/private-key-passphrase delivery additionally requires the maintained root-controlled AskPass executable/helper/parent provenance; mutable Portable/per-user paths fail closed rather than weakening secret delivery.

## Shared settings and localization

Windows and Linux use the same typed profile/settings model. Saved credentials are opt-in and account-identity bound. Conflict policy, retry behavior, parallelism, timeout, appearance, language and independent upload/download bandwidth ceilings share one normalized contract.

Classic Light is the fresh/fallback appearance and Dark remains a maintained explicit choice. Ghost FTP ships one local **24-language** desktop registry with English default/fallback; no remote theme/font/localization service is used.

## Transfer and file-management parity

Both desktop platforms expose:

- local and remote panes with navigation/refresh;
- create, rename, delete and remote permission operations where supported;
- upload/download with pause, resume, cancel, retry and clear lifecycle;
- queued **Top / Up / Down / Bottom** priority ordering;
- truthful progress/speed/ETA and connection-generation ownership;
- sorting and current-folder filtering;
- bounded recursive search with cancellation;
- conservative directory comparison and synchronized navigation;
- bookmarks and profile start directories;
- built-in Remote Edit.

Bandwidth ceilings are aggregate directional budgets. FTP/FTPS map to curl rate enforcement and desktop SFTP to OpenSSH `sftp -l` with conservative unit conversion.

## Remote Edit parity

Windows and Linux use the same Remote Edit engine contract for FTP, FTPS and SFTP: bounded text size, strict UTF-8/binary validation, line-ending preservation, SHA-256 conflict detection, staged upload/read-back verification, trustworthy permission preservation and metadata refresh.

0.0.6 retains the Windows editor-session lifecycle guards added before this release: async open/save/reload work remains serialized and stale sessions cannot publish over a replacement session.

## Security parity

Both desktop frontends preserve strict FTPS certificate/hostname validation, strict SFTP host-key trust, no silent secure-to-plain downgrade, validated path boundaries, staged commit/rollback behavior, protected credentials and privacy-safe diagnostics. Windows async mutation/transfer completions are connection-generation bound; Linux adds the trusted AskPass provenance boundary.

## Windows-specific implementation

Windows uses native Win32 UI and DPI-aware layout. Official public Windows publication requires trusted Authenticode.

```text
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_BOOTSTRAP_PE=x86
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
WINDOWS_AUTHENTICODE=signed
```

Public Windows distribution exposes only:

```text
Ghost-FTP-0.0.6-Setup.exe
Ghost-FTP-0.0.6-Portable.exe
```

Verified x64/x86/ARM64 payloads remain internal. The public bootstrap selects the native embedded payload with `GetNativeSystemInfo`, verifies staged bytes and performs no runtime architecture download. Maintained CI structurally verifies ARM64 but does not claim native Windows ARM64 runtime execution.

## Linux-specific implementation

The canonical 0.0.6 Linux path is `linux/BUILD-DISTROS.sh`:

- Debian DEB: `amd64`, `arm64`, `i386`;
- Ubuntu DEB: `amd64`, `arm64`, `i386`;
- Fedora RPM: `x86_64`, `aarch64`, `i686`;
- Portable tar.gz: `amd64`, `arm64`, `i386`.

One production executable per Go architecture is reused across matching package families. Native install/remove/runtime/GUI evidence is **x86-64 only** on Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64; other architectures retain exact-head build, metadata, extraction and byte-parity evidence.

## Android public parity boundary

Android is a public 0.0.6 application, but parity is expressed through Android-native storage/lifecycle primitives rather than fake desktop controls. It exposes Files/Sites/Bookmarks/Transfers/Settings/About, FTP + strict explicit FTPS, SAF-scoped local storage, local/remote file management, sort/filter, bounded recursive search, directory comparison/synchronized navigation and protected Remote Edit behavior.

Android SFTP remains hidden until strict maintained host-key verification exists. The production-signed public APK does not change this security boundary.

## macOS source parity boundary

macOS is an active native development source surface using the shared engine and AppKit frontend. Universal development-app success is not Developer ID/notarization evidence. macOS remains outside the public 21-file release until the dedicated production signing/notarization/publication chain succeeds.

## Browser helper boundary

Chrome, Edge and Firefox helper ZIPs are public 0.0.6 companion packages built from one shared runtime. They remain local parser/copy helpers with no broad permissions and **no supported browser-to-desktop launch/handoff**. Browser packages are not a substitute for native application parity.

## Release parity

Every public platform stage is independently required: a successful Windows build cannot substitute for failed Linux or Android publication, and browser package failure blocks the canonical release.

The Ghost FTP 0.0.6 public contract is **18 platform artifacts / 21 public files** and publishes `ghcr.io/bren-wp/ghost-ftp:0.0.6` as a verified distribution bundle. macOS remains development/source only.

See [Architecture](ARCHITECTURE.md), [Installation](INSTALLATION.md), [Settings](SETTINGS.md), [Reference UI](REFERENCE-UI.md), [Testing](TESTING.md), [Signing](SIGNING.md) and [Security](SECURITY.md).
