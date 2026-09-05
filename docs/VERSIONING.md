# Versioning policy

Ghost FTP uses one canonical desktop application version from the root `VERSION` file. Windows Setup, Windows Portable, Linux packages, release tags, release notes and build metadata must all resolve from that same value.

## Current baseline

The active development line starts at **0.1.0**.

All versions in the `0.x.y` range are **Beta** builds. They are intended for active development, testing, compatibility work and stabilization. They may be distributed for evaluation, but they are not represented as the first fully stable Ghost FTP release.

The first release that may be presented as fully stable is **1.0.0**. Moving to 1.0.0 is a product-quality decision, not merely a calendar or feature-count milestone.

## Version progression

The normal sequence is:

```text
0.1.0 Beta
0.1.1 Beta
0.2.0 Beta
0.3.0 Beta
...
0.9.x Beta
1.0.0
1.0.1
1.1.0
...
```

Patch increments (`0.1.1`, `0.1.2`) are used for compatible fixes and refinements inside the same development milestone. Minor increments (`0.2.0`, `0.3.0`) mark meaningful development milestones or capability groups. Before 1.0.0, compatibility may still evolve when required for correctness, security or architecture cleanup, but changes must remain documented.

## Beta rule

A build whose canonical major version is `0` is a Beta build.

The machine-readable value remains plain semantic versioning such as `0.1.0`. This is required for:

- Windows PE file/product version metadata;
- Debian package metadata;
- NuGet/GitHub Package versions;
- release workflow comparison and validation;
- `ghostftp-vX.Y.Z` tag generation;
- update/release automation that expects numeric `X.Y.Z` values.

The user-facing application version may append the word `Beta`, for example **0.1.0 Beta**. This presentation label must never replace the canonical numeric value inside package metadata.

GitHub Releases created from a `0.x.y` version are prereleases. The release workflow must not publish a `0.x.y` build as a stable release.

## Stable 1.0.0 gate

Ghost FTP moves to **1.0.0** only when the maintained product is considered complete and stable enough for the first stable public line. At minimum, the following areas must be release-ready:

- Windows native UI and Site Manager behavior;
- Linux maintained frontend and shared-core parity;
- FTP, FTPS and SFTP connection reliability;
- SFTP host-key trust and credential handling;
- upload/download correctness and transfer queue lifecycle;
- overwrite, retry, cancellation and recovery behavior;
- Windows Setup transaction, rollback and upgrade behavior;
- Windows Portable startup and runtime behavior;
- supported localization coverage and fallback behavior;
- security, privacy and dependency audits;
- release artifact verification and SHA-256 manifest generation;
- documentation, installation and support guidance;
- successful Windows and Linux production CI gates.

No individual feature completion automatically changes the version to 1.0.0. The stable cut happens only after the complete release contract is satisfied.

## Windows Setup and Portable

Setup and Portable are two packaging forms of the same Ghost FTP release. They must always carry the **same canonical version**.

For the current baseline this means:

```text
Ghost-FTP-0.1.0-Setup-x64.exe
Ghost-FTP-0.1.0-Setup-x86.exe
Ghost-FTP-0.1.0-Portable-x64.exe
Ghost-FTP-0.1.0-Portable-x86.exe
```

When the first stable version is reached, the corresponding packages become:

```text
Ghost-FTP-1.0.0-Setup-x64.exe
Ghost-FTP-1.0.0-Setup-x86.exe
Ghost-FTP-1.0.0-Portable-x64.exe
Ghost-FTP-1.0.0-Portable-x86.exe
```

The release workflow also creates the documented x32 compatibility alias of the x86 Setup build. The alias does not have an independent version.

## Canonical source of truth

Do not hard-code a production version in Go source, documentation templates, workflow defaults or package templates. The root `VERSION` file is authoritative.

Production builds inject that value into the application and installer at link time. Build scripts and CI validate that Windows and Linux package metadata match it exactly.

## Historical repository versions

Existing tags, releases, commits and changelog entries from earlier repository work are retained as immutable historical provenance. They are not deleted or rewritten.

The current **0.1.0 → 1.0.0** policy is the active product-development baseline going forward. Historical version numbers document earlier repository states; they do not change the active baseline and must not be used to skip the Beta stabilization process.

## Release tags

Current release tags remain namespaced:

```text
ghostftp-vX.Y.Z
```

Examples:

```text
ghostftp-v0.1.0
ghostftp-v0.2.0
ghostftp-v1.0.0
```

Published tags are immutable. A tag already pointing to a released commit must never be moved to another commit.
