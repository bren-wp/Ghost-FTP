# Ghost FTP GitHub Packages

Ghost FTP publishes a verified **distribution bundle** to GitHub Packages for each stable release. The package is an OCI artifact stored in GitHub Container Registry (GHCR) and mirrors the exact verified release files assembled by the production release workflow.

## Package reference

```text
ghcr.io/bren-wp/ghost-ftp:<version>
```

Ghost FTP **1.1.6 Stable is published**. Its canonical immutable version tag is:

```text
ghcr.io/bren-wp/ghost-ftp:1.1.6
```

The 1.1.6 publication also updated compatible aliases `1.1`, `1` and `latest` after successful registry publication and read-back. Automation that requires reproducibility should use the full semantic version and, when possible, pin the registry digest.

Published package versions through 1.1.6 are immutable historical distribution identities and are not rewritten by later source or packaging work.

## What the package contains

The image contains the verified release directory under:

```text
/ghostftp-release/
```

For 1.1.6, that directory contains the same Windows Setup/Portable packages, Linux DEB packages and Linux multiarch bundle, release notes, build metadata and `SHA256.txt` manifest published on the corresponding GitHub Release.

This is a **distribution bundle**, not a runtime container. Ghost FTP is a native desktop application for Windows and Linux; GHCR exists so CI systems, mirrors and administrators can retrieve a versioned, repository-linked release bundle.

Post-1.1.6 source/CI builds additionally produce package-manager-neutral Linux `.tar.gz` archives. They are not retroactive contents of the immutable 1.1.6 GHCR bundle; a later release may include them only after its own release-contract gates pass.

## Canonical installation source

For normal installation, use files attached to the official GitHub Release. GitHub Packages is an additional verified distribution surface and does not replace Setup, Portable or Linux packages.

The official product website is **https://ghostftp.com**. Ghost FTP is developed and published by **BRENDIGO LTD**, whose official website is **https://brendigo.com**.

## Verification

Every stable package is produced only after the same quality gates used for GitHub Releases:

- Go formatting, race tests and vet;
- security, privacy, dependency, repository, platform, localization and documentation audits;
- Windows x64/x86 Setup and Portable production builds;
- Linux production builds and package verification;
- release asset allow-list verification;
- SHA-256 manifest generation;
- Authenticode verification **when a trusted production certificate is configured**;
- explicit `WINDOWS_AUTHENTICODE=unsigned` metadata when no production certificate is configured;
- exact source/release version binding and post-publication read-back.

For the historical 1.1.6 release, the Linux public artifacts remain three DEBs plus the Linux multiarch ZIP. Current source/CI additionally validates distro-neutral Linux tarballs, but that does not mutate the already published bundle.

Production signing is optional, but its state is never ambiguous. A configured trusted signing identity is verified fail-closed; absence of a production certificate does not cause Ghost FTP to fabricate a self-signed publisher identity or label unsigned files as signed.

The OCI package carries repository source, version and commit labels. The release workflow performs registry read-back after push.

## Privacy boundary

The package is built only from the already assembled `release/` allow-list. It does not contain saved profiles, passwords, private-key passphrases, local application data, CI secrets, signing private-key material, source worktrees or user files. The bundle is created with Docker build networking disabled.

## Digest-first automation

For the published 1.1.6 package:

1. resolve `ghcr.io/bren-wp/ghost-ftp:1.1.6` to its OCI digest;
2. pin that digest in downstream automation where practical;
3. extract `/ghostftp-release/SHA256.txt`;
4. verify every release file before use;
5. compare the expected source commit with `BUILD-METADATA.txt` and the OCI revision label;
6. inspect `WINDOWS_AUTHENTICODE` before interpreting Windows publisher-signature state.

This provides two integrity references: the OCI manifest digest and the per-file SHA-256 manifest, plus an explicit Windows signing-state declaration.

Ghost FTP 1.1.6 publication is complete: the canonical release path succeeded and remote GitHub Release/GHCR read-back confirmed the final state. Future releases must independently satisfy the same fail-closed publication contract.
