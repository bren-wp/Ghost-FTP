# Ghost FTP testing and quality gates

Ghost FTP **1.0.0 Stable** is release-ready only when source tests, audits, native production builds, signing-state checks and distribution read-back all pass for the exact release revision.

## Continuous integration

The maintained CI contract validates:

1. core quality/security/documentation on Linux;
2. Windows x64/x86 production build;
3. Linux amd64/arm64/i386 production build.

Both platform families are mandatory; one cannot substitute for the other.

## Core Go gates

The maintained toolchain is Go **1.27.1**. Production checks include:

```text
go telemetry off
gofmt
go test -race ./...
go vet ./...
```

CI uses `GOTOOLCHAIN=local`, `GOPROXY=off` and `GOSUMDB=off` so the release quality gate cannot silently acquire an undeclared Go dependency.

## Repository audits

The quality job executes:

```text
python scripts/audit_brand_hardcut.py
python scripts/audit_repository.py
python scripts/audit_platform_contract.py
python scripts/audit_desktop_surface.py
python scripts/audit_dependencies.py
python scripts/audit_version.py
python scripts/audit_localization.py
python scripts/audit_security.py
python scripts/audit_privacy.py
python scripts/audit_docs.py
python scripts/audit_release.py
```

and then the Python tooling regressions:

```text
python -m unittest discover -s scripts -p 'test_*.py'
```

## High-risk regression areas

Go tests cover, among other areas:

- FTP/FTPS/SFTP validation and protocol/tool behavior;
- SFTP host-key fingerprints and key paths;
- runtime secret/AskPass handling;
- endpoint/profile binding;
- process lifecycle and disconnect races;
- transfer staging, source snapshots and commit cleanup;
- retry/cancel/terminal state correctness;
- symlink/reparse-aware filesystem operations;
- settings/profile validation and recovery;
- privacy-safe connection diagnostics;
- truthful transfer metrics.

## Windows production gate

`BUILD-WINDOWS.ps1` produces and verifies:

```text
Setup x64
Setup x86
Portable x64
Portable x86
```

The release assembly also creates the byte-identical x32 Setup compatibility alias from x86.

CI validates executable/package metadata and runs the development Authenticode pipeline smoke test with a short-lived development certificate. Production publication does not convert that test identity into a trusted publisher. When real protected Authenticode secrets are configured, the release workflow signs and verifies each Windows artifact; when they are absent, the release is explicitly marked unsigned.

## Linux production gate

`linux/BUILD.sh` builds:

```text
amd64
arm64
i386
```

DEB metadata is verified for package name, semantic version and architecture. The release job creates the multiarch ZIP only from those verified packages.

## Desktop/UI regression

Windows UI regression tests protect native workspace geometry, connection/action state, Site Manager behavior, localization, keyboard workflow and screenshot capture contracts.

Linux tests protect shared Engine access, SFTP password/key/passphrase parity, queue controls, settings/profile behavior and native renderer operation. Idle redraw behavior is optimized so unchanged state does not force unnecessary full-workspace redraw.

## Localization gate

Localization checks require exactly 24 canonical languages, English default/fallback, valid catalog keys/format verbs, Windows live localization, Setup primary copy coverage and Linux runtime switching.

## Privacy gate

Privacy audit rejects fixed product telemetry URLs, known tracking vendor markers, forbidden general-purpose network imports in runtime source, credential-file regressions and ineffective build telemetry controls.

The release-package contract additionally ensures the GHCR bundle copies only the verified release directory and builds with Docker networking disabled.

## Release gate

`.github/workflows/release.yml` runs the quality, Windows and Linux jobs before publication.

The assembled GitHub Release contains **9 platform artifacts** and **12 public files** total. Stable version `1.0.0` is published with `prerelease=false`.

Before and after publication, the workflow verifies that `main` is still the exact release commit and that an existing version tag is not being rewritten.

The release gate also validates that `WINDOWS_SIGNING_STATE` is either `signed` or `unsigned`. A configured signing identity that does not produce valid signatures fails. Absence of a production certificate does not fail the release; it is carried through as `WINDOWS_AUTHENTICODE=unsigned` in release metadata.

## GitHub Packages gate

Stable releases publish:

```text
ghcr.io/bren-wp/ghost-ftp:<version>
```

The OCI distribution bundle is built from `FROM scratch`, copies only `release/`, publishes stable semantic aliases and is read back through the registry before the release job is considered complete.

## Release-readiness rule

A source branch that merely compiles is not a release. Stable readiness requires all automated gates plus exact artifact/signing-state/package/release verification on the final source revision.

See [Release verification](RELEASE-VERIFICATION.md), [Security](SECURITY.md), [Privacy](PRIVACY.md) and [Packages](PACKAGES.md).
