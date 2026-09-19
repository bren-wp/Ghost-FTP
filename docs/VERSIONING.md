# Versioning

Ghost FTP uses semantic versions in root `VERSION`.

## Active source platforms

```text
WINDOWS,LINUX,ANDROID
```

Browser helpers are supporting packages and do not change the native application platform set.

## Rules

- `0.0.0` is reserved;
- published versions are immutable;
- a new release requires a new version;
- release tags use `ghostftp-vX.Y.Z`;
- a release branch uses `release/ghostftp-vX.Y.Z`;
- VERSION, release notes, metadata and package names must agree;
- release artifacts must be rebuilt from the exact release commit.

## Current line

Current published version: **0.0.8**.

Current development line: **0.0.9**.

## Platform identity

Windows Setup/Portable, Linux bundles and the Android APK all derive product version identity from the repository release version contract.

The retired macOS application no longer participates in version validation or artifact counts.

## Artifact totals

The active release contract is:

- 13 platform artifacts;
- 16 public files including release metadata/checksums.
