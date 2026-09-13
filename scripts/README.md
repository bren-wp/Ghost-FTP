# Ghost FTP build and audit scripts

This directory contains the maintained build, packaging, security, privacy, documentation and release-verification tooling used by Ghost FTP **0.0.5**.

The scripts exist to keep source, platform behavior, public documentation and release artifacts bound to one explicit contract. A successful compile alone is not sufficient release evidence.

## Canonical public release path

GitHub Releases are assembled only by `.github/workflows/release.yml`. The canonical user-facing public release is identified by:

```text
VERSION=0.0.5
TAG=ghostftp-v0.0.5
PRERELEASE=false
PUBLIC_PLATFORM_ARTIFACTS=14
PUBLIC_RELEASE_FILES=17
```

The current public allow-list contains Windows and Linux artifacts only. Android and macOS remain active development/source surfaces with separate native build gates and are not silently added to the 17-file public release set.

The same verified `release/` assembly is also published as an OCI **distribution bundle** to:

```text
ghcr.io/bren-wp/ghost-ftp:0.0.5
```

The GHCR object is distribution infrastructure, not a runtime application container and not a second independent application build.

Keeping GitHub Release and GHCR publication in the canonical release lifecycle prevents version, tag, artifact and provenance drift.

## Important maintained tools

- `release_notes.py` — generates release notes from the matching `CHANGELOG.md` section.
- `make_payload.py` — creates the verified Windows Setup payload.
- `verify_release.py` / `verify_bundle.py` — verify public release/bundle structure, identity and integrity.
- `audit_brand_hardcut.py` — enforces the Ghost FTP public product identity boundary.
- `audit_repository.py` — rejects repository layout, generated-artifact, private-key and current-release drift.
- `audit_platform_contract.py` — enforces the current public Windows/Linux release boundary while allowing active Android/macOS source surfaces.
- `audit_desktop_surface.py` — validates maintained native desktop/source platform contracts.
- `audit_dependencies.py` — rejects unexpected dependency and tracking/analytics SDK drift.
- `audit_version.py` — validates root version, release identity, platform binding and signing contract.
- `audit_localization.py` — validates the maintained 24-language desktop catalog and release identity.
- `audit_security.py` — security-policy and fail-closed behavior checks.
- `audit_privacy.py` — telemetry, secret-retention and diagnostic-privacy checks.
- `audit_docs.py` — validates active documentation, local media, release shape, platform truthfulness and signed-only public Windows policy.
- `audit_release.py` — validates exact public artifact, signing, package and release-workflow contracts.
- `assemble_ui_evidence.py` — validates immutable exact-head runtime evidence before cross-platform evidence bundling.
- platform-specific build/package helpers referenced by maintained CI or documented local workflows.

## Platform boundary

The current public release platforms are:

```text
WINDOWS,LINUX
```

The active source platforms are:

```text
WINDOWS,LINUX,ANDROID,MACOS
```

Android produces a maintained development APK. macOS produces a maintained universal development app and also has a separate fail-closed Developer ID signing/notarization path. Neither development artifact is part of the current 17-file public Windows/Linux allow-list.

Browser companion source under `ekstenzije/` is a privacy-minimal local connection helper. It parses supported FTP-family targets locally; it does **not** provide a supported browser-to-desktop launch/handoff contract today.

## Release identity

Ghost FTP uses root `VERSION` plus namespaced tags:

```text
ghostftp-vX.Y.Z
```

Published release identities are immutable. A newer public version uses a new semantic version and must complete exact-head validation, publication, remote read-back and retention verification before superseded public Ghost FTP identities may be removed.

For the current line:

```text
VERSION=0.0.5
TAG=ghostftp-v0.0.5
CHANNEL=current
PRERELEASE=false
```

## Build invariants

Maintained production workflows:

- disable Go telemetry;
- use the repository-pinned Go toolchain contract;
- use `GOTOOLCHAIN=local`;
- use `GOPROXY=off` and `GOSUMDB=off` for the dependency-free maintained Go module;
- bind artifacts to the exact source revision;
- run security/privacy/version/documentation/release audits before public publication;
- generate final SHA-256 metadata only from finalized artifact bytes.

## Windows Authenticode invariant

Official Windows publication is **signed-only**.

The canonical `Publish Ghost FTP` workflow requires the protected production Authenticode identity, signs the finalized public Setup and Portable executables, verifies each final file with the Windows Authenticode API and accepts only:

```text
WINDOWS_AUTHENTICODE=signed
```

A missing production PFX/password, signing failure, missing signer certificate, invalid Authenticode status or non-`signed` publication state is a release failure.

Local development builds and ordinary CI Windows packaging may be unsigned so engineers can build/test without production private-key material. Those outputs are development/test artifacts and do not satisfy the official public-release contract.

A short-lived self-signed certificate may be used only by the dedicated CI signing mechanics smoke test. It must never be substituted for or represented as the trusted production publisher identity.

The repository must never contain the production private key, signing password or other protected production signing material.

## Canonical public artifact shape

The current public Ghost FTP 0.0.5 release contains **14 platform artifacts / 17 public files**.

Windows:

```text
Ghost-FTP-0.0.5-Setup.exe
Ghost-FTP-0.0.5-Portable.exe
```

Linux:

```text
Ghost-FTP-0.0.5-Linux-Debian-amd64.deb
Ghost-FTP-0.0.5-Linux-Debian-arm64.deb
Ghost-FTP-0.0.5-Linux-Debian-i386.deb
Ghost-FTP-0.0.5-Linux-Ubuntu-amd64.deb
Ghost-FTP-0.0.5-Linux-Ubuntu-arm64.deb
Ghost-FTP-0.0.5-Linux-Ubuntu-i386.deb
Ghost-FTP-0.0.5-Linux-Fedora-x86_64.rpm
Ghost-FTP-0.0.5-Linux-Fedora-aarch64.rpm
Ghost-FTP-0.0.5-Linux-Fedora-i686.rpm
Ghost-FTP-0.0.5-Linux-Portable-amd64.tar.gz
Ghost-FTP-0.0.5-Linux-Portable-arm64.tar.gz
Ghost-FTP-0.0.5-Linux-Portable-i386.tar.gz
```

Release metadata/verification:

```text
BUILD-METADATA.txt
RELEASE-NOTES.txt
SHA256.txt
```

## Package invariant

The GitHub Packages distribution bundle is built with network access disabled from only the already verified `release/` directory. Source worktrees, user data, signing material and release secrets must never be copied into the package.

Do not add an independent release script or package-registry publisher. Distribution logic belongs in the canonical release workflow and must preserve exact source binding, tag immutability, the 17-file allow-list, SHA-256 generation, signed-only Windows publication, `prerelease=false`, remote release read-back, GHCR read-back and latest-only retention.

## Verification before publication

The maintained release lifecycle verifies, as applicable:

- Go formatting, race tests and vet;
- repository/platform/dependency/version/localization contracts;
- security, privacy, documentation and release audits;
- the complete Python regression suite;
- Windows universal Setup and Portable construction;
- trusted Authenticode on both official public Windows executables;
- canonical Linux Debian/Ubuntu/Fedora/Portable build, metadata, extraction and binary parity;
- the exact **14 platform artifacts / 17 public files** allow-list;
- exact current source identity;
- `prerelease=false`;
- SHA-256 coverage;
- GitHub Release immediate and delayed read-back;
- `ghcr.io/bren-wp/ghost-ftp:0.0.5` read-back;
- latest-only release/tag/package retention only after successful successor verification.

A script change that weakens one of these checks is a release/security change, not routine documentation maintenance.

## Security

Never embed production signing keys, tokens, FTP credentials, private-key passphrases, recovery secrets, customer data or private certificates in scripts. Production signing credentials and registry authorization are supplied only by protected CI environments, used for the shortest practical lifetime and must never be printed to logs, committed to source or published inside artifacts.

See [`../docs/SIGNING.md`](../docs/SIGNING.md), [`../docs/RELEASE-VERIFICATION.md`](../docs/RELEASE-VERIFICATION.md), [`../docs/TESTING.md`](../docs/TESTING.md) and [`../docs/SECURITY.md`](../docs/SECURITY.md).
