# Ghost FTP versioning

Ghost FTP uses semantic versioning with the root `VERSION` file as the authoritative production version source.

Current source candidate: **0.0.3**.

## Version format

```text
MAJOR.MINOR.PATCH
```

Production tags use `ghostftp-vMAJOR.MINOR.PATCH`.

The current release identity is:

```text
VERSION=0.0.3
TAG=ghostftp-v0.0.3
CHANNEL=Current
PRERELEASE=false
```

## Public numbering line

The current public numbering started at **0.0.1**. `0.0.0` is reserved and must never be published. The intended sequence is `0.0.1`, `0.0.2`, `0.0.3`, ... . A release identity is never rewritten in place. For this project, major version `0` does not imply prerelease; the 0.0.x line uses `prerelease=false` unless policy is explicitly changed across workflow, audits and documentation.

## Latest-only public release retention

Ghost FTP intentionally keeps only the latest public version visible in release infrastructure. After a newly published release passes immediate and delayed remote read-back verification, `.github/workflows/release-retention.yml` removes superseded GitHub Releases, `ghostftp-v*` tags, superseded `release/ghostftp-v*` branches and obsolete package versions. It never rewrites Git commit history on `main`.

## Release trigger

A `VERSION` edit or ordinary push to `main` does not publish a release. The canonical release branch namespace is:

```text
release/ghostftp-v<version>
```

For 0.0.3:

```text
release/ghostftp-v0.0.3
```

`.github/workflows/release-branch-trigger.yml` accepts the branch only when its semantic version equals root `VERSION`, the branch points to exact current `main`, and the canonical release workflow is dispatched with the same version guard. It waits for publication success before retention is dispatched and verified.

## Binary and package identity

The semantic version is injected into Windows application binaries, universal Setup/Portable filenames, Linux DEB/RPM metadata, Portable archive names, release notes, build metadata and GitHub Release identity. Source entry points retain a `dev` fallback and receive production version through linker flags.

## Current GitHub Release rule

Publication uses:

```text
CHANNEL=Current
PRERELEASE=false
```

The canonical 0.0.3 release contains **14 platform artifacts / 17 public files**: two universal Windows executables, twelve Linux distro/Portable artifacts and three metadata/verification files.

The exact verified release directory is also published as a distribution bundle at:

```text
ghcr.io/bren-wp/ghost-ftp:0.0.3
```

The GHCR bundle is not a supported runtime container.

## Windows signing state

Windows Authenticode is an **optional production hardening layer**. When a trusted production identity is configured through protected Actions secrets, signatures must verify or publication fails. Without a production certificate, publication remains truthful through `WINDOWS_AUTHENTICODE=unsigned`. **Absence of a production Authenticode certificate by itself is not a versioning failure.** Ghost FTP never creates a self-signed production certificate and represents it as a trusted publisher.

## Version source integrity

Release and CI validation reject malformed semantic versions, `0.0.0`, release-branch/source-version mismatch, a release branch that does not equal exact current `main`, existing conflicting tags/releases, incomplete assets, failed configured Windows signatures, source/main drift during publication, incorrect GitHub prerelease state, a missing exact-version GHCR bundle, failed release read-back or failed latest-only retention cleanup.

## Changelog and release notes

`CHANGELOG.md` contains a `## <VERSION>` section. `scripts/release_notes.py` extracts that section for public release notes.

## 0.0.3 release checklist

The exact candidate must pass:

- Go formatting, `go test -race ./...` and `go vet ./...`;
- repository/platform/desktop/dependency/version/localization/security/privacy/documentation/release audits;
- complete Python regressions;
- FTP/FTPS/SFTP trust/no-downgrade and transfer safety tests;
- bandwidth settings/scheduler/transport tests;
- Windows universal Setup/Portable production build and Authenticode smoke;
- Debian/Ubuntu/Fedora/Portable package build, metadata and binary parity;
- Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64 install/remove/GUI smoke;
- 24-language localization and authentic Windows UI evidence;
- exact-head PR gates and exact post-merge `main` gates;
- exact-main `release/ghostftp-v0.0.3` validation;
- GitHub Release `ghostftp-v0.0.3` with `prerelease=false` and exact 17-file read-back;
- GHCR `ghcr.io/bren-wp/ghost-ftp:0.0.3` publication/read-back;
- successful latest-only retention cleanup after publication.

## Next release

Only after 0.0.3 publication and retention are completely green should root `VERSION` advance again through a separate reviewed release-prep change.
