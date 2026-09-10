# Ghost FTP testing and quality gates

Ghost FTP **0.0.3** is validated through layered source, security, native build, packaging, UI-action and release-lifecycle gates.

## Core quality gate

The canonical Core gate requires:

```text
gofmt
go test -race ./...
go vet ./...
```

It also runs repository, platform, desktop-surface, dependency, version, localization, security, privacy, documentation and release audits plus the complete Python regression suite.

## Runtime regressions

Tests protect explicit FTPS verification/no downgrade, strict SFTP host-key verification, rooted local path and transfer confinement, transfer staging/activation/rollback, connection-generation guards, privacy-safe diagnostics, bounded retries, queue lifecycle and Remote Edit size/text/revision/conflict/permission/read-back behavior.

Current-folder filtering is non-destructive and operates only on already-loaded snapshots. Recursive search is a separate bounded I/O operation with depth/item/result/time limits, cancellation and fresh-list navigation. Directory comparison remains conservative and read-only; synchronized navigation requires freshly listed matching ordinary directories before either pane path is committed.

## Bandwidth regression contract

0.0.3 adds independent upload/download ceilings. Tests require:

- validated range `0..1,048,576 KiB/s` with `0 = unlimited`;
- migration-safe zero defaults for older settings;
- independent upload/download persistence;
- conservative aggregate budget allocation across configured parallel workers;
- no busy-wait/UI-only fake throttle;
- curl `limit-rate` enforcement for FTP/FTPS;
- OpenSSH `sftp -l` enforcement for SFTP with conservative KiB/s→Kbit/s conversion;
- each transfer attempt to snapshot its effective budget so saving settings cannot mutate an already-running transport process;
- Windows and Linux settings controls to map to the same shared model.

## Windows production gate

The Windows job builds and verifies the public universal artifacts:

```text
Ghost-FTP-0.0.3-Setup.exe
Ghost-FTP-0.0.3-Portable.exe
```

The build internally creates native x64/x86 payloads, verifies the universal bootstrap contract, rejects architecture-specific EXEs in the public output directory and runs the Authenticode private-key pipeline smoke test. Production signing is optional; configured signatures must verify.

## Linux production and distro package gate

The legacy generic `linux/BUILD.sh` remains a CI compatibility build, while canonical release packaging uses:

```text
linux/BUILD-DISTROS.sh
```

`.github/workflows/linux-distro-packages.yml` builds Debian/Ubuntu/Fedora/Portable artifacts for all maintained architectures and verifies package metadata and byte parity. These packages are canonical members of the **14 platform artifacts / 17 public files** 0.0.3 release allow-list.

## Native distro lifecycle gate

`.github/workflows/linux-distro-install.yml` verifies native install/remove/runtime/GUI behavior on:

- **Debian 13 amd64**;
- **Ubuntu 26.04 LTS amd64**;
- **Fedora 44 x86_64**.

**Native package-manager/runtime coverage is deliberately limited to x86-64.** Additional architectures retain build/parity coverage.

## Authentic UI evidence

`.github/workflows/ui-screenshots.yml` runs the real Windows production build and captures maintained Main Workspace, Site Manager, Settings and About surfaces from the verified internal native x64 payload. The workflow does not require an architecture-specific public download. Mockups and generated approximations are not release evidence.

## Exact-head and post-merge rule

**Exact-head and post-merge rule:** a PR is not merge-ready until every required workflow triggered for its exact final head is `completed/success`. After merge, required `push` workflows are identified by the exact merge SHA and must also finish `completed/success` before release preparation continues.

Expected release-prep gates are:

1. Ghost FTP CI;
2. Ghost FTP Linux Distro Packages;
3. Ghost FTP Linux Distro Install Matrix;
4. Authentic UI Screenshots when its path/trigger contract applies.

A green run for an older commit does not satisfy a newer PR head.

## Release publication gate

0.0.3 publication additionally requires exact current `main` release-branch validation, canonical release quality/build jobs, exact 17-file GitHub Release allow-list, immediate and delayed remote release read-back, `prerelease=false`, verified `ghcr.io/bren-wp/ghost-ftp:0.0.3` distribution-bundle read-back and successful latest-only retention cleanup.

## Retention validation

Before destructive cleanup, retention independently verifies current release identity, `draft=false`, `prerelease=false`, exactly **17 assets**, tag SHA equality with current `main` and presence of the exact-version package. `main` history must remain untouched.

## Quality rule for new capabilities

A feature is not release-ready until applicable shared runtime behavior, validation/defaults, Windows/Linux exposure, localization, failure/cancel semantics, tests, action wiring, documentation and authentic UI evidence all exist.

See [GitHub Releases](GITHUB-RELEASES.md), [Release verification](RELEASE-VERIFICATION.md), [Settings](SETTINGS.md), [Roadmap](ROADMAP.md) and [Versioning](VERSIONING.md).
