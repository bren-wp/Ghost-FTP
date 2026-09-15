# Ghost FTP testing and quality gates

Ghost FTP **0.0.6** is validated through layered source, protocol/security, native build, packaging, signing, runtime-evidence and release-readback gates.

## Core quality gate

```text
gofmt
go test -race ./...
go vet ./...
```

Canonical CI also runs repository, platform, desktop-surface, dependency, version, localization, security, privacy, documentation and release audits plus the complete Python regression suite.

## Protocol and transfer regressions

Coverage protects strict FTPS certificate/hostname verification, no silent secure-to-plain downgrade, strict desktop SFTP host-key verification/pinning, rooted local path confinement, staged activation/rollback, connection-generation guards, privacy-safe diagnostics, retry policy, transfer queue lifecycle, Remote Edit conflict/read-back behavior and bookmark/start-directory revalidation.

Authentication-error redaction is a maintained privacy contract: server-controlled replies and external-tool diagnostics must not expose credentials or protected secret payloads in user-facing errors.

## Desktop lifecycle and action wiring

Visible controls require matching command/click handlers and state guards. Regression coverage includes profile-persistence ownership, modal-loop behavior, local/remote mutation re-entry guards, Remote Edit serialization, transfer-generation ownership across reconnect and queue-priority state.

## Android source, APK and release-signing gate

`.github/workflows/android-apk.yml` validates Android source/security contracts, JVM tests, lint and APK construction. CI-only signing identities may exercise signing mechanics but are never public publisher evidence.

Android contracts protect strict explicit FTPS, no trust-all fallback, SAF-only storage, staged transfer commit, bounded parsing/search, lifecycle cancellation/generation ownership, authentication-error redaction, file-management validation and Remote Edit safeguards.

The public release additionally requires the protected production keystore and an exact `GHOSTFTP_ANDROID_CERT_SHA256` signer-certificate match before `Ghost-FTP-0.0.6-Android.apk` can enter the release allow-list. Android SFTP remains hidden until strict maintained host-key verification exists.

## Browser package gate

Browser workflows validate brand/privacy contracts and build four deterministic packages:

```text
Ghost-FTP-0.0.6-Chrome-Extension.zip
Ghost-FTP-0.0.6-Edge-Extension.zip
Ghost-FTP-0.0.6-Firefox-Extension.zip
Ghost-FTP-0.0.6-Opera-Extension.zip
```

The package contract rejects brand drift, manifest version drift, unnecessary permissions, credential persistence, remote executable code and unsupported desktop launch/handoff behavior.

## macOS development-app gate

The maintained macOS workflow builds and validates the native development frontend. This is source/build evidence only; it is not Developer ID-signed/notarized public-distribution evidence and does not enlarge the public release contract.

## Windows build and signing gates

Ordinary Windows CI builds and verifies `Ghost-FTP-0.0.6-Setup.exe` and `Ghost-FTP-0.0.6-Portable.exe`. Internally, x64/x86/ARM64 payloads are verified and embedded in those two universal public executables; architecture-specific executables must not leak into the public artifact set.

Official publication is stricter than CI smoke signing: protected production Authenticode signing is mandatory and the resulting signatures must validate.

## Linux production and distro gates

Canonical Linux packaging produces Installer and Portable bundles for Debian, Ubuntu and Fedora. Native runtime evidence and cross-built architecture evidence must remain clearly distinguished.

## Authentic runtime evidence

UI screenshot workflows capture real application surfaces for Windows, Linux and Android. Repository evidence must remain tied to the source SHA. Generated mockups are not accepted as runtime evidence.

## Exact-head and post-merge rule

A candidate is not merge-ready until workflows triggered for its exact final head complete successfully. A green older SHA never proves a newer source candidate.

## Release publication gate

0.0.6 publication requires canonical quality/build jobs, trusted Authenticode for both public Windows executables, production Android signing with exact signer SHA-256 verification, deterministic Chrome/Edge/Firefox/Opera packages and the exact **13 platform artifacts / 16 public files** allow-list before release publication/readback succeeds.

## Retired-source guard

The documentation audit also verifies that retired repository application surfaces remain removed. Reintroducing obsolete application directories, workflows, contract tests or documentation without an explicit architecture decision is treated as a regression.

See [Security](SECURITY.md), [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md), [GitHub Releases](GITHUB-RELEASES.md) and [Versioning](VERSIONING.md).
