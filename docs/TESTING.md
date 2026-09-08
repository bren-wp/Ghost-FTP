# Ghost FTP testing and quality gates

Ghost FTP **1.1.6 Stable** is the current published release. A later release is release-ready only when source tests, audits, native production builds, signing-state checks, packaging verification, authentic UI evidence and distribution read-back all pass for the exact release revision.

Published Stable releases through 1.1.6 are immutable historical identities. New source/CI gates may strengthen the next-release contract without rewriting an already published tag, asset set or checksum manifest.

## Continuous integration

The maintained CI contract has five distinct proof layers:

1. core quality, security and documentation;
2. Windows x64/x86 production build;
3. canonical Linux amd64/arm64/i386 production build;
4. supplemental Debian/Ubuntu/Fedora/Portable distro-package build and parity verification;
5. native x86-64 package install/remove and installed-GUI smoke verification on Debian 13, Ubuntu 26.04 LTS and Fedora 44.

The first three run in the main Ghost FTP CI workflow. The distro-specific layers run in `.github/workflows/linux-distro-packages.yml` and `.github/workflows/linux-distro-install.yml`. A green result from one layer does not substitute for a failure in another layer when that layer is part of the change being verified.

## Core Go gates

The maintained toolchain is Go **1.27.1**. Production checks include:

```text
go telemetry off
gofmt
go test -race ./...
go vet ./...
```

CI uses `GOTOOLCHAIN=local`, `GOPROXY=off` and `GOSUMDB=off` so the release quality gate cannot silently acquire an undeclared Go dependency.

## Repository audits

The quality job executes:

```text
python scripts/audit_brand_hardcut.py
python scripts/audit_repository.py
python scripts/audit_platform_contract.py
python scripts/audit_desktop_surface.py
python scripts/audit_dependencies.py
python scripts/audit_version.py
python scripts/audit_localization.py
python scripts/audit_security.py
python scripts/audit_privacy.py
python scripts/audit_docs.py
python scripts/audit_release.py
```

and then the Python tooling regressions:

```text
python -m unittest discover -s scripts -p 'test_*.py'
```

## Protocol and connection regressions

The maintained loopback FTP test server exercises real protocol behavior rather than only mocks. Coverage includes authentication, listing, directory creation, upload, size/list verification, rename, download with byte-content equality, delete and final listing, plus invalid credentials.

The application connection-manager lifecycle used by both native frontends is covered for:

- successful `remote.Manager.Connect()`;
- initial remote list/operation access;
- connection identity/generation propagation;
- clean `Disconnect()` behavior;
- invalid FTP credentials;
- FTPS against a plaintext-only FTP endpoint failing instead of silently downgrading to plain FTP.

External public FTP/SFTP services are not required for deterministic CI. SFTP host-key, credential-lifetime, tool invocation and trust behavior are covered by local regression/unit paths and production platform builds.

## High-risk regression areas

Go tests cover, among other areas:

- FTP/FTPS/SFTP validation and protocol/tool behavior;
- SFTP host-key fingerprints and key paths;
- runtime secret/AskPass handling and protected-secret ownership;
- pending SFTP trust credential cleanup/transfer;
- endpoint/profile binding;
- process lifecycle and disconnect races;
- transfer staging, source snapshots and commit cleanup;
- retry/cancel/terminal state correctness;
- symlink/reparse-aware filesystem operations;
- settings/profile validation and recovery;
- Classic Light/Dark persisted appearance behavior;
- FTPS fresh/default protocol policy;
- privacy-safe connection diagnostics;
- truthful transfer metrics.

## Windows production gate

`BUILD-WINDOWS.ps1` produces and verifies:

```text
Setup x64
Setup x86
Portable x64
Portable x86
```

The release assembly creates `Ghost-FTP-X.Y.Z-Setup-x32.exe` as a byte-identical compatibility alias of the verified x86 Setup artifact. `x32` is not a separate architecture build.

CI validates executable/package metadata and runs the Authenticode pipeline smoke test with a short-lived development certificate. Production publication does not convert that test identity into a trusted publisher. When real protected Authenticode secrets are configured, the release workflow signs and verifies each Windows artifact; when they are absent, the release is explicitly marked unsigned.

## Canonical Linux production gate

`linux/BUILD.sh` builds the canonical next-release Linux family for:

```text
amd64
arm64
i386
```

For every architecture it produces a generic DEB and generic portable `.tar.gz` archive from the same compiled executable. CI verifies DEB package name/version/architecture, extracts both formats and compares the `ghostftp` executable byte-for-byte. The release workflow consumes this canonical stage and creates the multiarch ZIP from the verified generic DEBs.

## Supplemental distro-package gate

`linux/BUILD-DISTROS.sh` builds one executable per Go architecture and packages it as:

- Debian DEB for `amd64`, `arm64`, `i386`;
- Ubuntu DEB for `amd64`, `arm64`, `i386`;
- Fedora RPM for `x86_64`, `aarch64`, `i686`;
- distro-neutral Portable tar.gz for `amd64`, `arm64`, `i386`.

`.github/workflows/linux-distro-packages.yml` requires both DEB and RPM tooling, validates package metadata and proves byte-for-byte executable parity between Debian, Ubuntu, Fedora and Portable packages for each architecture mapping.

This gate proves packaging construction and payload parity. It does not by itself claim that each package was installed and launched under every target architecture.

## Native distro installation gate

`.github/workflows/linux-distro-install.yml` rebuilds packages from the exact checked-out source revision and runs a clean-container package lifecycle on:

```text
Debian 13 amd64
Ubuntu 26.04 LTS amd64
Fedora 44 x86_64
```

For each target the verifier checks:

- `/etc/os-release` identity/version;
- package metadata and target architecture;
- package-manager dependency resolution;
- real installed package state and package ownership;
- availability of `curl`, `ssh`, `sftp` and CA trust;
- required desktop executable/entry/icon files;
- startup of installed `/usr/bin/ghostftp` under local Xvfb;
- package removal;
- absence of package-owned system residue.

The GUI smoke uses a private HOME under `/var/lib`, pre-creates the normal `$HOME/.local/share` trusted data root and uses a private `XDG_RUNTIME_DIR`. Xvfb runs with `-nolisten tcp`. The test reproduces normal Linux desktop prerequisites instead of weakening production safe-path validation.

Fedora verification does not assume that the RPM requirement named `curl` maps to a package literally named `curl`; it verifies the installed executable and its RPM owner, allowing a provider such as `curl-minimal`. CA trust is derived from the installed `ca-certificates` RPM. The Ghost FTP Fedora transaction clears minimal-container `tsflags=nodocs` so LICENSE and README payload files are installed and verified as part of the lifecycle.

Native package-manager/runtime coverage is deliberately limited to x86-64. `arm64`/`aarch64` and `i386`/`i686` remain protected by exact-head build, metadata, extraction and payload-parity gates, but are not described as native install-tested until such infrastructure exists.

## Desktop/UI regression

Windows UI regression tests protect native workspace geometry, connection/action state, Site Manager behavior, localization, keyboard workflow and screenshot capture contracts.

Fresh/fallback appearance must resolve to Classic Light while an explicitly persisted Dark choice remains Dark. Fresh quick-connect protocol must resolve to explicit FTPS/21 while plain FTP remains an explicit compatibility choice. Privacy-sensitive profile credential persistence uses the same opt-in semantics in the main profile flow and Site Manager.

Linux tests protect shared Engine access, SFTP password/key/passphrase parity, queue controls, settings/profile behavior and native renderer operation. Idle redraw behavior is optimized so unchanged state does not force unnecessary full-workspace redraw.

## Authentic screenshot gate

Documentation screenshots in `docs/images/` are not hand-authored release evidence. The dedicated Windows screenshot workflow builds and launches the real x64 Portable executable, captures the maintained main workspace and Site Manager windows, verifies the outputs and persists only those authentic screenshots.

A screenshot workflow failure is a release-documentation failure, not permission to substitute a mockup.

## Localization gate

Localization checks require exactly 24 canonical languages, English default/fallback, valid catalog keys/format verbs, Windows live localization, Setup primary copy coverage and Linux runtime switching. Security/privacy-sensitive profile credential consent copy is part of the maintained localization surface.

## Privacy gate

Privacy audit rejects fixed product telemetry URLs, known tracking vendor markers, forbidden general-purpose network imports in runtime source, credential-file regressions and ineffective build telemetry controls.

The release-package contract additionally ensures the GHCR bundle copies only the verified release directory and builds with Docker networking disabled.

## Release gate

`.github/workflows/release.yml` runs the quality, Windows and canonical Linux jobs before publication.

The **already published 1.1.6** GitHub Release remains immutable with **9 platform artifacts / 12 public files**.

The maintained canonical workflow for a later release currently assembles **12 platform artifacts / 15 public files**: Windows Setup/Portable outputs, generic Linux DEBs, generic Linux portable tarballs, the Linux multiarch ZIP and release metadata/checksum files. Supplemental distro-specific Debian/Ubuntu/Fedora/Portable CI packages are not part of that public release count until the release workflow explicitly stages, allow-lists, hashes, publishes and reads them back.

Before and after publication, the workflow verifies that `main` is still the exact release commit and that an existing version tag is not being rewritten.

The release gate also validates that `WINDOWS_SIGNING_STATE` is either `signed` or `unsigned`. A configured signing identity that does not produce valid signatures fails. Absence of a production certificate does not fail the release; it is carried through as `WINDOWS_AUTHENTICODE=unsigned` in release metadata.

## GitHub Packages gate

Stable releases publish:

```text
ghcr.io/bren-wp/ghost-ftp:<version>
```

The OCI distribution bundle is built from `FROM scratch`, copies only the canonical assembled `release/` directory, publishes stable semantic aliases and is read back through the registry before the release job is considered complete. It is a distribution bundle, not a runtime application container.

## Exact-head and post-merge rule

A branch result is authoritative only for the exact code head that produced it. Any code-head change invalidates earlier green results for merge purposes.

Packaging changes are merged only after their relevant exact-head gates and the existing Ghost FTP CI are green. After merge, the corresponding push workflows on the resulting `main` SHA must also finish successfully before the phase is considered complete.

## Release-readiness rule

A source branch that merely compiles is not a release. Stable readiness requires all automated gates plus exact artifact/signing-state/package/release verification on the final source revision.

See [Release verification](RELEASE-VERIFICATION.md), [Installation](INSTALLATION.md), [Platform parity](PLATFORM-PARITY.md), [Security](SECURITY.md), [Privacy](PRIVACY.md) and [Packages](PACKAGES.md).
