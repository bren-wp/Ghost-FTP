# Ghost FTP documentation

This directory contains the maintained product, architecture, security, privacy, operator, packaging and release documentation for **Ghost FTP**.

**Current Ghost FTP release: 0.2.1**

Development status: **Beta**

Ghost FTP currently maintains one application product for two desktop platforms: **Windows and Linux**. The root `VERSION` is canonical for both platforms and for every Setup, Portable and Linux package produced by the release workflow.

## Start here

- [Installation](INSTALLATION.md) — Windows/Linux prerequisites, Setup, Portable and Linux packages.
- [Desktop reference UI](REFERENCE-UI.md) — maintained workspace hierarchy, authentic capture and native presentation rules.
- [Platform parity](PLATFORM-PARITY.md) — shared actions/security/settings and deliberate native presentation differences.
- [Localization](LOCALIZATION.md) — English-first 24-language registry and runtime/setup coverage.
- [Settings](SETTINGS.md) — real persisted options, safe defaults, validation and migration semantics.
- [Dependencies](DEPENDENCIES.md) — zero external Go modules, OS transport prerequisites and tracking-SDK prohibition.
- [Architecture](ARCHITECTURE.md) — shared typed engine and native Windows/Linux frontend boundaries.
- [Security](SECURITY.md) — credentials, host trust, path protections and transfer hardening.
- [Privacy](PRIVACY.md) — telemetry prohibition and runtime network/data boundaries.
- [Testing](TESTING.md) — automated quality, security, localization and production-build gates.
- [Versioning](VERSIONING.md) — the `0.x.y` Beta progression and the stable 1.0.0 gate.
- [Release history](RELEASE-HISTORY.md) — immutable historical engineering/release provenance.
- [GitHub Releases](GITHUB-RELEASES.md) — release channel, asset and tag policy.
- [Release verification](RELEASE-VERIFICATION.md) — SHA-256, metadata and provenance verification.
- [Signing](SIGNING.md) — Windows signing and Linux package provenance expectations.
- [Roadmap](ROADMAP.md) — desktop Beta stabilization and 1.0 quality priorities.
- [Third-party notices](THIRD-PARTY-NOTICES.md) — operating-system transport attribution boundary.
- [Contributing](CONTRIBUTING.md) — contribution workflow and quality requirements.
- [Support](SUPPORT.md) — issue reporting and support boundaries.
- [Linux packaging](../linux/README.md) — Linux build/package details.

## Active application platforms

### Windows

- native Win32 UI;
- normal resizable/minimizable/maximizable desktop window;
- x64 and x86 builds;
- Setup and Portable editions;
- one canonical dual-pane workspace;
- integrated Windows Installed Apps uninstall without a separate uninstall executable;
- the same shared transfer/security engine used by Linux.

### Linux

- native X11/XWayland-compatible graphical frontend;
- hardened terminal fallback;
- amd64, arm64 and i386 packages;
- the same typed `internal/api.Engine` used by Windows;
- the same connection, transfer, settings, profile, localization and security model;
- runtime selection from the shared 24-language registry.

Native Win32 and Linux controls are not claimed to be pixel-identical. Product actions, settings semantics, security boundaries, palette and versioning are kept aligned.

## Current product line

The maintained maturity sequence is:

```text
0.1.0 Beta → 0.1.1 Beta → 0.2.0 Beta → 0.2.1 Beta → 0.x.y Beta → 1.0.0 stable
```

Every `0.x.y` build is a Beta/prerelease. The first release that may be treated as stable is **1.0.0**, after the complete desktop quality gate is intentionally satisfied.

Windows Setup and Portable are packaging variants of the same application release and therefore always use the same canonical version.

Release tags use:

```text
ghostftp-vX.Y.Z
```

Published release tags are immutable.

## Release artifact contract

A complete Windows/Linux release contains **9 platform artifacts**:

1. Windows Setup x64
2. Windows Setup x86
3. Windows Setup x32 compatibility alias
4. Windows Portable x64
5. Windows Portable x86
6. Linux amd64 DEB
7. Linux arm64 DEB
8. Linux i386 DEB
9. Linux multiarch ZIP

The release also publishes:

- `RELEASE-NOTES.txt`
- `BUILD-METADATA.txt`
- `SHA256.txt`

This produces **12 public files** for a complete release.

## 0.2.0 documentation focus

Version 0.2.0 documents and verifies the desktop-only product contract introduced by the current source cleanup:

- one canonical Windows workspace instead of overlapping presentation layers;
- Linux graphical runtime language selection using the same 24-language registry;
- real settings parity for parallelism, conflict policy, retries, retry delay, timeout and delete confirmation;
- integrated Windows uninstall through the installed application;
- removal of retired non-desktop application source from the active tree;
- explicit distinction between zero external Go modules and OS-provided `curl`, `ssh` and `sftp` transport executables;
- fail-closed privacy/dependency/platform audits;
- authentic Windows UI screenshots generated from the real production executable.

## Documentation invariants

Maintained documentation must follow these rules:

- current guides describe Windows and Linux as the only supported application platforms;
- historical release notes remain historical facts and are not rewritten to pretend earlier releases had the current platform matrix;
- root `VERSION` remains the single machine-readable semantic version source;
- Windows Setup and Portable never have separate version numbers or divergent application functionality;
- remote permission metadata is shown only when supplied and validated from server listing/mode data;
- dependency guidance distinguishes Go module dependencies from operating-system prerequisites;
- privacy guidance does not imply an analytics/update service that the application does not have;
- Linux native presentation is allowed to differ from Win32 presentation while using the same typed Engine contract;
- the 24-language registry remains English-first with safe English fallback;
- release asset counts must match the release workflow;
- screenshots used as application evidence come from the authentic production capture workflow rather than generated mockups;
- published release tags remain immutable.
