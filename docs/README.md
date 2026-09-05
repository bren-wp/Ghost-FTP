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
- [Release history](RELEASE-HISTORY.md) — detailed immutable release history and current development scope.
- [GitHub Releases](GITHUB-RELEASES.md) — 2.x release assets and version/tag policy.
- [Release verification](RELEASE-VERIFICATION.md) — SHA-256, metadata and provenance verification.
- [Signing](SIGNING.md) — Windows signing and Linux package provenance expectations.
- [Shared hosting](SHARED-HOSTING.md) — separate Web companion deployment guidance.
- [Roadmap](ROADMAP.md) — Windows/Linux product direction and quality priorities.
- [Third-party notices](THIRD-PARTY-NOTICES.md) — operating-system transport components and attribution boundary.
- [Contributing](CONTRIBUTING.md) — contribution workflow and quality expectations.
- [Support](SUPPORT.md) — issue reporting and support boundaries.

## Active application platforms

Ghost FTP 2.x supports:

- **Windows** — native Win32 GUI, Setup and portable packages for x64/x86.
- **Linux** — shared core with hardened terminal frontend and DEB packages for amd64/arm64/i386.

Android, iOS and macOS application targets were retired from active 2.x source. Historical 1.x tags/releases remain available for provenance and must not be rewritten.

The existing **Ghost FTP Web companion** remains in the repository as a separate shared-hosting/PWA source surface. It is not counted as a Windows/Linux desktop application artifact in the 2.x release contract.

## Current product line

**Current Ghost FTP release: 2.0.0**

The 2.x line uses tags named `ghostftp-vX.Y.Z`. The platform consolidation is a semantic-major change because supported application targets changed.

A complete 2.x desktop release contains **9 platform artifacts** plus `RELEASE-NOTES.txt`, `BUILD-METADATA.txt` and `SHA256.txt`, for **12 public files** total.

The active 2.x release must pass the shared quality/security/documentation gate and both Windows and Linux production build gates before publication.

## Platform documentation

- [Windows release/build information](../README.md#windows-experience)
- [Linux packaging](../linux/README.md)
- [Windows/Linux parity](PLATFORM-PARITY.md)
- [Web companion](../GhostFTP%20WEB/README.md)

## Documentation invariants

Long-lived documentation must describe the current product contract without deleting historical release facts. In particular:

- old 1.x release notes may mention platforms that existed at the time;
- current installation, architecture, release and roadmap documents must not advertise retired application platforms as supported;
- dependency documentation must distinguish zero external Go modules from OS-provided runtime transport tools;
- privacy documentation must not imply communication with an application analytics/update service that does not exist;
- release asset counts must match the workflow contract.
