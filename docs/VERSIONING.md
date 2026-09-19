# Versioning

Ghost FTP uses one canonical version source: the repository root `VERSION` file.

Current source candidate: **0.0.9**

```text
VERSION=0.0.9
TAG=ghostftp-v0.0.9
CHANNEL=Current
PRERELEASE=false
```

Current package identity:

`ghcr.io/bren-wp/ghost-ftp:0.0.9`

The source release candidate is **0.0.9**. Semantic-version major version `0` does not imply prerelease status; prerelease state is explicit.

## Tag format

`ghostftp-v<version>`

Published tags and release assets are immutable. If application behavior, UI, release-facing documentation, packaging or security behavior changes after publication, the next publication uses a higher version. Existing published assets are never rewritten.

## Active source platforms

```text
WINDOWS,LINUX,ANDROID
```

Browser extensions are retired and are not part of the active source or release surface.

## Development after a release

The repository may keep `VERSION` at the last published version while next-version work is reviewed. Before a new public release, `VERSION`, package names, documentation, release metadata, checksums and runtime evidence must move together to the new version.

0.0.9 is published only from an exact verified main commit. After publication, every subsequent source, UI, behavior, documentation, packaging or release-metadata change advances the version again.

## required public-release trust boundary

Official publication is fail-closed.

```text
WINDOWS_AUTHENTICODE=signed
```

Absence of the production Authenticode identity is a release failure. The protected Android publisher identity and exact signer fingerprint are likewise required by the protected production workflow.


## Release retention

`.github/workflows/release-retention.yml` preserves the current version identity and the protected historical `ghostftp-v0.0.7` identity while removing superseded release/package state according to the documented retention policy.

## Retired platform

macOS source/build identity is retired. Current version validation, active source-platform metadata and future artifact counts do not include macOS.

## Windows architecture evidence

```text
WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64
WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci
```

The public Windows Setup and Portable launchers carry x64, x86 and ARM64 native payloads. CI verifies the ARM64 payload structure and PE identity, but does not claim native ARM64 runtime execution; that limitation is recorded explicitly by `WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci`.
