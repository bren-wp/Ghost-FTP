# GitHub Releases

Ghost FTP uses immutable namespaced tags and a fail-closed release workflow.

## Version and tag policy

The canonical desktop application version is the root `VERSION` file and must be semantic `X.Y.Z`.

Current source version: **0.2.0**.

Current release channel: **Beta / prerelease**.

Release tags use:

```text
ghostftp-vX.Y.Z
```

Examples:

```text
ghostftp-v0.1.0
ghostftp-v0.1.1
ghostftp-v1.0.0
```

The release workflow refuses to move an existing release tag to a different commit. Published historical tags/releases remain immutable provenance and are not rewritten when the active development baseline changes.

## Beta and stable channels

All `0.x.y` versions are pre-1.0 **Beta** builds.

When the release workflow resolves a version whose major component is `0`, it automatically:

- sets `RELEASE_CHANNEL=beta`;
- uses a release title such as `Ghost FTP 0.2.0 Beta`;
- creates or updates the GitHub Release with `prerelease=true`;
- verifies the prerelease flag again during remote release readback;
- records the release channel in `BUILD-METADATA.txt`.

A `0.x.y` build must never be presented as the first fully stable Ghost FTP release.

The first version eligible for the stable channel is **1.0.0**. When the canonical version is 1.0.0 or later, the workflow uses `RELEASE_CHANNEL=stable` and verifies that the GitHub Release is not marked as a prerelease.

See [Versioning policy](VERSIONING.md) for the product-quality gate that must be satisfied before the project intentionally moves to 1.0.0.

## Active application platforms

Ghost FTP publishes maintained desktop application artifacts for:

- Windows;
- Linux.

Android, iOS and macOS application artifacts are not part of the active release matrix.

## Platform artifact contract

A complete desktop release contains **9 platform artifacts**:

1. `Ghost-FTP-X.Y.Z-Setup-x64.exe`
2. `Ghost-FTP-X.Y.Z-Setup-x86.exe`
3. `Ghost-FTP-X.Y.Z-Setup-x32.exe`
4. `Ghost-FTP-X.Y.Z-Portable-x64.exe`
5. `Ghost-FTP-X.Y.Z-Portable-x86.exe`
6. `Ghost-FTP-X.Y.Z-Linux-amd64.deb`
7. `Ghost-FTP-X.Y.Z-Linux-arm64.deb`
8. `Ghost-FTP-X.Y.Z-Linux-i386.deb`
9. `Ghost-FTP-X.Y.Z-Linux-multiarch.zip`

`Setup-x32` is intentionally a compatibility alias of the x86 Setup build and the workflow verifies that both files have the same SHA-256 digest.

## Setup and Portable version invariant

Windows Setup and Portable are packaging forms of the same Ghost FTP release. They must always use the identical canonical `VERSION`.

For the current Beta release:

```text
Ghost-FTP-0.2.0-Setup-x64.exe
Ghost-FTP-0.2.0-Setup-x86.exe
Ghost-FTP-0.2.0-Portable-x64.exe
Ghost-FTP-0.2.0-Portable-x86.exe
```

For the first stable release:

```text
Ghost-FTP-1.0.0-Setup-x64.exe
Ghost-FTP-1.0.0-Setup-x86.exe
Ghost-FTP-1.0.0-Portable-x64.exe
Ghost-FTP-1.0.0-Portable-x86.exe
```

The package form never has an independent version counter.

## Release support files

Each release also contains:

- `RELEASE-NOTES.txt` — release-specific changes derived from maintained release notes;
- `BUILD-METADATA.txt` — version, release channel, commit and platform/build contract;
- `SHA256.txt` — SHA-256 for every other public release file.

Therefore a complete release has **12 public files**.


## Release quality gate

Publication depends on all three jobs:

1. shared quality/security/documentation gate;
2. Windows production build;
3. Linux production build.

The quality gate includes:

- formatting;
- Go race tests and vet;
- repository integrity;
- Windows/Linux platform-contract audit;
- dependency/no-tracking audit;
- version/release audit;
- Beta-versus-stable channel validation;
- 24-language localization audit;
- security/privacy audits;
- documentation audit;
- retired-surface/desktop-source audit.

## Publication protections

Before a release is created or updated, the workflow verifies that `main` still points at the release commit. If `main` moved, publication stops.

If a release tag already exists, the workflow verifies that it resolves to the current release commit. If not, publication stops with a refusal to rewrite historical provenance.

After upload, the workflow reads the published asset list back and verifies that exactly 12 public files are present.

The workflow also reads the GitHub Release metadata back and verifies that `prerelease=true` for the Beta channel and `prerelease=false` for the stable channel.

## Release metadata

`BUILD-METADATA.txt` identifies:

- public brand and technical identity;
- semantic version;
- release tag;
- release channel (`beta` or `stable`);
- commit SHA;
- active application platforms;
- Windows architectures/package types;
- Linux package architectures;
- language count and default language;
- telemetry status;
- artifact/file counts.

## Historical releases

Older repository releases may contain Android, iOS, macOS or Web artifacts because those surfaces existed in earlier release contracts. Their presence in historical release notes/assets is expected and must not be confused with the active Windows/Linux product matrix.

Historical version numbers are retained for reproducibility. The current pre-1.0 line started at **0.1.0 Beta**; the active release prepared here is **0.2.0 Beta**, advancing toward the first stable **1.0.0** release.

See [Release history](RELEASE-HISTORY.md), [Release verification](RELEASE-VERIFICATION.md) and [Versioning policy](VERSIONING.md).

## Automated production trigger

After the release candidate is merged and the exact `main` commit has passed all required gates, maintainers create `release/ghostftp-vX.Y.Z` at that exact `main` SHA. The dedicated `release-branch-trigger.yml` workflow accepts only that branch namespace, verifies that the branch SHA still equals current `main` and that `X.Y.Z` equals the root `VERSION`, then dispatches the canonical `release.yml` workflow on `main` with the same version guard.

The canonical release workflow then re-runs the complete quality, Windows and Linux production gates, checks `main` again before publication and delayed readback, preserves immutable `ghostftp-vX.Y.Z` tag provenance, publishes the 12-file GitHub Release set and verifies the remote asset manifest/checksum state. A manual `workflow_dispatch` remains available as an explicit maintainer fallback.
