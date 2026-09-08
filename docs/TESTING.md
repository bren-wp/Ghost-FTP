# Ghost FTP testing and quality gates

Ghost FTP **1.1.7 Stable** is the current release. A release is accepted only when source tests, audits, native production builds, signing-state checks, packaging verification, authentic UI evidence and distribution read-back pass for the exact release revision.

Historical Stable releases remain immutable. New gates strengthen future/current source without rewriting older tags, assets or checksums.

## Continuous integration

The maintained quality contract has five proof layers:

1. core quality, security and documentation;
2. Windows x64/x86 production build;
3. canonical Linux amd64/arm64/i386 production build;
4. supplemental Debian/Ubuntu/Fedora/Portable distro-package build and parity verification;
5. native x86-64 package install/remove and installed-GUI smoke verification on Debian 13, Ubuntu 26.04 LTS and Fedora 44.

The first three run in Ghost FTP CI. Distro layers run in `.github/workflows/linux-distro-packages.yml` and `.github/workflows/linux-distro-install.yml`. A green result in one layer never substitutes for a required failing layer.

## Core Go gates

The maintained toolchain is Go **1.27.1**:

```text
go telemetry off
gofmt
go test -race ./...
go vet ./...
```

CI uses `GOTOOLCHAIN=local`, `GOPROXY=off` and `GOSUMDB=off` so release quality cannot silently acquire an undeclared Go dependency.

## Repository audits

The quality job runs brand/repository/platform/desktop/dependency/version/localization/security/privacy/documentation/release audits and then:

```text
python -m unittest discover -s scripts -p 'test_*.py'
```

## Protocol and transfer regressions

Deterministic tests cover real loopback FTP lifecycle, invalid credentials, FTPS no-downgrade behavior, SFTP host-key and secret ownership, transfer staging/rollback, cancellation/retry generation binding, local-root containment, recursive filesystem operations, path/symlink/reparse safeguards, settings/profile recovery and privacy-safe diagnostics.

The 1.1.7 line additionally protects rooted tree preparation and the Windows modal/picker localization changes with explicit Go/Python regression contracts.

## Windows production gate

`BUILD-WINDOWS.ps1` produces and verifies:

```text
Setup x64
Setup x86
Portable x64
Portable x86
```

The release assembly creates `Ghost-FTP-X.Y.Z-Setup-x32.exe` as a byte-identical compatibility alias of verified x86 Setup; it is not a separate architecture build.

CI validates package metadata and runs an Authenticode private-key pipeline smoke test with a short-lived development certificate. Production publication signs only when a protected trusted production certificate is configured; otherwise metadata records `WINDOWS_AUTHENTICODE=unsigned`.

## Windows modal/localization regression gate

Tests require:

- application-owned Confirm/Info/Error DecisionCard primary routing;
- stock TaskDialog/MessageBox only as creation-failure fallback;
- shared native Light/Dark palette, DPI, owner modality and keyboard loop;
- live-locale OK/Cancel/Yes/No labels;
- 24-language Save Profile privacy/security text;
- runtime-localized native SSH-key and folder pickers;
- adaptive DecisionCard geometry for long localized text;
- unchanged credential clear/retain binding semantics.

## Canonical Linux production gate

`linux/BUILD.sh` builds for:

```text
amd64
arm64
i386
```

For every architecture it produces a generic DEB and generic portable `.tar.gz` archive from the same compiled executable. CI validates DEB metadata, archive structure and byte-for-byte `ghostftp` executable parity. The release workflow creates the multiarch ZIP from verified generic DEBs.

## Supplemental distro-package gate

`linux/BUILD-DISTROS.sh` builds:

- Debian DEB for `amd64`, `arm64`, `i386`;
- Ubuntu DEB for `amd64`, `arm64`, `i386`;
- Fedora RPM for `x86_64`, `aarch64`, `i686`;
- distro-neutral Portable tar.gz for `amd64`, `arm64`, `i386`.

`.github/workflows/linux-distro-packages.yml` validates metadata and executable parity across Debian, Ubuntu, Fedora and Portable package families.

These supplemental packages are not canonical public release assets unless the release allow-list explicitly includes them.

## Native distro installation gate

`.github/workflows/linux-distro-install.yml` rebuilds packages from exact source and performs package-manager lifecycle plus installed-GUI smoke on:

```text
Debian 13 amd64
Ubuntu 26.04 LTS amd64
Fedora 44 x86_64
```

It verifies OS identity, package metadata/architecture, dependency resolution, installed ownership/state, runtime tools, required desktop files, `/usr/bin/ghostftp` startup under local Xvfb, package removal and absence of package-owned system residue.

Native package-manager/runtime coverage is deliberately limited to x86-64. arm64/aarch64 and i386/i686 are still covered by exact-head build, metadata, extraction and payload-parity gates.

## Authentic screenshot gate

The dedicated Windows screenshot workflow builds and launches the real x64 Portable executable and captures Main Workspace, Site Manager, Settings and About. Release-prep evidence must come from that exact final head and must display the intended 1.1.7 public version/branding without clipping or overlap. A mockup is not accepted as evidence.

## Localization gate

Localization checks require exactly 24 canonical languages, English default/fallback, valid catalog keys/format verbs, Windows live localization, Setup primary-copy coverage and Linux runtime switching. Native picker titles/filters and privacy-sensitive profile credential decisions are part of the maintained localization surface.

## Privacy gate

Privacy audit rejects fixed product telemetry URLs, tracking-vendor markers, forbidden general-purpose runtime network imports, credential-file regressions and ineffective build telemetry controls. The GHCR bundle copies only the verified release directory and builds with networking disabled.

## Release gate

`.github/workflows/release.yml` runs quality, Windows and canonical Linux jobs before publication.

Ghost FTP 1.1.7 assembles **12 platform artifacts / 15 public files**: five Windows artifacts, three generic Linux DEBs, three generic Linux tar.gz archives, the Linux multiarch ZIP and three metadata/checksum files.

The historical 1.1.6 release remains immutable at **9 platform artifacts / 12 public files**. Supplemental distro-specific Debian/Ubuntu/Fedora/Portable CI packages are not included in the 1.1.7 public release count.

Before and after publication, the workflow verifies that `main` is still the exact release commit and that an existing version tag is not rewritten. Signing state must be either `signed` or `unsigned`; absence of a production certificate is carried as explicit unsigned metadata.

## GitHub Packages gate

Stable releases publish:

```text
ghcr.io/bren-wp/ghost-ftp:<version>
```

The OCI distribution bundle is built from `FROM scratch`, copies only canonical `release/`, publishes compatible aliases only after semantic-version push/read-back and is a distribution bundle, not a runtime container.

## Exact-head and post-merge rule

A branch result is authoritative only for the exact code head that produced it. Any code-head change invalidates earlier green results for merge purposes.

Release-prep changes merge only after all relevant exact-head gates are green. After merge, corresponding push workflows on the resulting `main` SHA must also finish successfully before canonical release-branch creation.

## Release-readiness rule

A source branch that merely compiles is not a release. Stable readiness requires all automated gates plus exact artifact/signing-state/package/release verification on the final source revision.

See [Release verification](RELEASE-VERIFICATION.md), [Installation](INSTALLATION.md), [Platform parity](PLATFORM-PARITY.md), [Security](SECURITY.md), [Privacy](PRIVACY.md) and [Packages](PACKAGES.md).
