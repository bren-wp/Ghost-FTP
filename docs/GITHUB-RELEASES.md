# Ghost FTP GitHub Releases

Ghost FTP **1.1.4 Stable** is the current maintained stable release candidate. Ghost FTP **1.1.3 Stable** is the previously published maintenance release; earlier published Stable tags/releases remain immutable historical identities. Official releases are created only by the canonical release workflow from the exact verified `main` commit.

## Release identity

For version `1.1.4`:

```text
Tag: ghostftp-v1.1.4
Title: Ghost FTP 1.1.4
Prerelease: false
```

The release workflow reads `VERSION` directly and rejects a manual workflow version that differs from the source version.

## Stable publication rule

A version with major number `1` or greater is treated as Stable. The release workflow does not pass GitHub's prerelease flag for stable versions.

Historical 0.x releases were Beta/prerelease builds and remain part of release history; they are not rewritten or relabeled as stable. Existing `ghostftp-v1.0.0`, `ghostftp-v1.1.0`, `ghostftp-v1.1.1`, `ghostftp-v1.1.2` and `ghostftp-v1.1.3` tags remain bound to their original published release commits.

## Canonical release trigger

`release.yml` is intentionally `workflow_dispatch`-only. A push to `main`, including a commit that changes `VERSION`, must not publish a release directly.

The generic canonical release-branch namespace is `release/ghostftp-vX.Y.Z`. For version 1.1.4, the release-prep PR first passes exact-head CI. After merge, the exact current `main` SHA must pass post-merge Core, Windows and Linux CI. Only then is `release/ghostftp-v1.1.4` created at that exact `main` SHA. `.github/workflows/release-branch-trigger.yml` verifies both branch-to-main SHA equality and branch-version-to-`VERSION` equality before dispatching `release.yml` on `main` with the expected version guard.

This keeps publication behind one canonical branch trigger and prevents duplicate or premature releases caused by a `VERSION` push.

## Required public files

The stable Release exposes **9 platform artifacts**.

Windows:

```text
Ghost-FTP-1.1.4-Setup-x64.exe
Ghost-FTP-1.1.4-Setup-x86.exe
Ghost-FTP-1.1.4-Setup-x32.exe
Ghost-FTP-1.1.4-Portable-x64.exe
Ghost-FTP-1.1.4-Portable-x86.exe
```

Linux:

```text
Ghost-FTP-1.1.4-Linux-amd64.deb
Ghost-FTP-1.1.4-Linux-arm64.deb
Ghost-FTP-1.1.4-Linux-i386.deb
Ghost-FTP-1.1.4-Linux-multiarch.zip
```

and three verification/metadata files:

```text
BUILD-METADATA.txt
RELEASE-NOTES.txt
SHA256.txt
```

That is **12 public files** in total.

## Exact-head rule

Before publication, the workflow queries the current `main` SHA and requires it to equal `GITHUB_SHA`. It verifies the condition again after release publication. If `main` moves during the transaction, publication fails instead of silently attaching files to stale source.

The canonical `release/ghostftp-v1.1.4` trigger branch must therefore be created from the exact `main` commit that passed the complete post-merge quality gate. The release-branch trigger independently verifies that equality before it dispatches publication.

## Immutable tag rule

If the new `ghostftp-v1.1.4` tag already exists unexpectedly before publication, the release must stop. The tag must not be moved, deleted, reused or force-pushed.

The already-published `ghostftp-v1.0.0`, `ghostftp-v1.1.0`, `ghostftp-v1.1.1`, `ghostftp-v1.1.2` and `ghostftp-v1.1.3` tags are immutable release history. After successful 1.1.4 publication, `ghostftp-v1.1.4` becomes immutable under the same rule.

## Windows signing state

Authenticode signing is optional for stable publication. If protected production signing secrets are configured, the Windows Setup/Portable artifacts are signed and each produced signature must verify successfully. If no production certificate is configured, Windows artifacts are published unsigned and `BUILD-METADATA.txt` explicitly records:

```text
WINDOWS_AUTHENTICODE=unsigned
```

The workflow never generates a self-signed production publisher identity and never labels an unsigned artifact as signed. Private signing material, when used, is supplied only via protected Actions secrets, written temporarily on the runner and removed after use.

## Artifact allow-list

The publish job assembles a fresh `release/` directory from only the verified Windows and Linux staging artifacts plus generated notes/metadata/checksums. The final file count and expected filenames are checked before upload.

`Setup-x32.exe` is intentionally a byte-identical compatibility alias of `Setup-x86.exe`; the workflow verifies their SHA-256 values match. It is not a separate architecture build.

## Read-back verification

After creating/updating the Release, the workflow reads the remote asset set from GitHub and compares it with the expected sorted list. It also reads the `prerelease` property and requires it to be `false` for the stable channel. A delayed second read-back catches asynchronous publication issues.

A release is not considered published merely because a local build succeeded; remote Release/tag state must agree with the verified source revision and file contract.

## GitHub Packages

Stable 1.1.4 publication additionally pushes the verified release directory to:

```text
ghcr.io/bren-wp/ghost-ftp:1.1.4
```

Compatible stable aliases are published only after successful registry publication/read-back:

```text
1.1
1
latest
```

The registry package is an OCI distribution bundle, not a runtime container. The package build uses `FROM scratch`, copies only `release/`, disables Docker build networking, adds source/version/revision labels and verifies registry read-back.

See [Packages](PACKAGES.md).

## Release metadata

`BUILD-METADATA.txt` records:

- product and technical identity;
- semantic version and tag;
- source commit;
- stable/Beta channel;
- active application platforms;
- Windows architecture/signing state;
- Linux architecture set;
- language count/default language;
- telemetry-disabled state;
- public artifact/file counts;
- GitHub Package reference.

## 1.1.4 UI/documentation evidence

The 1.1.4 release documentation describes the maintained native behavior: Classic Light remains the fresh/fallback primary appearance, Dark remains an explicit saved Windows choice, FTPS remains the fresh quick-connect protocol, and Windows navigation remains centralized in the canonical left sidebar.

The 1.1.4 changes improve interaction and UI plumbing rather than introducing a new layout: language dropdown closure, asynchronous language persistence, serialized settings writes, canonical Settings limits/defaults, removal of redundant relayout work, idle transfer fast-path behavior and deduplicated local Segoe Fluent/MDL2 icon registration. No remote icon/font dependency was added.

Repository UI screenshots must be produced from the real production Windows x64 Portable executable whenever a release materially changes visual layout/rendering. Mockups or generated approximations are not accepted as release evidence.

## Failure behavior

A failed quality gate, production build, configured-signing verification, package push, tag validation, Release upload or read-back check causes the workflow to fail. Absence of a production Authenticode certificate alone does not fail publication; that state is preserved as `unsigned` metadata.

See [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md) and [Versioning](VERSIONING.md).
