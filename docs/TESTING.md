# Testing and quality gates

Tests, audits and native platform builds are release requirements for Ghost FTP. A feature is not considered production-ready merely because it compiles or appears in one frontend.

## Continuous integration

`.github/workflows/ci.yml` runs three release-relevant jobs:

1. **Core quality, security and documentation** on Linux;
2. **Windows x64/x86 production build**;
3. **Linux amd64/arm64/i386 production build**.

Android, iOS and macOS application build jobs are not part of the active Windows/Linux matrix.

## Active release lifecycle

The current baseline is **0.1.0 Beta**. Every `0.x.y` build is a Beta/Prerelease. The first stable release is **1.0.0**.

Setup and Portable artifacts always use the same canonical `VERSION` value.

## Go toolchain and core gates

The maintained Go toolchain is **1.27.1**. The desktop/core module intentionally has no external Go module dependency graph.

Core verification includes:

```text
go telemetry off
gofmt
go test -race ./...
go vet ./...
```

CI uses `GOTOOLCHAIN=local`, `GOPROXY=off` and `GOSUMDB=off` so a production validation run cannot silently fetch an undeclared Go dependency.

## Repository audits

### Repository integrity

`scripts/audit_repository.py` protects tracked-source hygiene including unsafe/generated paths, case collisions, malformed text and accidental one-shot workflow/tooling leakage.

### Platform contract

`scripts/audit_platform_contract.py` requires:

- active Windows and Linux workflow jobs;
- a semantic VERSION at or above the 0.1.0 product baseline;
- absence of active `android/`, `ios/` and `macos/` roots;
- absence of retired mobile packaging/audit tooling;
- no retired-platform markers in the active CI/release workflow.

### Version and release contract

`scripts/audit_version.py` binds root `VERSION` to Windows/Linux build metadata, the Beta/stable channel rule and release workflow.

`scripts/audit_release.py` verifies the Windows/Linux artifact names, release counts, immutable tag rules, Windows installer/payload identity, NuGet packaging and Linux package contract.

### Dependencies

`scripts/audit_dependencies.py` verifies:

- zero external Go modules;
- no `go.sum`/vendored module graph;
- zero third-party Web Composer packages;
- explicit documentation of OS `curl`, `ssh` and `sftp` transport prerequisites;
- tracking/advertising/crash SDK markers remain forbidden.

## Security and privacy audits

`scripts/audit_security.py` protects high-risk invariants including:

- SFTP AskPass behavior and no disk password/passphrase artifact;
- private-key symlink/reparse validation;
- protected runtime credential handling;
- profile endpoint/account/private-key identity binding;
- transfer staging validation;
- root-delete protection;
- session close/race handling;
- Linux SFTP password/key/passphrase parity;
- Linux queue/settings access through the shared engine;
- Linux authenticated profile-storage handling.

`scripts/audit_privacy.py` rejects:

- application telemetry/vendor markers;
- forbidden general-purpose runtime network imports;
- fixed HTTP(S) URLs in desktop runtime source;
- proxy/credential environment leakage;
- ineffective build-time telemetry-disable configuration.

## Localization gate

`scripts/audit_localization.py` verifies the English-first 24-language registry.

The gate checks:

- English is first/default;
- all 24 runtime catalogs are present;
- catalog/test invariants and format placeholders remain valid;
- Windows live localization remains wired;
- Windows Setup primary copy remains available for the canonical language set;
- Linux resolves and changes language through the shared settings/localization layer.

## Documentation gate

`scripts/audit_docs.py` verifies:

- local Markdown/HTML links resolve;
- documentation is indexed from `docs/README.md` and the root README;
- current version markers match `VERSION`;
- the release contract remains **9 platform artifacts / 12 public files**;
- active documentation does not present a retired version line as current;
- current installation/architecture/roadmap/release documentation does not link to retired application roots;
- the Windows/Linux parity and versioning documents remain present.

## Web companion gate

The Web companion remains a separate source surface but is still validated by `scripts/audit_web.py`.

This does not make the Web companion a Windows/Linux desktop release artifact.

## Windows production gate

`BUILD-WINDOWS.ps1` is the canonical Windows build entry point.

The CI job verifies non-empty:

- Setup x64;
- Setup x86;
- Portable x64;
- Portable x86.

The release workflow also creates the x32 compatibility alias from the x86 Setup artifact and verifies byte/hash identity.

Windows production validation additionally protects installer payload integrity, architecture/subsystem metadata, manifest/icon/version resources, hardening flags where applicable and signing-status reporting.

## Authenticode signing gate

`scripts/Sign-WindowsArtifacts.ps1` is the only supported Windows artifact-signing helper.

When a PFX is configured:

1. the Portable executable is signed after final PE resource mutation;
2. the signed Portable bytes are embedded in Setup;
3. Setup is signed after its own final PE resource mutation;
4. release verification confirms signed state;
5. SHA-256 manifests are generated only after signing.

A development self-signed certificate can be created with `scripts/New-DevCodeSigningCertificate.ps1` to test the pipeline. Development/self-signed certificates are not trusted publisher identities and must never be committed to the repository.

Production signing requires a protected external PFX/private key. If production signing is configured, build verification fails if any Setup/Portable artifact remains unsigned.

## Authentic UI screenshot gate

`.github/workflows/ui-screenshots.yml` builds the real Windows Portable x64 executable, launches it, captures the real main workspace and Site Manager native windows, verifies the PNG outputs and persists changed captures under `docs/images/`.

The workflow does not generate a mockup or AI illustration. Screenshot persistence refuses to overwrite a branch that moved after capture.

## Linux production gate

`linux/BUILD.sh` builds Debian packages for:

- amd64;
- arm64;
- i386.

CI verifies each package is non-empty and reports:

- package name `ghost-ftp`;
- VERSION matching root `VERSION`;
- the expected Debian architecture.

The release workflow also combines those three DEBs into `Ghost-FTP-X.Y.Z-Linux-multiarch.zip`.

## Runtime parity verification

High-value parity is protected by shared-core use rather than duplicate platform implementations.

Linux frontend security auditing explicitly requires access to shared-engine operations for:

- connect/list/transfer;
- SFTP password and key+passphrase authentication;
- pause/resume/cancel/retry/clear queue controls;
- settings persistence;
- profile listing.

See [Platform parity](PLATFORM-PARITY.md).

## Python tooling regression suite

Maintenance/security tooling regressions run with:

```text
python -m unittest discover -s scripts -p 'test_*.py'
```

When a platform/tooling contract is removed, its obsolete tests must be removed or rewritten in the same change so CI does not preserve dead requirements.

## Production release gate

`.github/workflows/release.yml` publishes only after:

- quality succeeds;
- Windows succeeds;
- Linux succeeds.

The release contains **9 platform artifacts**:

1. Setup x64
2. Setup x86
3. Setup x32 compatibility alias
4. Portable x64
5. Portable x86
6. Linux amd64 DEB
7. Linux arm64 DEB
8. Linux i386 DEB
9. Linux multiarch ZIP

The workflow adds `RELEASE-NOTES.txt`, `BUILD-METADATA.txt` and `SHA256.txt`, for **12 public files**.

Every `0.x.y` release is marked Beta/Prerelease. Stable publication begins at `1.0.0`.

Before publication it verifies `main` still points at the build commit. Existing `ghostftp-vX.Y.Z` tags are never moved to another commit.

## Release-readiness rule

Do not describe a release as stable/ready solely from source review. Release readiness requires the current branch/commit to pass the complete automated quality and platform build gates, followed by verification of the assembled release metadata/assets.
