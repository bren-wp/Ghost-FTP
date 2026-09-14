# Ghost FTP architecture

Ghost FTP **0.0.6** uses one shared typed Go core with native Windows and Linux desktop frontends, a native Android client, a maintained macOS development/source frontend, and privacy-minimal browser helper packages. The canonical public release contains **18 platform artifacts / 21 public files**: two Windows executables, twelve Linux packages/archives, one production-signed Android APK, three deterministic browser ZIPs, and three metadata/verification files.

The architecture is built around explicit platform adapters, local-only persistent state, fail-closed protocol/security boundaries, bounded asynchronous work, and one canonical root `VERSION`.

## Release identity

The root `VERSION` file is authoritative. Release binaries and packages derive their version from it; development source keeps explicit development fallbacks instead of hard-coding a production version.

```text
ghostftp-vX.Y.Z
```

Current public identity is `ghostftp-v0.0.6`, Current channel, `prerelease=false`. A release branch must be `release/ghostftp-vX.Y.Z`, match `VERSION`, and point to the exact verified `main` commit before publication may start.

## Core layers

### `cmd/ghostftp`

Desktop application entry point. It initializes product identity, local state and the platform frontend while protocol behavior remains below the UI layer.

### `internal/api`

Typed application engine shared by maintained desktop frontends. It coordinates connection state, navigation, bookmarks/start directories, file operations, transfers, comparison/search/filter state and Remote Edit without exposing platform-specific UI details to protocol code.

### `internal/remote`

FTP/FTPS/SFTP implementation for maintained desktop surfaces. It owns command construction, external transfer-tool lifecycle, strict trust behavior, remote staging/commit semantics and privacy-safe diagnostic classification.

### `internal/transfer`

Transfer queue and lifecycle state: generations, status transitions, progress snapshots, Top/Up/Down/Bottom priority ordering, retry/cancel boundaries and stale-completion rejection.

### `internal/config`

Local settings/profile persistence, normalization and protected saved-secret references. Writes are bounded and replacement-oriented; credentials are opt-in and identity-bound.

### `internal/security`

Host/path validation, protected-secret handling, remote-path guards, symlink/reparse-aware local safety and SFTP fingerprint validation.

### `internal/desktop`

Native Windows and Linux frontends using the same typed engine. Platform rendering/input primitives may differ, but supported protocol, transfer, security, navigation, filtering/sorting/search/comparison, queue, bookmark and Remote Edit semantics remain shared.

## Windows architecture

Windows uses native Win32 UI and two public universal executables:

```text
Ghost-FTP-0.0.6-Setup.exe
Ghost-FTP-0.0.6-Portable.exe
```

Packaging builds verified native x64, x86 and ARM64 Setup/Portable payloads, embeds them into the two public bootstrap executables and removes staging payload source. `GetNativeSystemInfo` selects the matching embedded payload at runtime; no architecture payload is downloaded.

```text
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_BOOTSTRAP_PE=x86
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
WINDOWS_AUTHENTICODE=signed
```

Official public Windows publication requires trusted Authenticode. Ordinary CI may build unsigned engineering artifacts, but the public workflow has no unsigned continuation and never manufactures a self-signed production identity. `WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci` remains explicit because maintained Windows CI does not provide native ARM64 execution evidence.

## Linux architecture

Linux uses the maintained native X11/XWayland-compatible frontend. `linux/BUILD-DISTROS.sh` builds Debian and Ubuntu DEBs for amd64/arm64/i386, Fedora RPMs for x86_64/aarch64/i686, and Portable tarballs for amd64/arm64/i386.

One verified production executable per architecture is reused across matching package variants and byte-parity checked. Native package-manager/runtime/GUI evidence is maintained on Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64. Cross-built architectures retain exact-head build, metadata, extraction and parity evidence without a false native-runtime claim.

Linux trusted automatic SFTP secret delivery remains constrained to the root-controlled AskPass executable/helper/parent provenance boundary.

## Android architecture

Android 0.0.6 is a public native application through the protected production-signing path:

```text
Ghost-FTP-0.0.6-Android.apk
```

The client uses Android 35, Java, Storage Access Framework document access, strict explicit FTPS, bounded FTP parsing, staged upload/download semantics, lifecycle generation ownership and local-only profile/navigation state. File management, sorting/filtering, bounded recursive search, directory comparison/synchronized navigation and protected Remote Edit behavior are maintained under Android-specific lifecycle/storage primitives.

Publication requires a protected keystore and exact signer-certificate SHA-256 verification. Ordinary CI produces only `Ghost-FTP-Android-dev.apk` and may use an ephemeral identity for pipeline smoke evidence.

Android SFTP remains hidden until strict maintained host-key verification exists. Production signing does not weaken that boundary.

## Browser helper architecture

Chrome, Microsoft Edge and Firefox packages are built from one canonical shared runtime plus browser-specific MV3 manifests. The official 0.0.6 ZIPs are deterministic and brand-contract verified.

Browser helpers parse/copy supported FTP-family targets locally. They request no broad host/tab/history/storage/network permissions and have **no supported browser-to-desktop launch/handoff**. They are public companion packages, not application runtime transports.

## macOS client

The macOS source surface remains an active native AppKit development frontend connected to the shared engine. `.github/workflows/macos-app.yml` builds and verifies the universal development app.

Development evidence is not public production evidence. macOS stays outside the 21-file public release until a real Developer ID Application signing and Apple notarization/publication chain succeeds with protected credentials.

## Connection and trust architecture

Transport selection is explicit. FTPS is the secure desktop Quick Connect default; plain FTP is an explicit compatibility choice; desktop SFTP uses strict host-key trust/pinning. Secure transport failure is never silently downgraded.

Connection/session generations invalidate stale asynchronous callbacks after cancel, disconnect or reconnect. User-facing errors pass through privacy-safe classification so passwords, passphrases and protected credential payloads are not exposed.

## Transfer and Remote Edit integrity

Important invariants include:

- local path confinement before mutation;
- bounded/cancellable recursive work;
- upload source snapshots and staged final-name activation;
- cancellation checks before commit;
- symlink/reparse-aware destructive cleanup;
- connection-generation ownership for async completions;
- truthful progress/speed/ETA;
- Remote Edit UTF-8/binary/size validation, line-ending preservation, SHA-256 conflict checks, staged upload/read-back verification and trustworthy permission preservation.

## State and privacy architecture

Profiles, settings, bookmarks and protected credentials remain local. Ghost FTP has no product account backend, application telemetry pipeline, ads or hidden relay service. Production/CI workflows explicitly disable Go telemetry where Go executes.

The maintained Go module has no external module requirements; release/CI Go builds use `GOPROXY=off` and `GOSUMDB=off` after toolchain setup.

## Authentic evidence architecture

`.github/workflows/ui-screenshots.yml` captures real exact-head Windows, Linux and Android runtime surfaces and assembles a read-only verified 15-image evidence bundle with source SHA and SHA-256 checks. Mockups, image-generation output and manually composed approximations are not production runtime evidence.

macOS development validation remains separate and must not be relabeled as notarized public distribution evidence.

## Release architecture

The canonical publication sequence is:

1. exact-head formatting/race/vet plus repository, platform, dependency, version, localization, security, privacy, docs and release audits;
2. full Python regression suite and protocol/lifecycle contracts;
3. Windows universal x64/x86/ARM64 packaging with required trusted Authenticode;
4. twelve canonical Linux artifacts with metadata/binary-parity validation;
5. production Android build, protected signing, `apksigner` verification and signer-fingerprint pin;
6. deterministic Chrome/Edge/Firefox helper builds;
7. exact 18-product-artifact / 21-file allow-list assembly and SHA-256 manifest;
8. non-prerelease GitHub Release and exact remote digest readback;
9. exact-version GHCR distribution-bundle publication/readback;
10. latest-only retention only after publication/integrity success.

The GHCR object is a distribution bundle, not a supported runtime container.

## Supported production boundary

Ghost FTP 0.0.6 publicly releases Windows, Linux and Android plus official Chrome/Edge/Firefox helper ZIPs. macOS remains development/source only. Android SFTP remains unavailable pending strict host-key verification, and browser helpers do not provide a desktop handoff.

See [Security](SECURITY.md), [Privacy](PRIVACY.md), [Platform parity](PLATFORM-PARITY.md), [Signing](SIGNING.md), [Packages](PACKAGES.md) and [Release verification](RELEASE-VERIFICATION.md).
