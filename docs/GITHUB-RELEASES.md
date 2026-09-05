# GitHub Releases

Ghost FTP uses immutable namespaced tags and a fail-closed release workflow.

## Version and tag policy

The canonical source version is the root `VERSION` file and must be semantic `X.Y.Z`.

Current source line: **2.0.0**.

Release tags use:

```text
ghostftp-vX.Y.Z
```

The release workflow refuses to move an existing release tag to a different commit. Published 1.x tags/releases remain historical provenance and are not rewritten when the active platform contract changes.

## Active 2.x application platforms

Ghost FTP 2.x publishes desktop application artifacts for:

- Windows;
- Linux.

Android, iOS and macOS application artifacts are not part of the 2.x release matrix.

The existing Web companion source is maintained separately and is not counted as a desktop/platform artifact in this release contract.

## Platform artifact contract

A complete 2.x release contains **9 platform artifacts**:

1. `Ghost-FTP-X.Y.Z-Setup-x64.exe`
2. `Ghost-FTP-X.Y.Z-Setup-x86.exe`
3. `Ghost-FTP-X.Y.Z-Setup-x32.exe`
4. `Ghost-FTP-X.Y.Z-Portable-x64.exe`
5. `Ghost-FTP-X.Y.Z-Portable-x86.exe`
6. `Ghost-FTP-X.Y.Z-Linux-amd64.deb`
7. `Ghost-FTP-X.Y.Z-Linux-arm64.deb`
8. `Ghost-FTP-X.Y.Z-Linux-i386.deb`
9. `Ghost-FTP-X.Y.Z-Linux-multiarch.zip`

`Setup-x32` is intentionally a compatibility alias of the x86 setup build and the workflow verifies that both files have the same SHA-256 digest.

## Release support files

Each release also contains:

- `RELEASE-NOTES.txt` — release-specific changes derived from maintained release notes;
- `BUILD-METADATA.txt` — version, commit and platform/build contract;
- `SHA256.txt` — SHA-256 for every other public release file.

Therefore a complete 2.x release has **12 public files**.

## NuGet/GitHub Package

The workflow also builds/publishes the `GhostFTP` NuGet/GitHub Package containing the Windows portable executables for x64 and x86.

This package is separate from the 12 public GitHub Release files.

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
- 24-language localization audit;
- security/privacy audits;
- documentation audit;
- Web companion source/runtime audit.

## Publication protections

Before a release is created or updated, the workflow verifies that `main` still points at the release commit. If `main` moved, publication stops.

If a release tag already exists, the workflow verifies that it resolves to the current release commit. If not, publication stops with a refusal to rewrite historical provenance.

After upload, the workflow reads the published asset list back and verifies that exactly 12 public files are present.

## Release metadata

`BUILD-METADATA.txt` identifies:

- public brand and technical identity;
- semantic version;
- release tag;
- commit SHA;
- active application platforms;
- Windows architectures/package types;
- Linux package architectures;
- language count and default language;
- telemetry status;
- artifact/file counts.

## Historical 1.x releases

Older 1.x releases may contain Android, iOS, macOS or Web artifacts because those platforms were part of the release contract at that time. Their presence in historical release notes/assets is expected and must not be confused with current 2.x support.

See [Release history](RELEASE-HISTORY.md) and [Release verification](RELEASE-VERIFICATION.md).
