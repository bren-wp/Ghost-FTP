# Ghost FTP testing and quality gates

Ghost FTP **0.0.7** is validated through layered source, protocol/security, native build, packaging, signing, authentic runtime evidence, exact-release readback and retention gates.

## Core quality gate

```text
gofmt
go test -race ./...
go vet ./...
```

Canonical CI also runs repository, platform, desktop-surface, dependency, version, localization, security, privacy, documentation and release audits plus the complete Python regression suite.

## Protocol and transfer regressions

Coverage protects strict FTPS certificate/hostname verification, no silent secure-to-plain downgrade, strict desktop SFTP host-key verification/pinning, rooted local path confinement, staged activation/rollback, connection-generation guards, privacy-safe diagnostics, retry policy, transfer queue lifecycle, queued Top/Up/Down/Bottom ordering, Remote Edit conflict/read-back behavior and bookmark/start-directory revalidation.

Authentication-error redaction is a maintained privacy contract: server-controlled authentication replies and external-tool diagnostics must not expose credentials or protected secret payloads in user-facing errors.

## Bandwidth regression contract

Upload/download ceilings are real runtime policy. Tests require bounded `0–1,048,576 KiB/s` values with `0 = unlimited`, conservative aggregate directional allocation, attempt-scoped snapshots, curl rate enforcement for FTP/FTPS and OpenSSH `sftp -l` enforcement for desktop SFTP.

## Current-folder filter and sorting regression contract

Current-folder filtering operates only on the loaded snapshot and performs no hidden filesystem/network scan. It is **deliberately separate from bounded recursive search**. Sorting is directories-first, **filtering and subsequent sorting** operate over copies of authoritative loaded snapshots, and row-indexed actions remain bound to the visible authoritative slice.

## Bounded recursive search regression contract

Bounded recursive search is explicit I/O with cancellation and depth/item/result/batch/time ceilings. Search result activation requires a fresh parent listing before selection becomes authoritative. Directory comparison uses conservative `same/local_only/remote_only/newer_local/newer_remote/conflict/unknown` semantics and synchronized navigation only for safely proven paired ordinary directories.

## Desktop lifecycle and action wiring

Visible controls require matching command/click handlers and code-level state guards. Regression coverage includes profile-persistence ownership, Windows nested modal `WM_QUIT` preservation, local/remote mutation re-entry guards, serialized Remote Edit open/save/reload, transfer-generation ownership across reconnect, Linux modal input isolation and queue-priority state.

## Android source, APK and release-signing gate

`.github/workflows/android-apk.yml` runs Android source/security contracts, JVM tests, `lintDebug`, `lintRelease`, development APK construction and unsigned release construction. Ordinary CI uses an **ephemeral CI-only** signing identity solely to exercise `apksigner sign` + `verify`; it is never public publisher evidence.

Android contracts protect strict explicit FTPS, no trust-all fallback, SAF-only storage, staged transfer commit, bounded parsing/search, lifecycle cancellation/generation ownership, authentication-error redaction, file-management path validation, comparison/search semantics and Remote Edit conflict/read-back safeguards.

The canonical public release additionally requires a protected production keystore and exact signer-certificate SHA-256 match before `Ghost-FTP-0.0.7-Android.apk` can enter the allow-list. Android SFTP remains hidden until strict maintained host-key verification exists.

## Browser package gate

`.github/workflows/browser-extensions.yml` validates the canonical brand/privacy contract and deterministically builds exactly three packages:

```text
Ghost-FTP-0.0.7-Chrome-Extension.zip
Ghost-FTP-0.0.7-Edge-Extension.zip
Ghost-FTP-0.0.7-Firefox-Extension.zip
```

The package contract rejects brand drift, manifest version drift, broad permissions, credential persistence, remote executable code and any unsupported desktop launch/handoff behavior.

## macOS development-app gate

`.github/workflows/macos-app.yml` — **Ghost FTP macOS Development App** — builds and validates the maintained universal native development frontend. This is source/build evidence only; it is not Developer ID-signed/notarized public-distribution evidence and does not enlarge the 21-file release.

## Windows build and public signing gates

Ordinary Windows CI builds and verifies `Ghost-FTP-0.0.7-Setup.exe` and `Ghost-FTP-0.0.7-Portable.exe`. Internally, native x64/x86/ARM64 staging pairs are PE/resource verified and embedded in those two universal public files; architecture-specific executables are forbidden from leaking publicly.

```text
WINDOWS_SETUP=universal-x86-x64-arm64
WINDOWS_PORTABLE=universal-x86-x64-arm64
WINDOWS_BOOTSTRAP_PE=x86
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```

Ordinary CI exercises an Authenticode private-key pipeline smoke with development-only material. Official `Publish Ghost FTP` is stricter: protected production signing is mandatory, `Get-AuthenticodeSignature` must be valid and `WINDOWS_SIGNING_STATE=signed` is required. Cross-build and structural ARM64 verification are not native ARM64 runtime execution.

## Linux production and distro gates

Canonical Linux packaging is `linux/BUILD-DISTROS.sh` and `.github/workflows/linux-distro-packages.yml`, producing twelve Debian/Ubuntu/Fedora/Portable artifacts with metadata, extraction and byte-parity checks.

`.github/workflows/linux-distro-install.yml` verifies native installation/runtime/removal on Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64. Native lifecycle evidence is deliberately x86-64 only; additional architectures retain exact-head build/metadata/extraction/parity coverage.

## Authentic runtime evidence

`.github/workflows/ui-screenshots.yml` captures exact-head real runtime UI: Windows — Main Workspace, Site Manager, Bookmarks, Settings, About; Linux — Main Workspace, Bookmarks, Settings; Android — Files, Navigation, Sites, Bookmarks, Transfers, Settings, About.

The final read-only verifier assembles exactly 15 images into the immutable evidence bundle and validates source SHA, file set, sizes and SHA-256 hashes. Generated mockups are not accepted as runtime evidence.

## Exact-head and post-merge rule

**Exact-head and post-merge rule:** a PR is not merge-ready until every workflow triggered for its exact final head is `completed/success`. After merge, required push workflows are identified by the exact merge SHA and must also finish `completed/success` before release preparation continues. A green older SHA never satisfies a newer candidate.

For 0.0.6, the broad gate set includes Ghost FTP CI, Android APK, Browser Extensions, macOS Development App, Linux Distro Packages, Linux Distro Install Matrix, Windows Modal Keyboard Runtime, Govulncheck, CodeQL and Authentic Cross-Platform UI Screenshots whenever path filters trigger them.

## Release publication gate

0.0.6 publication requires exact current `main` release-branch validation, canonical quality/build jobs, trusted Authenticode on both public Windows executables, production Android signing plus exact signer SHA-256 verification, deterministic Chrome/Edge/Firefox packages, exact **18 platform artifacts / 21 public files** allow-list, `prerelease=false`, exact GitHub Release/SHA-256 readback, verified `ghcr.io/bren-wp/ghost-ftp:0.0.7` distribution bundle and successful release-integrity/latest-only retention chains.

## Deterministic release-to-retention gate

The release-branch trigger records prior run IDs, dispatches canonical publication, identifies the newly created exact-main run, waits for terminal success, then dispatches retention and requires that exact retention run to succeed. Dispatch acknowledgement alone is not publication proof.

## Retention validation

Retention must leave only the current `ghostftp-v0.0.7` public release/tag, retain the current canonical release branch and exact-version GHCR package, remove superseded Ghost FTP release/tag/branch/package identities, and leave `main` history untouched.

See [Security](SECURITY.md), [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md), [GitHub Releases](GITHUB-RELEASES.md) and [Versioning](VERSIONING.md).
