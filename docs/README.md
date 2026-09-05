# Ghost FTP documentation

This directory contains maintained product, security, operator and release documentation for **Ghost FTP**.

## Start here

- [Installation](INSTALLATION.md) — Windows/Linux packages, prerequisites and installation expectations.
- [Platform parity](PLATFORM-PARITY.md) — exact Windows/Linux functional parity, presentation differences and parity rules.
- [Localization](LOCALIZATION.md) — English-first 24-language registry and runtime/setup coverage.
- [Settings](SETTINGS.md) — persisted options, safe defaults, validation and migration semantics.
- [Dependencies](DEPENDENCIES.md) — standard-library-only Go module policy, OS transport prerequisites and no-tracking rules.
- [Architecture](ARCHITECTURE.md) — shared core, Windows/Linux frontend boundaries and Web companion separation.
- [Security](SECURITY.md) — threat boundaries, credentials, host trust, filesystem and transfer hardening.
- [Privacy](PRIVACY.md) — telemetry prohibition and data/network handling policy.
- [Testing](TESTING.md) — automated quality, security and Windows/Linux build gates.
- [Versioning](VERSIONING.md) — active 0.1.0 Beta baseline, 0.x progression and stable 1.0.0 gate.
- [Release history](RELEASE-HISTORY.md) — current development scope plus immutable historical release provenance.
- [GitHub Releases](GITHUB-RELEASES.md) — prerelease/stable channel rules, release assets and tag policy.
- [Release verification](RELEASE-VERIFICATION.md) — SHA-256, metadata and provenance verification.
- [Signing](SIGNING.md) — Windows signing and Linux package provenance expectations.
- [Shared hosting](SHARED-HOSTING.md) — separate Web companion deployment guidance.
- [Roadmap](ROADMAP.md) — Windows/Linux Beta stabilization and 1.0 quality priorities.
- [Third-party notices](THIRD-PARTY-NOTICES.md) — operating-system transport components and attribution boundary.
- [Contributing](CONTRIBUTING.md) — contribution workflow and quality expectations.
- [Support](SUPPORT.md) — issue reporting and support boundaries.

## Active application platforms

Ghost FTP currently maintains:

- **Windows** — native Win32 GUI, Setup and Portable packages for x64/x86.
- **Linux** — shared core with hardened terminal frontend and DEB packages for amd64/arm64/i386.

Android, iOS and macOS application targets are not part of the active source/build matrix. Historical commits, tags and releases remain available for provenance and must not be rewritten.

The existing **Ghost FTP Web companion** remains in the repository as a separate shared-hosting/PWA source surface. It is not counted as a Windows/Linux desktop application artifact in the desktop release contract.

## Current product line

**Current Ghost FTP release: 0.1.0**

Development status: **Beta**

The active product baseline begins at **0.1.0 Beta**. Every `0.x.y` build remains Beta/prerelease while the application is being completed and stabilized. The first version that may be treated as stable is **1.0.0**.

Windows Setup and Portable are packaging variants of the same release and always use the same canonical `VERSION`.

Release tags use `ghostftp-vX.Y.Z`. Published tags remain immutable.

A complete desktop release contains **9 platform artifacts** plus `RELEASE-NOTES.txt`, `BUILD-METADATA.txt` and `SHA256.txt`, for **12 public files** total.

Every release must pass the shared quality/security/documentation gate and both Windows and Linux production build gates before publication. Pre-1.0 releases are additionally required to be published as GitHub prereleases.

## Platform documentation

- [Windows release/build information](../README.md#windows-experience)
- [Linux packaging](../linux/README.md)
- [Windows/Linux parity](PLATFORM-PARITY.md)
- [Web companion](../GhostFTP%20WEB/README.md)

## Documentation invariants

Long-lived documentation must describe the current product contract without deleting historical facts. In particular:

- historical release notes may mention platforms, package matrices or versions that existed at the time;
- current installation, architecture, release and roadmap documents must describe the maintained Windows/Linux contract;
- current versioning documentation must preserve the `0.1.0 → 0.x.y Beta → 1.0.0 stable` policy;
- Windows Setup and Portable must never claim independent release versions;
- dependency documentation must distinguish zero external Go modules from OS-provided runtime transport tools;
- privacy documentation must not imply communication with an application analytics/update service that does not exist;
- release asset counts must match the workflow contract;
- historical tags/releases remain immutable and are never rewritten merely because the active maturity baseline changed.
