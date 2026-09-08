# Ghost FTP GitHub Packages

Ghost FTP publishes a verified **distribution bundle** to GitHub Packages for each stable release. The package is an OCI artifact stored in GitHub Container Registry (GHCR) and mirrors the exact verified release files assembled by the production release workflow.

## Package reference

```text
ghcr.io/bren-wp/ghost-ftp:<version>
```

Ghost FTP **1.1.7 Stable is published**. Its canonical immutable version tag is:

```text
ghcr.io/bren-wp/ghost-ftp:1.1.7
```

Compatible aliases `1.1`, `1` and `latest` are updated only after the semantic-version package has been published and read back successfully. Automation that requires reproducibility should use the full semantic version and, when possible, pin the OCI digest.

Historical package versions, including 1.1.6, remain immutable distribution identities and are not rewritten by later releases.

## What the package contains

The OCI object contains the verified release directory under:

```text
/ghostftp-release/
```

For 1.1.7 that directory mirrors the canonical **12 platform artifacts / 15 public files** GitHub Release assembly:

- five Windows Setup/Portable files;
- Linux DEBs for amd64, arm64 and i386;
- the Linux multiarch ZIP;
- Linux portable tar.gz archives for amd64, arm64 and i386;
- `BUILD-METADATA.txt`;
- `RELEASE-NOTES.txt`;
- `SHA256.txt`.

This is a **distribution bundle**, not a runtime container. Ghost FTP remains a native Windows/Linux desktop application.

## Canonical release-bundle contract

Before 1.1.7 can be accepted as published, the production workflow must:

- build DEB and `.tar.gz` outputs for amd64, arm64 and i386;
- validate DEB metadata and portable archive structure;
- prove matching DEB/portable `ghostftp` executables are byte-identical;
- match the exact 15-file release allow-list;
- generate `SHA256.txt` over the assembly;
- publish the GitHub Release with `prerelease=false`;
- perform immediate and delayed GitHub Release asset read-back;
- build the GHCR object from only the verified `release/` directory with build networking disabled;
- perform registry read-back before compatible aliases are updated.

Supplemental distro-labelled Debian/Ubuntu/Fedora/Portable CI artifacts are not copied into this bundle unless a later canonical release explicitly adds them to its allow-list.

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

Production signing is optional, but its state is never ambiguous. A configured trusted signing identity is verified fail-closed; absence of a production certificate does not cause Ghost FTP to fabricate a self-signed publisher identity or label unsigned files as signed.

The OCI package carries source, version and revision labels. Release CI verifies the package after push.

## Privacy boundary

The package is built only from the already assembled `release/` allow-list. It does not contain saved profiles, passwords, private-key passphrases, local application data, CI secrets, signing private-key material, source worktrees or user files.

## Digest-first automation

For Ghost FTP 1.1.7:

1. resolve `ghcr.io/bren-wp/ghost-ftp:1.1.7` to its OCI digest;
2. pin that digest where practical;
3. extract `/ghostftp-release/SHA256.txt`;
4. verify every release file;
5. compare `BUILD-METADATA.txt` source/version/tag with the OCI labels;
6. inspect `WINDOWS_AUTHENTICODE` before interpreting Windows publisher-signature state.

This gives two integrity references: the OCI manifest digest and the per-file SHA-256 manifest, plus an explicit Windows signing-state declaration.

The immutable 1.1.6 GHCR object retains its historical 9-platform-artifact/12-public-file shape. 1.1.7 expands only the new versioned bundle.
