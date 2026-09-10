# Ghost FTP GitHub Releases

Ghost FTP **0.0.2** is the current published release contract. Official releases are created only by the canonical release workflow from the exact verified `main` commit.

## Release identity

```text
Tag: ghostftp-v0.0.2
Title: Ghost FTP 0.0.2
Prerelease: false
```

Root `VERSION` is authoritative. The workflow rejects a manually supplied version that differs from source. Major version zero is not automatically mapped to GitHub prerelease state for this project.

## Current publication rule

The current 0.0.x public line publishes with `prerelease=false`. Any future change to prerelease policy must be explicit across the workflow, audits and documentation rather than inferred from the semantic-version major component.

A normal push to `main`, including a `VERSION` change, does not publish a release directly.

## Canonical release trigger

`release.yml` is `workflow_dispatch`-only. The canonical release branch namespace is:

```text
release/ghostftp-vX.Y.Z
```

For the current candidate:

```text
release/ghostftp-v0.0.2
```

`.github/workflows/release-branch-trigger.yml` accepts the branch only when its semantic version matches root `VERSION` and its SHA equals exact current `main`.

The trigger then performs a deterministic lifecycle rather than treating dispatch as success:

1. snapshot the existing `release.yml` workflow-dispatch run IDs;
2. dispatch canonical `release.yml` on `main` with the exact version guard;
3. discover the newly created `Publish Ghost FTP` run whose `headSha` equals the validated `main` SHA;
4. wait for that exact run with `gh run watch --exit-status`;
5. verify its terminal conclusion is `success`;
6. only then snapshot existing retention runs and explicitly dispatch canonical `release-retention.yml` on `main`;
7. discover the new exact-main retention run;
8. wait for it and require terminal `success` before the release-branch trigger itself can succeed.

This explicit completion chain exists because a workflow dispatched with the repository `GITHUB_TOKEN` must not rely on a downstream `workflow_run` notification as its only retention path. `release-retention.yml` keeps its `workflow_run` trigger as defense in depth, but the release branch lifecycle is successful only after canonical retention is observed successfully.

The trigger never force-moves release identities and never dispatches retention before the canonical release run succeeds.

## 0.0.2 public files

Ghost FTP 0.0.2 exposes **12 platform artifacts**.

Windows:

```text
Ghost-FTP-0.0.2-Setup-x64.exe
Ghost-FTP-0.0.2-Setup-x86.exe
Ghost-FTP-0.0.2-Setup-x32.exe
Ghost-FTP-0.0.2-Portable-x64.exe
Ghost-FTP-0.0.2-Portable-x86.exe
```

Linux:

```text
Ghost-FTP-0.0.2-Linux-amd64.deb
Ghost-FTP-0.0.2-Linux-arm64.deb
Ghost-FTP-0.0.2-Linux-i386.deb
Ghost-FTP-0.0.2-Linux-multiarch.zip
Ghost-FTP-0.0.2-Linux-amd64.tar.gz
Ghost-FTP-0.0.2-Linux-arm64.tar.gz
Ghost-FTP-0.0.2-Linux-i386.tar.gz
```

Verification/metadata:

```text
BUILD-METADATA.txt
RELEASE-NOTES.txt
SHA256.txt
```

That is **15 public files** total.

## Exact-head rule

Before publication the release workflow compares current `main` with the release workflow source SHA. It performs the comparison again after publication before the delayed remote asset read-back. If `main` moves, publication fails rather than claiming stale source.

The release-branch trigger also filters the newly dispatched publish and retention workflow runs by the validated exact `main` SHA. An older successful workflow run cannot satisfy a new release lifecycle.

## Immutable-current publication transaction

The requested `ghostftp-v0.0.2` tag/release must not already exist. The publish workflow never clobbers a release asset or rewrites an existing current release tag.

After the new release is successfully published and remotely verified, `.github/workflows/release-retention.yml` enforces the project policy that **only the latest public Ghost FTP version remains**. It removes older `ghostftp-v*` GitHub Releases, older/orphan `ghostftp-v*` tags, superseded `release/ghostftp-v*` branches and obsolete Ghost FTP container package versions.

The current release branch and current package version are retained. Cleanup independently verifies that the current release is non-draft, `prerelease=false`, has exactly 15 assets and points to current `main` before destructive cleanup. Repository commit history on `main` is not rewritten.

## Failure behavior

The release lifecycle fails closed if the release branch does not match root `VERSION`, if its SHA is not exact current `main`, if the newly dispatched exact release or retention run cannot be identified, if either canonical run fails/cancels, or if the current release/tag/asset set does not pass retention preflight.

A successful workflow dispatch request by itself is **not** treated as successful publication.

## Linux portable parity gate

For each Linux architecture the production job requires both a `.deb` and `.tar.gz`, verifies DEB metadata and portable archive structure, and compares the installed/portable `ghostftp` executable byte-for-byte.

Supplemental distro-specific Debian/Ubuntu/Fedora/Portable CI packages built by `linux/BUILD-DISTROS.sh` remain **not yet part of the canonical release allow-list**.

## Windows signing state

Authenticode signing is optional. If protected production signing secrets are configured, all Windows artifacts must verify successfully. If no production certificate is configured, Windows artifacts remain explicitly unsigned and `BUILD-METADATA.txt` records:

```text
WINDOWS_AUTHENTICODE=unsigned
```

The workflow never creates a self-signed production identity and never labels an unsigned artifact as signed.

## Artifact allow-list

The publish job assembles a fresh release directory and records:

```text
LINUX_PORTABLE=amd64,arm64,i386
PUBLIC_PLATFORM_ARTIFACTS=12
PUBLIC_RELEASE_FILES=15
```

`Ghost-FTP-0.0.2-Setup-x32.exe` is intentionally a byte-identical alias of the verified x86 Setup file.

## Read-back verification

The release transaction compares the remote sorted asset set with the expected allow-list immediately and again after a delay. For 0.0.2 it requires `prerelease=false`.

Only after this verification succeeds may the retention workflow delete superseded public version identities.

## GitHub Packages

The same verified release directory is published as an OCI distribution bundle at:

```text
ghcr.io/bren-wp/ghost-ftp:0.0.2
```

It is a **distribution bundle**, not a runtime container. The release workflow publishes the exact-version tag together with current aliases and verifies the exact-version package after push. Retention preserves the package version carrying the current exact semantic-version tag and removes obsolete package versions only after release verification succeeds.

See [Packages](PACKAGES.md), [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md), [Testing](TESTING.md) and [Versioning](VERSIONING.md).
