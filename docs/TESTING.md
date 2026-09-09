# Ghost FTP testing and quality gates

Ghost FTP **0.0.1 Beta** is validated through layered source, security, native build, packaging and lifecycle gates.

## Core quality gate

The canonical Core gate requires:

```text
gofmt
go test -race ./...
go vet ./...
```

It also runs repository, platform, desktop-surface, dependency, version, localization, security, privacy, documentation and release audits plus the Python regression suite.

## Protocol and transfer regressions

Tests cover the maintained FTP/FTPS/SFTP engine contract, including:

- explicit FTPS verification and no silent downgrade;
- strict SFTP host-key verification/pinning;
- rooted local path and transfer confinement;
- transfer staging/activation/rollback behavior;
- connection-generation guards;
- privacy-safe diagnostics;
- Remote Edit text/binary, size, revision/conflict, permission, read-back and metadata-refresh behavior.

## Windows production gate

The Windows production job builds and verifies:

```text
Ghost-FTP-0.0.1-Setup-x64.exe
Ghost-FTP-0.0.1-Setup-x86.exe
Ghost-FTP-0.0.1-Setup-x32.exe
Ghost-FTP-0.0.1-Portable-x64.exe
Ghost-FTP-0.0.1-Portable-x86.exe
```

It verifies release artifacts and exercises the Authenticode private-key pipeline policy. Production signing is optional; configured signatures must verify.

## Linux production gate

The Linux production job builds DEB and portable tar.gz packages for `amd64`, `arm64` and `i386` and compares the DEB/portable executable bytes for parity.

## Supplemental distro package gate

`.github/workflows/linux-distro-packages.yml` builds and verifies supplemental distro artifacts through:

```text
linux/BUILD-DISTROS.sh
```

Representative supplemental names include Debian, Ubuntu, Fedora and Portable families. These are CI verification artifacts, not additions to the canonical **12 platform artifacts / 15 public files** release allow-list.

## Native distro lifecycle gate

`.github/workflows/linux-distro-install.yml` verifies native install/remove/runtime/GUI behavior on:

- **Debian 13 amd64**;
- **Ubuntu 26.04 LTS amd64**;
- **Fedora 44 x86_64**.

**Native package-manager/runtime coverage is deliberately limited to x86-64.** Canonical production builds still include the documented additional Linux architectures.

## Authentic UI evidence

`.github/workflows/ui-screenshots.yml` builds the real Windows x64 Portable application and captures maintained Main Workspace, Site Manager, Settings and About windows. Mockups and generated approximations are not release evidence.

A release-prep change affecting `VERSION` or maintained desktop UI must obtain authentic evidence from the exact final source revision where the screenshot workflow is triggered.

## Exact-head and post-merge rule

**Exact-head and post-merge rule:** a PR is not merge-ready until every required workflow triggered for its exact final head is `completed/success`. After merge, required `push` workflows are identified by the exact merge SHA and must also finish `completed/success` before release preparation continues.

For a release-prep change that affects the canonical production build, the expected gates are:

1. Ghost FTP CI;
2. Ghost FTP Linux Distro Packages;
3. Ghost FTP Linux Distro Install Matrix;
4. Authentic UI Screenshots when its path/trigger contract applies.

## Release publication gate

0.0.1 publication additionally requires:

- exact current `main` release-branch validation;
- canonical release workflow quality/build jobs;
- exact 15-file GitHub Release allow-list;
- immediate and delayed remote release read-back;
- `prerelease=true` for the 0.x Beta channel;
- successful latest-only retention cleanup after publication.

## Retention validation

The retention workflow must leave only the current `ghostftp-v0.0.1` release/tag, remove completed versioned release branches, and remove obsolete package versions. It does not rewrite `main` commit history.

See [GitHub Releases](GITHUB-RELEASES.md), [Release verification](RELEASE-VERIFICATION.md) and [Versioning](VERSIONING.md).
