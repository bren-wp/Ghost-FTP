# Ghost FTP versioning

Ghost FTP uses semantic versioning with the root `VERSION` file as the authoritative production version source.

Current source candidate: **0.0.1 Beta**.

## Version format

```text
MAJOR.MINOR.PATCH
```

Production tags use:

```text
ghostftp-vMAJOR.MINOR.PATCH
```

The current release identity is:

```text
VERSION=0.0.1
TAG=ghostftp-v0.0.1
CHANNEL=Beta
PRERELEASE=true
```

## New public numbering line

The current public numbering starts at **0.0.1**.

`0.0.0` is reserved and must never be published.

The intended sequence is incremental:

```text
0.0.1
0.0.2
0.0.3
...
```

Each new release must be based on a fully verified current `main` revision. A future version is not created until the current version has been published, verified and retention cleanup has completed.

All versions with `MAJOR=0` are **Beta** prereleases and use `prerelease=true`. A future Stable policy may begin only through an explicit versioning change and corresponding documentation/audit update.

## Latest-only public release retention

Ghost FTP intentionally keeps only the latest public version visible in release infrastructure.

After a newly published release passes immediate and delayed remote read-back verification, `.github/workflows/release-retention.yml` removes superseded Ghost FTP:

- GitHub Releases;
- `ghostftp-v*` tags;
- completed `release/ghostftp-v*` branches;
- obsolete container package versions.

The retention workflow must never run before a successful canonical release transaction. Git commit history on `main` is not rewritten.

This policy means old release URLs and tags are not a supported archival interface. Users and downstream automation must resolve the current release rather than pinning a superseded public version.

## Release trigger

A `VERSION` edit or ordinary push to `main` does not publish a release.

The canonical release branch namespace is:

```text
release/ghostftp-v<version>
```

For 0.0.1:

```text
release/ghostftp-v0.0.1
```

`.github/workflows/release-branch-trigger.yml` accepts the branch only when:

1. its semantic version equals root `VERSION`;
2. the branch points to exact current `main`;
3. the release workflow is dispatched with the same version guard.

## Binary and package identity

The semantic version is injected into Windows application binaries, Setup/Portable filenames, Linux DEB metadata, portable archive names, release notes, build metadata and GitHub Release identity.

Source entry points retain a development fallback and receive the production version through build linker flags.

## Beta GitHub Release rule

For `MAJOR=0`, publication uses:

```text
CHANNEL=Beta
PRERELEASE=true
```

The canonical 0.0.1 release contains **12 platform artifacts / 15 public files**.

Pre-1.0 Beta releases do not publish the Stable GHCR distribution bundle. If an obsolete package version exists from an older numbering line, retention removes it after 0.0.1 is verified.

## Windows signing state

Windows Authenticode is an **optional production hardening layer**. When a trusted production identity is configured through protected Actions secrets, signatures must verify or publication fails.

Without a production certificate, publication remains truthful through:

```text
WINDOWS_AUTHENTICODE=unsigned
```

Absence of a production Authenticode certificate by itself is not a versioning failure. Ghost FTP never creates a self-signed production certificate and represents it as a trusted publisher.

## Version source integrity

Release and CI validation must reject:

- malformed semantic versions;
- `0.0.0`;
- a release-branch version different from root `VERSION`;
- a release branch that does not equal exact current `main`;
- an existing conflicting current tag/release;
- incomplete release assets;
- failed configured Windows signatures;
- source/main drift during publication;
- failed release read-back;
- failed latest-only retention cleanup.

The active documentation must agree with root `VERSION` and the current Beta/Stable channel derived from it.

## Changelog and release notes

`CHANGELOG.md` contains only the current maintained public release line and must include a `## <VERSION>` section. `scripts/release_notes.py` extracts that section for public release notes.

## 0.0.1 release checklist

The exact candidate must pass:

- Go formatting, `go test -race ./...` and `go vet ./...`;
- repository/platform/desktop/dependency/version/localization/security/privacy/documentation/release audits;
- the complete Python regression suite;
- real FTP/FTPS/SFTP behavior regressions and strict trust/no-downgrade checks;
- rooted local transfer/filesystem safeguards;
- Remote Edit size/text/revision/conflict/permission/read-back/metadata-refresh safeguards;
- Windows installer/uninstaller/shortcut ownership and exact-object cleanup checks;
- Linux trusted transport/AskPass provenance checks;
- Windows x64/x86 Setup and Portable production builds;
- Linux amd64/arm64/i386 DEB and portable production builds;
- supplemental distro package/parity checks;
- Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64 install/remove/GUI smoke;
- 24-language localization and authentic Windows UI evidence;
- exact-head PR gates and exact post-merge `main` gates;
- exact-main `release/ghostftp-v0.0.1` validation;
- Beta GitHub Release `ghostftp-v0.0.1` with `prerelease=true` and exact 15-file read-back;
- successful latest-only retention cleanup after publication.

## Next release

Only after 0.0.1 publication and retention are completely green should root `VERSION` advance to **0.0.2** through a separate reviewed release-prep change.
