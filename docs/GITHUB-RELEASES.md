# Ghost FTP GitHub Releases

Ghost FTP **1.1.8 Stable** is the current published stable release contract. Ghost FTP **1.1.7 Stable** is the previous immutable maintenance release; all published Stable tags/releases remain immutable historical identities. Official releases are created only by the canonical release workflow from the exact verified `main` commit.

## Release identity

The 1.1.8 Stable identity is:

```text
Tag: ghostftp-v1.1.8
Title: Ghost FTP 1.1.8
Prerelease: false
```

The release workflow reads `VERSION` directly and rejects a manually supplied expected version that differs from source.

## Stable publication rule

A version with major number 1 or greater is Stable. Stable GitHub Releases use `prerelease=false`. Historical Beta/prerelease identities and prior Stable tags are not moved, reused or rewritten.

## Canonical release trigger

`release.yml` is intentionally `workflow_dispatch`-only. **A push to `main`, including a commit that changes `VERSION`, must not publish a release directly.**

The canonical release-branch namespace is `release/ghostftp-vX.Y.Z`. For 1.1.8, release preparation must first pass exact-head Core, Windows, Linux, distro and authentic Windows UI evidence. After merge, the exact resulting `main` SHA must pass the post-merge gates. Only then may `release/ghostftp-v1.1.8` be created at that same SHA.

`.github/workflows/release-branch-trigger.yml` validates both branch-to-main SHA equality and branch-version-to-`VERSION` equality before dispatching `.github/workflows/release.yml` with the expected version guard. This prevents duplicate or premature publication from an ordinary version bump.

## Published 1.1.8 public files

Ghost FTP 1.1.8 exposes **12 platform artifacts**.

Windows:

```text
Ghost-FTP-1.1.8-Setup-x64.exe
Ghost-FTP-1.1.8-Setup-x86.exe
Ghost-FTP-1.1.8-Setup-x32.exe
Ghost-FTP-1.1.8-Portable-x64.exe
Ghost-FTP-1.1.8-Portable-x86.exe
```

Linux:

```text
Ghost-FTP-1.1.8-Linux-amd64.deb
Ghost-FTP-1.1.8-Linux-arm64.deb
Ghost-FTP-1.1.8-Linux-i386.deb
Ghost-FTP-1.1.8-Linux-multiarch.zip
Ghost-FTP-1.1.8-Linux-amd64.tar.gz
Ghost-FTP-1.1.8-Linux-arm64.tar.gz
Ghost-FTP-1.1.8-Linux-i386.tar.gz
```

Verification/metadata:

```text
BUILD-METADATA.txt
RELEASE-NOTES.txt
SHA256.txt
```

That is **15 public files** total. Ghost FTP 1.1.8 preserves the canonical 12/15 release shape introduced in 1.1.7; the immutable 1.1.7 release remains unchanged. The older immutable 1.1.6 release remains at its historical 9-platform-artifact/12-public-file shape.

## Linux portable parity gate

For every Linux architecture the production job requires both a `.deb` and `.tar.gz`, validates DEB metadata, validates portable archive structure and compares the DEB-installed `ghostftp` executable with the portable archive executable byte-for-byte. The publish job stages the exact 15-file allow-list, hashes it and requires both immediate and delayed remote asset read-back to match exactly.

Supplemental distro-specific Debian/Ubuntu/Fedora/Portable CI packages built by `linux/BUILD-DISTROS.sh` remain **not yet part of the canonical release allow-list**. Their build/parity and native install evidence does not change the public 1.1.8 file count.

## Exact-head rule

Before publication, the workflow queries current `main` and requires it to equal the release source SHA. It verifies the condition again after publication. If `main` moves during the transaction, publication fails instead of attaching files to stale source.

## Immutable tag rule

The release workflow fails if the requested tag or GitHub Release already exists. Existing tags are not moved, deleted, reused or force-pushed, and published assets are not overwritten or clobbered.

## Public product identity

Active runtime, package, support and release metadata use only:

- product: **Ghost FTP**;
- official product website: **https://ghostftp.com**.

Author/publisher identity is intentionally confined to the application's About surface. GitHub is release/source infrastructure, not the product homepage shown in runtime/package metadata.

## Windows signing state

Authenticode signing is optional for Stable publication. If protected production signing secrets are configured, Windows Setup/Portable artifacts are signed and every signature must verify. If no production certificate is configured, Windows artifacts are published unsigned and `BUILD-METADATA.txt` records `WINDOWS_AUTHENTICODE=unsigned`.

The workflow never generates a self-signed production publisher identity and never labels an unsigned artifact as signed.

## Artifact allow-list

The publish job assembles a fresh `release/` directory from only verified Windows and Linux staging artifacts plus generated notes/metadata/checksums. Metadata records:

```text
LINUX_PORTABLE=amd64,arm64,i386
PUBLIC_PLATFORM_ARTIFACTS=12
PUBLIC_RELEASE_FILES=15
```

`Ghost-FTP-1.1.8-Setup-x32.exe` is intentionally a byte-identical compatibility alias of the verified x86 Setup artifact and is not a separate architecture build.

## Read-back verification

After creating the Release, the workflow reads the remote asset set and compares it with the expected sorted allow-list. It requires `prerelease=false` and repeats a delayed read-back to catch asynchronous publication issues.

A local build is not release evidence. Remote tag, Release assets and package registry state must agree with the verified source revision.

## GitHub Packages

Stable 1.1.8 is published at:

```text
ghcr.io/bren-wp/ghost-ftp:1.1.8
```

Compatible Stable aliases are updated only after semantic-version package publication and registry read-back:

```text
1.1
1
latest
```

The registry object is an OCI distribution bundle, not a runtime container. It copies only the verified `release/` assembly with build networking disabled and carries source/version/revision labels.

See [Packages](PACKAGES.md).

## 1.1.8 release evidence

The 1.1.8 line includes privacy-safe child-process diagnostic classification, strict Linux transport and AskPass executable provenance, state-directory identity pinning, Windows installer/uninstaller/shortcut ownership and exact-object cleanup, and monitor-work-area/mixed-DPI responsive geometry. It preserves FTPS/SFTP trust, rooted transfer/filesystem protections, the 24-language contract and the 12/15 release shape.

Authentic Main Workspace, Site Manager, Settings and About screenshots must be generated from the real production Windows x64 Portable executable on the exact final release-prep revision and visually reviewed. Mockups or generated approximations are not accepted.

## Failure behavior

A failed quality gate, production build, configured-signing verification, package push, tag validation, Release creation or read-back check causes the workflow to fail. Absence of a production Authenticode certificate alone does not fail publication; that state is preserved as `unsigned` metadata.

See [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md) and [Versioning](VERSIONING.md).
