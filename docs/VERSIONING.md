# Ghost FTP versioning

Ghost FTP uses semantic versioning with the root `VERSION` file as the authoritative production version source.

Current source candidate: **0.0.5**.

## Version format

```text
MAJOR.MINOR.PATCH
```

Production tags use `ghostftp-vMAJOR.MINOR.PATCH`.

The current release identity is:

```text
VERSION=0.0.5
TAG=ghostftp-v0.0.5
CHANNEL=Current
PRERELEASE=false
```

## Public numbering line

The current public numbering started at **0.0.1**. `0.0.0` is reserved and must never be published.

```text
0.0.1
0.0.2
0.0.3
0.0.4
0.0.5
...
```

Each new release must be based on a fully verified current `main` revision. A release identity is never rewritten in place. For this project, **major version `0` does not imply prerelease**: the 0.0.x line is the current public release line and uses `prerelease=false` unless a future explicit policy change says otherwise.

## Latest-only public release retention

Ghost FTP intentionally keeps only the **latest public version** visible in release infrastructure.

After a newly published release passes immediate/delayed remote read-back verification, `.github/workflows/release-retention.yml` removes superseded Ghost FTP GitHub Releases, `ghostftp-v*` tags, superseded canonical release branches and obsolete container package versions. It retains current identities and never rewrites Git commit history on `main`.

## Release trigger

A `VERSION` edit or ordinary push to `main` does not publish a release. The canonical release branch namespace is:

```text
release/ghostftp-v<version>
```

For 0.0.5:

```text
release/ghostftp-v0.0.5
```

The release-branch trigger accepts that branch only when its semantic version equals root `VERSION` and the branch points to exact current `main`. It then dispatches canonical `release.yml`, waits for the exact publish run to succeed, dispatches retention and waits for exact retention success.

## Current GitHub Release rule

Publication uses:

```text
CHANNEL=Current
PRERELEASE=false
```

The canonical 0.0.5 release contains **14 platform artifacts / 17 public files**: two universal Windows executables, twelve Linux Debian/Ubuntu/Fedora/Portable artifacts and three metadata/verification files.

The verified release directory is also published as:

```text
ghcr.io/bren-wp/ghost-ftp:0.0.5
```

The GHCR object is a distribution bundle, not a supported runtime container.

## Windows packaging identity

```text
Ghost-FTP-0.0.5-Setup.exe
Ghost-FTP-0.0.5-Portable.exe
```

Verified native x64/x86 application payloads remain internal staging artifacts embedded in the universal build. Architecture-specific Windows staging executables must not leak into the public directory.

## Linux packaging identity

Canonical Linux packaging uses `linux/BUILD-DISTROS.sh` and publishes Debian DEBs for `amd64`, `arm64`, `i386`; Ubuntu DEBs for the same architectures; Fedora RPMs for `x86_64`, `aarch64`, `i686`; and Portable tarballs for `amd64`, `arm64`, `i386`.

## Android source identity

Android source reads root `VERSION`, but the maintained artifact is an installable development APK with a `-dev` version identity. It is independently verified and is not silently included in the public Windows/Linux 17-file release allow-list.

## Windows signing state

Windows Authenticode is a **required public-release trust boundary**. Public Setup and Portable executables must be signed with the protected trusted production identity, and the signatures must verify successfully before publication can continue.

The canonical public release state is:

```text
WINDOWS_AUTHENTICODE=signed
```

**Absence of the production Authenticode identity is a release failure.** Ghost FTP never creates a self-signed production certificate or publishes unsigned Windows binaries as the current public release. Ordinary CI, development and local builds may remain unsigned because they are not public release artifacts.

## Version source integrity

Release/CI validation rejects malformed semantic versions, `0.0.0`, release-branch/source-version mismatch, non-exact-main release branches, conflicting current tags/releases, incomplete release assets, architecture-specific public Windows leakage, absent or failed public Windows signatures, source/main drift, incorrect prerelease flags, missing GHCR exact-version bundle, failed release read-back or failed latest-only retention cleanup.

Active release-bound documentation must agree with root `VERSION`, `CHANNEL=Current`, `PRERELEASE=false`, the 14/17 packaging contract and the current package identity.

## Changelog and release notes

`CHANGELOG.md` contains the maintained public release line and includes a `## <VERSION>` section. `scripts/release_notes.py` extracts that section and must describe the same public package names/counts as canonical `release.yml`.

## 0.0.5 release checklist

The exact candidate must pass:

- Go formatting, `go test -race ./...` and `go vet ./...`;
- repository/platform/desktop/dependency/version/localization/security/privacy/documentation/release audits;
- the complete Python regression suite;
- FTP/FTPS/SFTP trust/no-downgrade and local-path safeguards;
- Remote Edit size/text/revision/conflict/permission/read-back/metadata and session-lifecycle safeguards;
- profile persistence and local/remote mutation re-entry contracts;
- filtering, sorting, recursive search, directory comparison and synchronized-navigation contracts;
- queue Top/Up/Down/Bottom ordering and connection binding;
- navigation bookmark/profile-start account/session validation;
- upload/download bandwidth settings, aggregate scheduling and transport enforcement;
- Windows installer/uninstaller/shortcut ownership checks;
- Linux trusted transport/AskPass provenance checks;
- Android source contract, lifecycle connection ownership, strict FTPS/parser bounds, lint, installable development APK and APK verification;
- universal Windows Setup and Portable production builds with verified native payloads;
- trusted Authenticode signing and verification for both public Windows executables;
- Linux Debian/Ubuntu/Fedora/Portable build, metadata, extraction and binary parity;
- Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64 install/remove/GUI smoke;
- 24-language localization and authentic Windows/Linux/Android UI evidence;
- exact-head PR gates and exact post-merge `main` gates;
- exact-main `release/ghostftp-v0.0.5` validation;
- GitHub Release `ghostftp-v0.0.5` with `prerelease=false` and exact 17-file read-back;
- GHCR `ghcr.io/bren-wp/ghost-ftp:0.0.5` publication/read-back;
- successful latest-only retention cleanup.

## Next release

Only after 0.0.5 publication and retention are completely green should root `VERSION` advance again through a separate reviewed release-prep change.
