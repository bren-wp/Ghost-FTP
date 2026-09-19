# Versioning

Ghost FTP uses one canonical version source: the repository root `VERSION` file.

Current source candidate: **0.0.8**

```text
VERSION=0.0.8
TAG=ghostftp-v0.0.8
CHANNEL=Current
PRERELEASE=false
```

Current package identity:

`ghcr.io/bren-wp/ghost-ftp:0.0.8`

The current **latest public version** is 0.0.8. Semantic-version major version `0` does not imply prerelease status; prerelease state is explicit.

## Tag format

`ghostftp-v<version>`

Published tags and release assets are immutable. If application behavior, UI, release-facing documentation, packaging or security behavior changes after publication, the next publication uses a higher version. Existing 0.0.8 assets are never rewritten.

## Active source platforms

```text
WINDOWS,LINUX,ANDROID
```

Browser helpers are companion packages and do not expand the native application-platform list.

## Development after a release

The repository may keep `VERSION` at the last published version while next-version work is reviewed. Before a new public release, `VERSION`, package names, documentation, release metadata, checksums and runtime evidence must move together to the new version.

The current post-0.0.8 work is the development line for the next public release. A new release is not created until exact-head validation succeeds.

## required public-release trust boundary

Official publication is fail-closed.

```text
WINDOWS_AUTHENTICODE=signed
```

Absence of the production Authenticode identity is a release failure. The protected Android publisher identity and exact signer fingerprint are likewise required by the protected production workflow.

The separate no-secret 0.0.8 distribution records its weaker signing state explicitly and is not treated as equivalent to protected production signing.

## Release retention

`.github/workflows/release-retention.yml` preserves the current version identity and the protected historical `ghostftp-v0.0.7` identity while removing superseded release/package state according to the documented retention policy.

## Retired platform

macOS source/build identity is retired. Current version validation, active source-platform metadata and future artifact counts do not include macOS.
