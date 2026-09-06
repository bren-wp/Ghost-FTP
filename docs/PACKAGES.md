# Ghost FTP GitHub Packages

Ghost FTP publishes a verified **distribution bundle** to GitHub Packages for each maintained stable release. The package is an OCI artifact in GitHub Container Registry (GHCR) and mirrors the exact verified release files assembled by the production workflow.

## Package reference

```text
ghcr.io/bren-wp/ghost-ftp:<version>
```

For Ghost FTP 1.0.0:

```text
ghcr.io/bren-wp/ghost-ftp:1.0.0
```

Stable aliases are:

```text
1.0
1
latest
```

Automation that requires reproducibility should use the full semantic version and preferably pin the immutable OCI digest returned by the registry.

## What the package contains

The OCI bundle contains the verified release directory under:

```text
/ghostftp-release/
```

That directory contains the same Windows Setup/Portable files, Linux DEB packages, multiarch ZIP, release notes, `BUILD-METADATA.txt` and `SHA256.txt` published on the matching GitHub Release.

This is a **distribution bundle**, not a runtime container. Ghost FTP remains a native Windows/Linux desktop application. The GHCR object exists for automation, mirrors, archival workflows and integrity verification.

## Publication order

The production workflow verifies the GitHub Release before publishing GHCR. This prevents a package-only success from being treated as a completed product release.

Before the package push, the workflow verifies again that current `main` still equals the exact release commit.

## Isolated package build

The bundle image uses:

```text
FROM scratch
```

and copies only the already assembled `release/` allow-list. Docker build networking is disabled with `--network=none`.

The build context does not include saved profiles, user application data, signing private keys or runtime FTP/SFTP credentials.

## Registry authentication privacy

GHCR authentication uses the short-lived GitHub Actions token through `docker login --password-stdin`. The workflow points Docker at a temporary `DOCKER_CONFIG` directory, logs out and removes that directory at the end of the step.

The package token is not copied into the OCI filesystem or release metadata.

## OCI provenance labels

The semantic-version package carries labels for:

- Ghost FTP package identity;
- source repository;
- semantic version;
- source commit/revision;
- proprietary/source-available license identity.

These labels are verified again after registry pull.

## Strong read-back verification

Publishing is not considered complete merely because `docker push` returned success. The workflow:

1. pushes `1.0.0`, `1.0`, `1` and `latest`;
2. verifies the remote semantic-version manifest is readable;
3. removes the local package image/tags;
4. pulls `ghcr.io/bren-wp/ghost-ftp:1.0.0` from GHCR;
5. checks source/version/revision OCI labels;
6. copies `/ghostftp-release/SHA256.txt` from the pulled package and compares it byte-for-byte with the verified release manifest;
7. copies `/ghostftp-release/BUILD-METADATA.txt` and compares it byte-for-byte with the verified release metadata;
8. records the immutable repository digest in the workflow summary.

The successful gate emits `PACKAGE_READBACK=PASS`.

## Windows signing metadata inside the package

The package mirrors the truthful Windows release state from `BUILD-METADATA.txt`.

A release may contain:

```text
WINDOWS_AUTHENTICODE=signed
WINDOWS_TRUST_MODE=authenticode+sha256+github-provenance
```

or, when no protected production certificate is configured:

```text
WINDOWS_AUTHENTICODE=unsigned
WINDOWS_TRUST_MODE=sha256+github-release-provenance
```

The GHCR bundle does not create, add or fake a Windows signature. It packages only the already verified release files.

## Canonical installation source

Normal users should install from the files attached to the official GitHub Release. GitHub Packages is an additional verified distribution surface and does not replace Windows Setup/Portable or Linux DEB installation workflows.

## Digest-first automation

For high-assurance automation:

1. resolve `ghcr.io/bren-wp/ghost-ftp:1.0.0` to its OCI digest;
2. pin that digest downstream;
3. extract `/ghostftp-release/SHA256.txt` and `/ghostftp-release/BUILD-METADATA.txt`;
4. verify every file against `SHA256.txt`;
5. compare the source commit in metadata with the OCI revision label and GitHub Release tag.

This gives two independent integrity references: the OCI manifest digest and the per-file SHA-256 manifest.

## Release-channel rule

The maintained production workflow is stable-only and rejects `MAJOR=0`, so it does not publish new prerelease packages. Historical pre-1.0 GitHub Releases remain available as history but do not receive current stable aliases.

See [GitHub Releases](GITHUB-RELEASES.md), [Release verification](RELEASE-VERIFICATION.md), [Privacy](PRIVACY.md) and [Signing](SIGNING.md).
