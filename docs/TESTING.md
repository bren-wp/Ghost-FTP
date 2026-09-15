# Ghost FTP testing and quality gates

Ghost FTP **0.0.6** is validated through source, protocol/security, native build, packaging, signing, authentic runtime evidence and release-integrity gates. Published `ghostftp-v0.0.6` is release history; 0.0.7 cleanup work must not weaken these checks.

## Core quality gate

```text
gofmt
go test -race ./...
go vet ./...
```

Canonical CI also runs repository, platform, desktop-surface, dependency, version, localization, security, privacy, documentation and release audits plus the Python regression suite.

## Protocol and transfer regressions

Coverage protects strict FTPS certificate/hostname verification, no secure-to-plain downgrade, strict desktop SFTP host-key verification, local path confinement, staged activation/rollback, connection-generation guards, privacy-safe diagnostics, retry/cancel semantics, queue lifecycle and Remote Edit conflict/read-back behavior.

Expected network/server errors must remain application errors rather than crash paths. Authentication/server replies are untrusted input and must not expose credentials or protected payloads in user-facing messages.

## Android gate

`.github/workflows/android-apk.yml` validates Android source/security contracts, JVM tests, lint and APK construction/signing mechanics. Public 0.0.6 publication used the protected production keystore and exact `GHOSTFTP_ANDROID_CERT_SHA256` signer match. Android SFTP remains hidden until strict host-key verification exists.

## Browser package gate

`.github/workflows/browser-extensions.yml` validates and deterministically builds four packages:

```text
Ghost-FTP-0.0.6-Chrome-Extension.zip
Ghost-FTP-0.0.6-Edge-Extension.zip
Ghost-FTP-0.0.6-Firefox-Extension.zip
Ghost-FTP-0.0.6-Opera-Extension.zip
```

The 0.0.7 extension work must add tests for any local native-messaging protocol before enabling privileged behavior. Tests must verify manifest least privilege, strict message validation, no remote executable code, no telemetry/tracking, no credential persistence in browser storage and no unauthenticated localhost bridge.

## Windows gate

Ordinary Windows CI builds and verifies `Ghost-FTP-0.0.6-Setup.exe` and `Ghost-FTP-0.0.6-Portable.exe`. Native x64/x86/ARM64 staging payloads are PE/resource verified and embedded in the two public packages.

```text
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```

Production publication requires trusted Authenticode; development-only signing smoke material is not production evidence.

## Linux gate

Canonical Linux packaging produces six public bundles: Installer + Portable for Debian, Ubuntu and Fedora. Each bundle carries amd64/arm64/i386 payloads. Native lifecycle evidence is claimed only for environments that actually execute the package.

## macOS gate

The macOS workflow validates native source/build behavior. It is not Developer ID/notarization evidence and does not imply a public macOS release.

## Authentic runtime evidence

`.github/workflows/ui-screenshots.yml` captures exact-head runtime UI across Windows, Linux and Android. The read-only verifier assembles exactly 15 images and validates source SHA, file set, sizes and SHA-256 hashes. Generated mockups are not accepted as runtime evidence.

## Exact-head rule

A pull request is not merge-ready until every relevant workflow for its exact final head reaches terminal success. A green older SHA never satisfies a newer candidate. Tests, builds and security checks must not be disabled, marked allowed-failure or bypassed merely to obtain a green result.

## Published 0.0.6 release contract

The canonical published 0.0.6 set is **13 platform artifacts / 16 public files**. Release verification includes trusted Windows signing, production Android signing, deterministic browser packages, exact asset allow-list and digest/readback validation.

Future release preparation must use the canonical release workflow and must never rewrite the existing `ghostftp-v0.0.6` tag/release.

See [Security](SECURITY.md), [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md), [GitHub Releases](GITHUB-RELEASES.md) and [Versioning](VERSIONING.md).
