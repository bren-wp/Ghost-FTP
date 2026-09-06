# Ghost FTP GitHub Packages

Ghost FTP publishes a verified **distribution bundle** to GitHub Packages for each stable release. The package is an OCI artifact stored in GitHub Container Registry (GHCR) and mirrors the exact verified release files assembled by the production release workflow.

## Package reference

```text
ghcr.io/bren-wp/ghost-ftp:<version>
```

For the Ghost FTP 1.1.1 Stable candidate, the canonical immutable version tag will be:

```text
ghcr.io/bren-wp/ghost-ftp:1.1.1
```

Stable publication also updates the compatible aliases `1.1`, `1`, and `latest` only after successful registry publication/read-back. Automation that requires reproducibility should use the full semantic version and, when possible, pin the registry digest.

Published 1.1.0 and 1.0.0 package versions remain historical distribution identities and are not rewritten for 1.1.1.

## What the package contains

The image contains the verified release directory under:

```text
/ghostftp-release/
```

That directory contains the same Windows Setup/Portable packages, Linux DEB packages, release notes, build metadata and `SHA256.txt` manifest published on the corresponding GitHub Release.

This is a **distribution bundle**, not a runtime container. Ghost FTP is a native desktop application for Windows and Linux; the GHCR package exists so CI systems, mirrors and administrators can retrieve a versioned, repository-linked bundle from GitHub Packages.

## Canonical installation source

For normal installation, use the files attached to the official GitHub Release. GitHub Packages is an additional verified distribution surface and does not replace Setup, Portable or DEB packages.

## Verification

Every stable package is produced only after the same release quality gates used for GitHub Releases:

- Go formatting, race tests and vet;
- security, privacy, dependency, platform and documentation audits;
- Windows x64/x86 Setup and Portable builds;
- Linux amd64/arm64/i386 DEB builds;
- release asset allow-list verification;
- SHA-256 manifest generation;
- Authenticode verification **when a trusted production certificate is configured**;
- explicit `WINDOWS_AUTHENTICODE=unsigned` metadata when no production certificate is configured.

Production signing is optional, but its state is never ambiguous. A configured trusted signing identity is verified fail-closed; absence of a production certificate does not cause Ghost FTP to fabricate a self-signed publisher identity or label unsigned files as signed.

The OCI package carries repository source, version and commit labels. The release workflow performs a registry read-back with `docker buildx imagetools inspect` after push.

## Privacy boundary

The package is built only from the already assembled `release/` allow-list. It does not contain saved profiles, passwords, local application data, CI secrets, signing key material, source worktrees or user files. The bundle is created with Docker networking disabled during the build itself.

## Release channels

Pre-1.0 historical builds were Beta prereleases. Beginning with Ghost FTP 1.0.0, official stable releases are normal GitHub Releases and the production workflow publishes the stable GHCR bundle. Stable package aliases are not published by the Beta branch of the release-channel logic.

## Digest-first automation

For high-assurance automation after 1.1.1 has actually been published:

1. resolve `ghcr.io/bren-wp/ghost-ftp:1.1.1` to its OCI digest;
2. pin the digest in downstream automation;
3. extract `/ghostftp-release/SHA256.txt`;
4. verify every release file before use;
5. compare the expected source commit with `BUILD-METADATA.txt` and the OCI revision label;
6. inspect `WINDOWS_AUTHENTICODE` before interpreting Windows publisher-signature state.

This provides two independent integrity references: the OCI manifest digest and the per-file SHA-256 manifest, plus an explicit signing-state declaration for Windows artifacts.

Do not treat the presence of this documentation as proof that 1.1.1 has already been published. Publication is complete only after the release workflow and remote GitHub Release/GHCR read-back succeed.
