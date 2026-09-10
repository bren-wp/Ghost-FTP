# Ghost FTP versioning

Ghost FTP uses semantic versioning with the root `VERSION` file as the authoritative production version source.

Current source candidate: **0.0.2**.

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
VERSION=0.0.2
TAG=ghostftp-v0.0.2
CHANNEL=Current
PRERELEASE=false
```

## Public numbering line

The current public numbering started at **0.0.1**. `0.0.0` is reserved and must never be published.

The intended sequence is incremental:

```text
0.0.1
0.0.2
0.0.3
...
```

Each new release must be based on a fully verified current `main` revision. A release identity is never rewritten in place. For this project, major version `0` does not imply prerelease: the 0.0.x line is the current public release line and uses `prerelease=false` unless a future explicit policy change says otherwise.

## Latest-only public release retention

Ghost FTP intentionally keeps only the latest public version visible in release infrastructure.

After a newly published release passes immediate and delayed remote read-back verification, `.github/workflows/release-retention.yml` removes superseded Ghost FTP GitHub Releases, `ghostftp-v*` tags, superseded `release/ghostftp-v*` branches and obsolete container package versions. The retention workflow keeps the latest public version, current version tag, current canonical release branch and GHCR package carrying the exact current version tag. It never rewrites Git commit history on `main`.

This policy means old release URLs and tags are not a supported archival interface. Git history remains engineering provenance.

## Release trigger

A `VERSION` edit or ordinary push to `main` does not publish a release.

The canonical release branch namespace is:

```text
release/ghostftp-v<version>
```

For 0.0.2:

```text
release/ghostftp-v0.0.2
```

`.github/workflows/release-branch-trigger.yml` accepts the branch only when its semantic version equals root `VERSION`, the branch points to exact current `main`, and the canonical release workflow is dispatched with the same version guard. The trigger waits for the exact publish run to succeed before it can explicitly dispatch and verify retention.

## Binary and package identity

The semantic version is injected into Windows application binaries, Setup/Portable filenames, Linux DEB metadata, portable archive names, release notes, build metadata and GitHub Release identity.

Source entry points retain a development fallback and receive the production version through build linker flags. The user-facing version displays the canonical semantic version without automatically adding a `Beta` suffix for major version zero.

## Current GitHub Release rule

Publication uses:

```text
CHANNEL=Current
PRERELEASE=false
```

The canonical 0.0.2 release contains **12 platform artifacts / 15 public files**.

The exact verified release directory is also published as a distribution bundle at:

```text
ghcr.io/bren-wp/ghost-ftp:0.0.2
```

The GHCR bundle is not a supported runtime container. Publication also maintains current aliases derived from the semantic version and `latest`; the exact version tag is the immutable verification identity for the current release transaction.

## Windows signing state

Windows Authenticode is an **optional production hardening layer**. When a trusted production identity is configured through protected Actions secrets, signatures must verify or publication fails.

Without a production certificate, publication remains truthful through:

```text
WINDOWS_AUTHENTICODE=unsigned
```

Absence of a production Authenticode certificate by itself is not a versioning failure. Ghost FTP never creates a self-signed production certificate and represents it as a trusted publisher.

## Version source integrity

Release and CI validation must reject malformed semantic versions, `0.0.0`, release-branch/source-version mismatch, a release branch that does not equal exact current `main`, an existing conflicting current tag/release, incomplete release assets, failed configured Windows signatures, source/main drift during publication, an incorrect GitHub prerelease flag, a missing exact-version GHCR distribution bundle, failed release read-back or failed latest-only retention cleanup.

The active documentation must agree with root `VERSION`, `CHANNEL=Current`, `PRERELEASE=false` and the current package identity.

## Changelog and release notes

`CHANGELOG.md` contains the maintained public release line and includes a `## <VERSION>` section. `scripts/release_notes.py` extracts that section for public release notes.

## 0.0.2 release checklist

The exact candidate must pass:

- Go formatting, `go test -race ./...` and `go vet ./...`;
- repository/platform/desktop/dependency/version/localization/security/privacy/documentation/release audits;
- the complete Python regression suite;
- real FTP/FTPS/SFTP behavior regressions and strict trust/no-downgrade checks;
- rooted local transfer/filesystem safeguards;
- Remote Edit size/text/revision/conflict/permission/read-back/metadata-refresh safeguards;
- current-folder filter and bounded recursive local/server search regression contracts;
- conservative directory-comparison and synchronized-navigation regression contracts;
- Windows installer/uninstaller/shortcut ownership and exact-object cleanup checks;
- Linux trusted transport/AskPass provenance checks;
- Windows x64/x86 Setup and Portable production builds;
- Linux amd64/arm64/i386 DEB and portable production builds;
- supplemental distro package/parity checks;
- Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64 install/remove/GUI smoke;
- 24-language localization and authentic Windows UI evidence;
- exact-head PR gates and exact post-merge `main` gates;
- exact-main `release/ghostftp-v0.0.2` validation;
- GitHub Release `ghostftp-v0.0.2` with `prerelease=false` and exact 15-file read-back;
- GHCR `ghcr.io/bren-wp/ghost-ftp:0.0.2` publication/read-back;
- successful latest-only retention cleanup after publication.

## Next release

Only after 0.0.2 publication and retention are completely green should root `VERSION` advance to **0.0.3** through a separate reviewed release-prep change.
