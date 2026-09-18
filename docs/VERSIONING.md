# Versioning

Ghost FTP uses one canonical version source: the repository root `VERSION` file.

## Tag format

`ghostftp-v<version>`

Example:

`ghostftp-v0.0.8`

## Immutability

Published tags and release assets are immutable. If application behavior, UI, documentation that affects release truth, packaging or security changes after a release, the next publication uses a higher version.

## Active source platforms

```text
WINDOWS,LINUX,ANDROID
```

Browser helpers are companion packages and do not expand the native application-platform list.

## Development after a release

The repository may keep `VERSION` at the last published value while next-version work is under review, but a new public release must update `VERSION`, documentation, metadata and artifact names together before tagging.

## Retired platform

macOS source/build identity is retired. Current versioning and artifact counts do not include macOS.


## Current source and release identity

- Current source candidate: **0.0.8**
- Last actually published GitHub Release: **0.0.8**
- Next public release after post-0.0.8 source changes: **0.0.9**
- Release channel: **Current**
- Prerelease: **false**

Canonical metadata:

```text
VERSION=0.0.8
TAG=ghostftp-v0.0.8
CHANNEL=Current
PRERELEASE=false
GHCR=ghcr.io/bren-wp/ghost-ftp:0.0.8
```

The major version `0` does not imply prerelease. The latest public version is whatever immutable current GitHub Release and matching tag were actually published; source work after that publication does not rewrite that release.

## Release retention and signing boundary

The `.github/workflows/release-retention.yml` workflow preserves the current release contract and protected historical release identities.

The protected production workflow is the required public-release trust boundary:

- `WINDOWS_AUTHENTICODE=signed`
- Android production signing identity is required and verified.
- Absence of the production Authenticode identity is a release failure.
- Linux packages are built from the exact release source and verified by checksums.
- macOS is not an active release target and is not part of future artifact counts.

The separately documented no-secret distribution path has an explicitly weaker publisher-trust state and must never be presented as equivalent to the protected production-signing path.


## Windows architecture evidence

Ghost FTP Windows Setup and Portable are universal launchers with native **x64, x86 and ARM64** application payloads.

```text
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```

The ARM64 payload is built and structurally verified in CI. The metadata value above is intentionally explicit: current hosted CI does not claim native Windows-on-ARM runtime execution evidence.
