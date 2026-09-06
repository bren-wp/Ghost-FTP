# Ghost FTP GitHub Releases

Ghost FTP **1.0.0 Stable** is the first normal stable public release in the maintained version line. Official releases are created only by `.github/workflows/release.yml` from the exact verified `main` commit.

## Release identity

For version `1.0.0`:

```text
Tag: ghostftp-v1.0.0
Title: Ghost FTP 1.0.0
Prerelease: false
```

The release workflow reads `VERSION` directly and rejects a manual workflow version that differs from the source version.

## Stable publication rule

A version with major number `1` or greater is treated as Stable. The release workflow does not pass GitHub's prerelease flag for stable versions.

Historical 0.x releases were Beta/prerelease builds and remain part of release history; they are not rewritten or relabeled as stable.

## Required public files

The stable Release exposes **9 platform artifacts**:

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

and three verification/metadata files:

```text
BUILD-METADATA.txt
RELEASE-NOTES.txt
SHA256.txt
```

That is **12 public files** in total.

## Exact-head rule

Before publication, the workflow queries the current `main` SHA and requires it to equal `GITHUB_SHA`. It verifies the condition again after release publication. If `main` moves during the transaction, publication fails instead of silently attaching files to stale source.

## Immutable tag rule

If `ghostftp-v1.0.0` already exists, it must resolve to the exact release commit. The workflow refuses to rewrite an existing version tag to different source.

## Stable Windows signing rule

A stable Release is blocked unless the protected release environment reports the Windows artifacts as signed using the configured trusted Authenticode identity. Private signing material is supplied only via protected Actions secrets, written temporarily on the runner and removed after use.

## Artifact allow-list

The publish job assembles a fresh `release/` directory from only the verified Windows and Linux staging artifacts plus generated notes/metadata/checksums. The final file count and expected filenames are checked before upload.

`Setup-x32.exe` is intentionally a byte-identical alias of `Setup-x86.exe`; the workflow verifies their SHA-256 values match.

## Read-back verification

After creating/updating the Release, the workflow reads the remote asset set from GitHub and compares it with the expected sorted list. It also reads the `prerelease` property and requires it to be `false` for the stable channel. A delayed second read-back runs after a short wait to catch asynchronous publication issues.

## GitHub Packages

Stable publication additionally pushes the verified release directory to:

```text
ghcr.io/bren-wp/ghost-ftp:1.0.0
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

## Failure behavior

A failed quality gate, production build, signing check, package push, tag validation, Release upload or read-back check causes the workflow to fail. A failed workflow is not represented as a completed stable release contract.

See [Release verification](RELEASE-VERIFICATION.md), [Signing](SIGNING.md) and [Versioning](VERSIONING.md).
