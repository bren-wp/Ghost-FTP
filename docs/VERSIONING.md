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
