# Ghost FTP testing and quality gates

Ghost FTP **1.0.0 Stable** is release-ready only when source tests, repository audits, native Windows/Linux production builds and remote distribution read-back pass for the exact release revision.

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
- truthful transfer progress/speed/ETA.

## Windows production gate

`BUILD-WINDOWS.ps1` produces and verifies:

```text
Setup x64
Setup x86
Portable x64
Portable x86
```

The release assembly additionally creates the byte-identical x32 Setup compatibility alias from x86.

CI validates executable/package metadata and runs a development Authenticode pipeline smoke test. During production release, a configured protected Authenticode identity must produce valid signatures. If no protected production identity is configured, the Windows release is explicitly recorded as unsigned instead of being given a fake/self-signed production identity.

## Linux production gate

`linux/BUILD.sh` builds:

```text
amd64
arm64
i386
```

DEB metadata is verified for package name, semantic version and architecture. The release job creates the multiarch ZIP only from those verified packages.

## Desktop/UI regression

Windows UI regression tests protect native workspace geometry, connection/action state, Site Manager behavior, localization, keyboard workflow and authentic screenshot-capture contracts.

Linux tests protect shared Engine access, SFTP password/key/passphrase parity, queue controls, settings/profile behavior and native renderer operation. Idle redraw behavior is optimized so unchanged state does not force unnecessary full-workspace redraw.

## Localization gate

Localization checks require exactly 24 canonical languages, English default/fallback, valid catalog keys/format verbs, Windows live localization, Setup primary-copy coverage and Linux runtime switching.

## Privacy gate

Privacy audit rejects fixed product telemetry URLs, known tracking vendor markers, forbidden general-purpose network imports in runtime source, credential-file regressions and ineffective build telemetry controls.

Release/package tooling additionally verifies that GHCR copies only the explicit `release/` allow-list and does not receive runtime FTP/SFTP credentials or signing private keys.

## Stable release gate

`.github/workflows/release.yml` reruns quality, Windows and Linux jobs before publication. It rejects `MAJOR=0`, so maintained production publication never creates a new prerelease.

The assembled GitHub Release contains **9 platform artifacts** and **12 public files** total. Stable releases require:

```text
draft=false
prerelease=false
```

The workflow verifies exact `main` state before publication, preserves immutable tags, compares the remote asset set and downloads the published `SHA256.txt` for byte-for-byte read-back. It repeats Release verification after a short delay.

## Windows trust verification

`BUILD-METADATA.txt` must describe the actual Windows state:

```text
WINDOWS_AUTHENTICODE=signed|unsigned
WINDOWS_TRUST_MODE=...
```

A signed state requires successful Authenticode validation in the Windows job. An unsigned state is permitted only as an explicit unsigned state and relies on SHA-256 + official GitHub source provenance; no test/self-signed identity may be presented as production signing.

## GitHub Packages gate

After the GitHub Release is verified, stable releases publish:

```text
ghcr.io/bren-wp/ghost-ftp:<version>
```

The OCI distribution bundle uses `FROM scratch`, copies only `release/` and builds with Docker networking disabled. The workflow pushes semantic aliases, removes local images, pulls the semantic-version tag back from GHCR and verifies:

- source/version/revision OCI labels;
- readable registry manifest/digest;
- embedded `SHA256.txt` byte-for-byte equality;
- embedded `BUILD-METADATA.txt` byte-for-byte equality.

The successful remote package gate emits `PACKAGE_READBACK=PASS`.

## Release-readiness rule

A branch that merely compiles is not a release. Stable readiness requires all automated gates plus exact Release/Package read-back and truthful cryptographic state on the final source revision.

See [Release verification](RELEASE-VERIFICATION.md), [Security](SECURITY.md), [Privacy](PRIVACY.md), [Signing](SIGNING.md) and [Packages](PACKAGES.md).
