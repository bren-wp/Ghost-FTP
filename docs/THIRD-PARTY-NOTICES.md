# Third-party notices

Ghost FTP **0.0.5** keeps the maintained desktop/core Go module intentionally free of external Go module requirements. The shared desktop engine uses operating-system networking tools for protocol execution rather than bundling an untracked third-party FTP/SSH stack.

The current public release platforms are Windows and Linux. macOS is an active native development/source frontend using the same shared `internal/api.Engine`; Android and the browser connection helper use their own platform/runtime facilities and remain outside the current 17-file public desktop release allow-list.

## Desktop runtime transport tools

### FTP / FTPS

The shared desktop transport uses an operating-system `curl` executable for FTP and explicit FTPS operations on maintained desktop platforms.

`curl` is distributed and licensed by its respective operating-system/vendor package source. Ghost FTP does not claim ownership of curl and does not silently download a private bundled copy at runtime.

Security-sensitive executable provenance is platform-specific. Linux applies the documented trusted root-controlled transport-tool boundary before credential-bearing automation is allowed. Other desktop platforms retain their documented system-tool and transport validation contracts.

### SFTP

The shared desktop SFTP transport uses operating-system OpenSSH `ssh` / `sftp` tooling.

On Windows this normally means the Windows OpenSSH Client capability. On Linux it normally comes from the distribution's OpenSSH client package. The macOS development frontend consumes the same shared engine rather than adding a second FTP/SFTP protocol stack.

OpenSSH is distributed and licensed by its respective operating-system/vendor package source. Ghost FTP constrains child-process configuration, trust and credential handoff but does not relicense or impersonate OpenSSH.

## Linux package prerequisites

Canonical Debian/Ubuntu packages declare maintained runtime prerequisites including:

- `ca-certificates`;
- `curl`;
- `openssh-client`.

Canonical Fedora packages declare the corresponding `ca-certificates`, `curl` and `openssh-clients` requirements. Portable archives intentionally do not vendor private copies of those networking tools or CA stores.

These are operating-system packages, not vendored Go dependencies.

## macOS development dependency boundary

The native AppKit development frontend is built from repository source and uses the shared Go engine. AppKit, Security.framework/Keychain integration and Apple platform tooling are operating-system/developer-platform facilities rather than third-party Go modules bundled into the application.

The maintained development artifact is ad-hoc signed for development validation. A future public macOS distribution additionally requires the real Developer ID + Apple notarization path described in [`../macos/README.md`](../macos/README.md) and [`SIGNING.md`](SIGNING.md); availability of source or a development build is not proof that production signing/notarization succeeded.

## Android development dependency boundary

Android is an active development APK surface, not a current public release platform. Its Java/Android SDK and Gradle ecosystem are build/platform dependencies for the mobile client and do not alter the root Go module's zero-external-module contract.

Android uses platform networking and Storage Access Framework capabilities. The current client exposes FTP and strict explicit FTPS; SFTP remains hidden until verified native host-key identity support exists. Android CI may use test-only dependencies such as JUnit without turning them into a Ghost FTP production telemetry/networking service.

## Browser connection helper

`ekstenzije/` is a privacy-minimal Manifest V3 source companion for supported Chromium-family browsers and Firefox. It uses browser-provided extension APIs and local JavaScript parsing; it does not bundle a separate FTP/SFTP networking stack, does not connect to transfer servers and does not provide a supported browser-to-desktop URI/native-messaging handoff today.

The helper requests no broad host, tab/history, storage, scripting or network permissions. It is source/development material outside the current Windows/Linux GitHub Release allow-list.

## GitHub Actions

Build workflows use pinned GitHub Actions revisions for repository checkout, Go/Python setup and artifact upload/download. Those actions are build-system dependencies executed by GitHub Actions; they are not installed runtime components of Ghost FTP.

The current release pipeline also uses runner-provided packaging/container tooling where explicitly documented. Such CI tooling is not silently bundled into Ghost FTP runtime packages.

## Signing material

Code-signing certificates, Apple Developer credentials and private keys are not runtime libraries and must never be committed to this repository.

Official public Windows publication requires the protected trusted production Authenticode identity and valid signatures on both public Windows executables. A successful public release records:

```text
WINDOWS_AUTHENTICODE=signed
```

There is no supported unsigned continuation for the official `Publish Ghost FTP` workflow. Local/development and ordinary CI Windows builds may remain unsigned because they are not official public release artifacts.

The development self-signed Windows signing helper exists only to validate signing mechanics. It does not create a trusted production publisher identity and its generated PFX/CER files are not source or public-release artifacts.

macOS production distribution has a separate fail-closed Developer ID + notarization boundary. Ghost FTP does not substitute a generated/self-signed production identity or claim notarization success from an ad-hoc development build.

## Identity and historical provenance

Historical Git history can contain retired source paths, dependency assumptions or platform artifacts that matched their original release line. Historical provenance is not the active 0.0.5 dependency contract and must not be copied into current guidance as an active surface.

Current public product identity is **Ghost FTP**. Internal compatibility identifiers containing `GhostFTP` are retained only where changing them could break installed identity or upgrade behavior; they do not represent a separate current product.

See [Dependencies](DEPENDENCIES.md), [Security](SECURITY.md), [Privacy](PRIVACY.md), [Signing](SIGNING.md) and [Platform parity](PLATFORM-PARITY.md).
