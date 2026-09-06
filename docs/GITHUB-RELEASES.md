# Ghost FTP GitHub Releases

Ghost FTP **1.0.0 Stable** is the maintained production baseline. Official releases are created only by `.github/workflows/release.yml` from the exact verified `main` commit.

## Release identity

For version `1.0.0`:

```text
Tag: ghostftp-v1.0.0
Title: Ghost FTP 1.0.0
Draft: false
Prerelease: false
```

The release workflow reads `VERSION` directly and rejects a manual workflow version that differs from source.

## Stable-only publication

The maintained production workflow requires `MAJOR >= 1`. Pre-1.0 publication is rejected before release assembly, so the workflow no longer creates new GitHub prereleases.

Historical 0.x prereleases remain immutable release history. They are not deleted, rewritten or relabeled merely because the current product is stable.

## Required public files

A complete stable Release exposes **9 platform artifacts**.

Windows:

```text
Ghost-FTP-1.0.0-Setup-x64.exe
Ghost-FTP-1.0.0-Setup-x86.exe
Ghost-FTP-1.0.0-Setup-x32.exe
Ghost-FTP-1.0.0-Portable-x64.exe
Ghost-FTP-1.0.0-Portable-x86.exe
```

Linux:

```text
Ghost-FTP-1.0.0-Linux-amd64.deb
Ghost-FTP-1.0.0-Linux-arm64.deb
Ghost-FTP-1.0.0-Linux-i386.deb
Ghost-FTP-1.0.0-Linux-multiarch.zip
```

Verification/metadata:

```text
BUILD-METADATA.txt
RELEASE-NOTES.txt
SHA256.txt
```

That is exactly **12 public files**.

## Exact-head rule

Immediately before GitHub Release publication, the workflow queries current `main` and requires it to equal `GITHUB_SHA`. It verifies the same condition again during delayed read-back and again before GHCR publication. If `main` moves, the transaction fails rather than silently publishing stale source.

## Immutable tag rule

If `ghostftp-v1.0.0` already exists, it must resolve to the exact release commit. The workflow refuses to move an existing release tag to different source.

## Windows signing/trust state

Windows artifacts may be published in one of two explicit states:

```text
WINDOWS_AUTHENTICODE=signed
WINDOWS_AUTHENTICODE=unsigned
```

If protected production Authenticode credentials are configured, every Windows artifact must verify successfully before publication. If they are not configured, the release is explicitly unsigned and `BUILD-METADATA.txt` records:

```text
WINDOWS_TRUST_MODE=sha256+github-release-provenance
```

The workflow never uses a self-signed/development certificate as a fake production publisher identity. An unsigned release must be verified through the official Release/tag, `SHA256.txt` and source commit provenance.

## Artifact allow-list

The publish job creates a fresh `release/` directory from only the verified Windows/Linux staging artifacts plus generated notes, build metadata and checksums. The final count and expected filenames are verified before upload.

`Setup-x32.exe` is intentionally a byte-identical alias of `Setup-x86.exe`; their SHA-256 values must match.

## Release creation/update behavior

If the release does not yet exist, the workflow creates it without a prerelease flag. If it already exists at the exact immutable tag, the workflow explicitly patches:

```text
name=Ghost FTP <version>
draft=false
prerelease=false
```

and then replaces assets with the verified allow-list. This makes a rerun idempotent for the same source/tag while refusing tag rewrites.

## Read-back verification

After upload, the workflow:

1. compares the remote asset names with the exact 12-file allow-list;
2. downloads the remote `SHA256.txt` and compares it byte-for-byte with the local manifest;
3. requires `draft=false`;
4. requires `prerelease=false`;
5. requires the exact release title;
6. repeats the checks after a short delay and another exact-`main` verification.

A release workflow is not considered successful until this remote read-back passes.

## GitHub Packages

After the GitHub Release is verified, the same workflow publishes the verified release directory to:

```text
ghcr.io/bren-wp/ghost-ftp:1.0.0
```

The registry object is an OCI **distribution bundle**, not a runtime container. The build uses `FROM scratch`, `--network=none` and copies only `release/` into `/ghostftp-release/`.

Stable aliases are:

```text
1.0
1
latest
```

After push the workflow removes its local image, pulls the semantic-version package from GHCR again, verifies OCI source/version/revision labels and copies `SHA256.txt` plus `BUILD-METADATA.txt` out of the pulled package for byte-for-byte comparison.

See [Packages](PACKAGES.md).

## Release metadata

`BUILD-METADATA.txt` records:

- public and technical product identity;
- semantic version and immutable tag;
- source commit;
- `RELEASE_CHANNEL=stable`;
- `GITHUB_RELEASE_PRERELEASE=false`;
- active application platforms;
- Windows package architecture and actual Authenticode state;
- Windows trust mode;
- Linux package architecture set;
- language count/default language;
- telemetry-disabled state;
- public artifact/file counts;
- GitHub Package reference.

## Failure behavior

A failed quality gate, build, configured-signature verification, tag check, Release upload/read-back, GHCR push or GHCR read-back makes the workflow fail. The failure must be corrected and the exact candidate rerun; security state is never changed by pretending an unsigned artifact is signed.

See [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md), [Packages](PACKAGES.md) and [Versioning](VERSIONING.md).
