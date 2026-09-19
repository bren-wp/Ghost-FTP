# Ghost FTP documentation

This directory documents the active Ghost FTP product line: **Windows, Linux and Android**.

The immutable published release is **0.0.8**. Current `main` may contain next-version development work; published tags and release assets are never rewritten.

Current source version: **0.0.9**
Publication status is verified from GitHub Releases; repository documentation does not rewrite a published release identity.
Release channel: **Current**
Product status: **Current**
`PRERELEASE=false`

The next-line distribution contract is **9 platform artifacts / 12 public files**.

## Start here

- [Architecture](ARCHITECTURE.md)
- [Installation](INSTALLATION.md)
- [Reference UI](REFERENCE-UI.md)
- [Platform parity](PLATFORM-PARITY.md)
- [Settings](SETTINGS.md)
- [Navigation and bookmarks](NAVIGATION-BOOKMARKS.md)
- [Queue priority](QUEUE-PRIORITY.md)
- [Testing](TESTING.md)
- [Release verification](RELEASE-VERIFICATION.md)
- [Versioning](VERSIONING.md)
- [Security](SECURITY.md)
- [Privacy](PRIVACY.md)
- [Signing](SIGNING.md)
- [Support](SUPPORT.md)

## Active native applications

| Platform | Status |
| --- | --- |
| Windows | Active native desktop app |
| Linux | Active native desktop app |
| Android | Active native mobile app |

macOS is retired from the active product line. Its application source, Darwin-only implementation files, dedicated workflows and platform-specific tests are intentionally removed.

## UI evidence

Authentic runtime screenshots live under [`images/`](images/). Windows, Linux and Android runtime captures are bound to source/workflow provenance and are the only acceptable product UI evidence. Generated mockups or reference artwork must not be presented as execution evidence.

## Documentation rule

Current-product documentation must describe only shipping or maintained source behavior. Historical documents may mention older macOS work only when clearly describing historical releases rather than current support.
