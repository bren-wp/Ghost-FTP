# Ghost FTP testing and quality gates

Ghost FTP **0.0.5** is validated through layered source, security, native build, packaging, Android APK, UI-action, authentic runtime evidence and release-lifecycle gates.

## Core quality gate

```text
gofmt
go test -race ./...
go vet ./...
```

Canonical CI additionally runs repository, platform, desktop-surface, dependency, version, localization, security, privacy, documentation and release audits plus the complete Python regression suite.

## Protocol and transfer regressions

Coverage protects explicit FTPS verification/no silent downgrade, strict SFTP host-key verification/pinning, rooted local path confinement, staged activation/rollback, connection-generation guards, privacy-safe diagnostics, retry policy, transfer queue lifecycle, queued Top/Up/Down/Bottom ordering, Remote Edit conflict/read-back behavior and navigation bookmark/profile-start revalidation.

## Bandwidth regression contract

The maintained source provides independent upload/download ceilings as real runtime policy. Tests require bounded `0–1,048,576 KiB/s` values with `0 = unlimited`, conservative aggregate directional allocation, attempt-scoped budget snapshots, curl `limit-rate` enforcement for FTP/FTPS and OpenSSH `sftp -l` enforcement for SFTP. Windows/Linux settings must expose the same shared values.

## Filtering, sorting, recursive search and comparison

### Current-folder filter and sorting regression contract

Current-folder filtering is non-destructive over loaded snapshots and performs no hidden scan. The current-folder filter is deliberately separate from bounded recursive search: it operates only on entries already loaded in the pane and performs no additional filesystem or network scan. Regression coverage verifies filtering and subsequent sorting over copies of the authoritative snapshot, directories-first ordering, visible-slice action binding and selection restoration.

### Bounded recursive search regression contract

Bounded recursive search is an explicit I/O-producing action with cancellation, maintained depth/item/result/batch/time ceilings and fresh-list navigation from a result. It reuses matching semantics without turning the instant loaded-snapshot filter into a hidden recursive scan.

Directory comparison uses conservative `same/local_only/remote_only/newer_local/newer_remote/conflict/unknown` semantics and synchronized navigation only for safely proven paired ordinary directories.

## Settings regression contract

Tests cover parallelism, independent bandwidth ceilings, timeout/retry bounds, conflict-policy normalization, Light/Dark appearance and Linux credential-save confirmation while preserving safe legacy migration behavior.

## Desktop action wiring and Windows lifecycle contracts

Visible main controls must have matching command/click handlers. 0.0.5 adds regression contracts for:

- encrypted Windows profile mutation ownership and close blocking while persistence is in flight;
- shared/nested Windows modal preservation of `WM_QUIT`;
- local/remote create-directory, rename, delete and remote permission mutation re-entry guards;
- Remote Edit session serialization across async open/save/reload cycles;
- code-level guards behind UI enablement so stale commands cannot bypass busy state.

## Android native source and APK gate

`.github/workflows/android-apk.yml` requires source/security contracts, Java/SDK/Gradle setup, Android lint, installable APK build, APK identity verification and artifact upload.

Android contracts protect strict explicit FTPS certificate/hostname verification, no trust-all fallback, SAF-only local storage, non-secret saved-site metadata, staged transfer final-name commit, non-blocking cancellation, semantic navigation and bounded FTP parsing.

0.0.5 additionally tests that a pending FTP/FTPS connection is owned by the current Activity instance, `onDestroy()` aborts it non-blockingly, and stale success/error callbacks cannot commit a session/UI state after destruction/recreation.

## Browser companion source contract

The optional browser companion source for Chrome, Microsoft Edge, Opera, Brave, Vivaldi and Firefox is tested as a source/privacy contract: supported FTP-family schemes are handled locally without telemetry, remote executable code, credential persistence, tab scraping or broad host permissions. These companions do not enlarge the 17-file desktop release allow-list.

## Windows production gate

The Windows production job builds and verifies:

```text
Ghost-FTP-0.0.5-Setup.exe
Ghost-FTP-0.0.5-Portable.exe
```

Native x64/x86 payloads are built and verified internally, then embedded in the two public universal bootstraps. CI rejects architecture-specific public executables and exercises optional Authenticode policy.

## Linux production and distro package gates

The regular Core CI retains generic Linux compatibility builds. Canonical release packaging is `linux/BUILD-DISTROS.sh` via `.github/workflows/linux-distro-packages.yml`, with Debian/Ubuntu/Fedora/Portable package metadata/extraction/binary-parity checks.

The 0.0.5 canonical set remains **14 platform artifacts / 17 public files**.

`.github/workflows/linux-distro-install.yml` verifies native installation lifecycle on **Debian 13 amd64**, **Ubuntu 26.04 LTS amd64** and **Fedora 44 x86_64**. **Native package-manager/runtime coverage is deliberately limited to x86-64.** Additional canonical architectures retain exact-head build/metadata/extraction/parity coverage.

## Authentic UI evidence

`.github/workflows/ui-screenshots.yml` captures real exact-head runtime UI:

- Windows: Main Workspace, Site Manager, Bookmarks, Settings, About;
- Linux: Main Workspace, Bookmarks, Settings;
- Android: Files, Navigation, Sites, Bookmarks, Transfers, Settings, About.

The final read-only evidence job verifies provenance, manifest and hashes and assembles the 15-image `ghostftp-authentic-ui-verified-bundle`. It never commits or pushes screenshots back to the tested branch.

## Exact-head and post-merge rule

**Exact-head and post-merge rule:** a PR is not merge-ready until every required workflow actually triggered for its exact final head is `completed/success`. After merge, required `push` workflows are identified by the exact merge SHA and must also be `completed/success` before release preparation continues.

For a 0.0.5 release-prep change, expected broad gates include:

1. Ghost FTP CI;
2. Ghost FTP Android APK when its path filters trigger;
3. Ghost FTP Linux Distro Packages;
4. Ghost FTP Linux Distro Install Matrix;
5. Ghost FTP Authentic Cross-Platform UI Screenshots;
6. any additional path-triggered Windows runtime gate.

A green run for an older commit does not satisfy a newer candidate.

## Release publication gate

0.0.5 publication additionally requires:

- exact current `main` release-branch validation;
- canonical release workflow quality/build jobs;
- exact **17-file** GitHub Release allow-list;
- immediate and delayed remote release read-back;
- `prerelease=false`;
- verified `ghcr.io/bren-wp/ghost-ftp:0.0.5` distribution-bundle publication/read-back;
- successful latest-only retention cleanup.

The Android development APK and browser companion source remain outside the 17-file public desktop release allow-list.

## Deterministic release-to-retention gate

The release branch trigger must record prior run IDs, dispatch canonical publication, identify the newly created exact-main run, wait for success, only then dispatch retention, identify the exact new retention run and require terminal success. Token-dispatch acknowledgement alone is not publication proof.

## Retention validation

Retention must leave only the current `ghostftp-v0.0.5` public release/tag, retain the current canonical release branch and exact-version GHCR package, remove superseded release branches/package versions, and leave `main` history untouched.

See [Security](SECURITY.md), [Release verification](RELEASE-VERIFICATION.md), [GitHub Releases](GITHUB-RELEASES.md) and [Versioning](VERSIONING.md).
