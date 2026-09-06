# Ghost FTP GitHub Releases

Ghost FTP **1.1.1 Stable** is the current maintained stable candidate line. Ghost FTP **1.1.0 Stable** remains the previously published feature release and **1.0.0 Stable** remains the first normal stable public release; published historical tags/releases must not be rewritten. Official releases are created only by `.github/workflows/release.yml` from the exact verified `main` commit.

## Release identity

For version `1.1.1`:

```text
Tag: ghostftp-v1.1.1
Title: Ghost FTP 1.1.1
Prerelease: false
```

The release workflow reads `VERSION` directly and rejects a manual workflow version that differs from the source version.

## Stable publication rule

A version with major number `1` or greater is treated as Stable. The release workflow does not pass GitHub's prerelease flag for stable versions.

Historical 0.x releases were Beta/prerelease builds and remain part of release history; they are not rewritten or relabeled as stable. Existing `ghostftp-v1.0.0` and `ghostftp-v1.1.0` tags remain bound to their original published release commits.

## Required public files

The stable Release exposes **9 platform artifacts**.

Windows:

```text
Ghost-FTP-1.1.1-Setup-x64.exe
Ghost-FTP-1.1.1-Setup-x86.exe
Ghost-FTP-1.1.1-Setup-x32.exe
Ghost-FTP-1.1.1-Portable-x64.exe
Ghost-FTP-1.1.1-Portable-x86.exe
```

Linux:

```text
Ghost-FTP-1.1.1-Linux-amd64.deb
Ghost-FTP-1.1.1-Linux-arm64.deb
Ghost-FTP-1.1.1-Linux-i386.deb
Ghost-FTP-1.1.1-Linux-multiarch.zip
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

The canonical `release/ghostftp-v1.1.1` trigger branch must therefore be created from the exact `main` commit that passed the complete post-merge quality gate.

## Immutable tag rule

If `ghostftp-v1.1.1` already exists, it must resolve to the exact intended release commit. The workflow refuses to rewrite an existing version tag to different source.

The already-published `ghostftp-v1.0.0` and `ghostftp-v1.1.0` tags are separate immutable history and are never moved or reused for 1.1.1.

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

After creating/updating the Release, the workflow reads the remote asset set from GitHub and compares it with the expected sorted list. It also reads the `prerelease` property and requires it to be `false` for the stable channel. A delayed second read-back runs after a short wait to catch asynchronous publication issues.

A release is not considered published merely because a local build succeeded; remote Release/tag state must agree with the verified source revision and file contract.

## GitHub Packages

Stable 1.1.1 publication additionally pushes the verified release directory to:

```text
ghcr.io/bren-wp/ghost-ftp:1.1.1
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

## UI/documentation evidence

The 1.1.1 release documentation must describe the actual maintained behavior: Classic Light is the fresh/fallback primary appearance, Dark remains an explicit Windows choice, and FTPS is the fresh quick-connect protocol while plain FTP remains an explicit compatibility option. Repository UI screenshots are produced from the real production Windows x64 Portable executable by the dedicated screenshot workflow; mockups or generated approximations are not accepted as release evidence.

## Failure behavior

A failed quality gate, production build, configured-signing verification, package push, tag validation, Release upload or read-back check causes the workflow to fail. Absence of a production Authenticode certificate alone does not fail publication; that state is preserved as `unsigned` metadata.

See [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md) and [Versioning](VERSIONING.md).
