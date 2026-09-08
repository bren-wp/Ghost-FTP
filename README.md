# Ghost FTP

**Ghost FTP** is a privacy-first native desktop file-transfer client for **Windows and Linux**, developed and published by **BRENDIGO LTD**. It provides a professional dual-pane workstation for **FTP, FTPS and SFTP**, local profiles, protected saved-secret handling, bounded transfer management and verified release packaging without application telemetry.

- Current Ghost FTP version: **1.1.7**
- Development status: **Stable**
- Release channel: **Stable**
- First stable release: **Ghost FTP 1.0.0**
- Default language: **English**
- Selectable local languages: **24 languages**
- Official product website: **https://ghostftp.com**
- Developer/publisher: **BRENDIGO LTD — https://brendigo.com**

- Releases: https://github.com/bren-wp/Ghost-FTP/releases
- Packages: https://github.com/users/bren-wp/packages?repo_name=Ghost-FTP
- Repository: https://github.com/bren-wp/Ghost-FTP

![Ghost FTP main workspace](docs/images/ghost-ftp-main-workspace.png)

## 1.1.7 Stable — native UI, localization and distribution quality

Ghost FTP 1.1.7 is a backward-compatible Windows/Linux maintenance release that consolidates the post-1.1.6 native desktop, localization, packaging and transfer-hardening work into one verified Stable release.

### Windows desktop and localization

- Confirm, information and error prompts now prefer application-owned Ghost FTP DecisionCard windows instead of mixing the selected application theme with unrelated stock surfaces.
- Native modal families share the same Light/Dark palette, DPI scaling, modal-owner behavior and keyboard/message-loop contract.
- DecisionCard heading/body geometry expands for longer localized or security-sensitive text while preserving the compact 680×320 presentation for short messages.
- OK, Cancel, Yes and No are resolved from the live desktop locale when a dialog opens, including asynchronous confirmation paths.
- Save Profile privacy/security decisions are localized across all 24 canonical languages without changing credential-retain/remove semantics.
- Native SSH private-key and local-folder pickers now resolve their titles and filter labels from the active runtime language instead of using hardcoded Croatian copy.
- Authentic Windows x64 Portable screenshots remain the release-evidence source for Main Workspace, Site Manager, Settings and About.

### Linux packaging and verification

The canonical 1.1.7 release adds package-manager-neutral Linux portable archives for `amd64`, `arm64` and `i386` alongside the matching DEBs. The release workflow verifies archive structure and byte-for-byte executable parity between each portable archive and `/usr/bin/ghostftp` from the corresponding DEB before publication.

The canonical public release shape is now **12 platform artifacts / 15 public files**.

Separately, maintained **supplemental distro-specific CI packages** are built with `linux/BUILD-DISTROS.sh` for Debian, Ubuntu, Fedora and a distro-neutral Portable family. Their native package lifecycle/GUI smoke matrix is green on **Debian 13 amd64**, **Ubuntu 26.04 LTS amd64** and **Fedora 44 x86_64**. Those supplemental distro-labelled packages are **not yet part of the canonical release allow-list**; canonical 1.1.7 publication uses the generic DEB/tar.gz release family below.

### Security and transfer invariants

The 1.1.7 line preserves the established security boundary: FTPS certificate/hostname validation, explicit secure-protocol selection with no silent downgrade, SFTP host-key verification/pinning, protected-secret ownership/lifetime rules, path containment, staged transfer activation/rollback, connection-generation guards and privacy-safe diagnostics. Post-1.1.6 transfer work also strengthens rooted tree preparation and the regression coverage around filesystem/path transitions.

No external Go module dependency, telemetry, analytics, advertising, tracking SDK, remote UI runtime or hidden product network service is introduced.

## Privacy by design

Ghost FTP does not include application analytics, advertising, tracking pixels, fingerprinting, automatic crash upload, a mandatory product account or hidden profile synchronization. Go telemetry is explicitly disabled in production CI and release workflows.

Normal network activity is limited to user-directed FTP/FTPS/SFTP operations and operating-system tools required by those protocols. Saved credentials are opt-in and local. Windows uses the current-user protection boundary; Linux persistent profile state remains local and platform-protected according to the documented storage model.

See [Privacy](docs/PRIVACY.md) and [Security](docs/SECURITY.md).

## Desktop workflow

Ghost FTP uses a two-pane workstation:

- **Local** pane for files on the current computer;
- **Remote** pane for the connected server;
- **Site Manager** for saved connection profiles;
- **Transfers** for queued, running and completed operations;
- connection diagnostics and status surfaces;
- keyboard-first navigation, sorting, selection and file actions;
- language, appearance, transfer and connection preferences.

The Windows frontend uses native Win32 drawing and controls. The Linux frontend uses the maintained X11/XWayland-compatible native path. Both consume the same typed Core behavior rather than separate protocol engines.

![Ghost FTP Site Manager](docs/images/ghost-ftp-site-manager.png)

## Appearance and local assets

Classic Light remains the fresh/fallback appearance. An explicitly saved Dark preference is preserved. Windows icons remain local to the operating system: Segoe Fluent Icons is preferred where available and Segoe MDL2 Assets is the compatibility fallback.

## Supported protocols

### FTPS — fresh default

A fresh connection uses explicit FTPS on port 21. TLS certificate/hostname validation is preserved. Failed TLS negotiation is not silently converted into plain FTP.

### SFTP

SFTP uses SSH transport semantics with host-key verification. Password and key-based authentication are supported through the maintained system-tool integration. Trust fingerprints remain bound to the scanned public-key material.

### FTP — explicit compatibility

Standard FTP remains available when a legacy server explicitly requires it. It is unencrypted and is not selected as the fresh default.

## Windows installation

```text
Ghost-FTP-1.1.7-Setup-x64.exe
Ghost-FTP-1.1.7-Setup-x86.exe
Ghost-FTP-1.1.7-Setup-x32.exe
Ghost-FTP-1.1.7-Portable-x64.exe
Ghost-FTP-1.1.7-Portable-x86.exe
```

`x32` is a byte-identical compatibility alias of the verified x86 Setup build; it is not a separate architecture build. Setup installs per user and uses staged/rollback-oriented replacement. Portable runs without normal installation registration.

Production Authenticode is optional. When a trusted certificate is configured through protected Actions secrets, produced signatures must verify. When it is absent, the official release is explicitly unsigned and `BUILD-METADATA.txt` records `WINDOWS_AUTHENTICODE=unsigned`; Ghost FTP never fabricates a trusted publisher identity.

See [Installation](docs/INSTALLATION.md) and [Signing](docs/SIGNING.md).

## Linux installation

Canonical 1.1.7 Linux files are:

```text
Ghost-FTP-1.1.7-Linux-amd64.deb
Ghost-FTP-1.1.7-Linux-arm64.deb
Ghost-FTP-1.1.7-Linux-i386.deb
Ghost-FTP-1.1.7-Linux-multiarch.zip
Ghost-FTP-1.1.7-Linux-amd64.tar.gz
Ghost-FTP-1.1.7-Linux-arm64.tar.gz
Ghost-FTP-1.1.7-Linux-i386.tar.gz
```

The DEB metadata is generated from root `VERSION`, uses `https://ghostftp.com` as Homepage and identifies BRENDIGO LTD as publisher/maintainer. The generic tar.gz archives are package-manager-neutral alternatives built from the same per-architecture executable as the matching DEB.

Supplemental distro-specific build/parity CI covers Debian/Ubuntu/Fedora/Portable package families and native x86-64 install/remove/UI smoke on the three documented distributions. These supplemental files remain CI evidence and are not canonical public 1.1.7 assets.

See [Linux documentation](linux/README.md), [Installation](docs/INSTALLATION.md) and [Testing](docs/TESTING.md).

## Releases and GitHub Packages

Ghost FTP 1.1.7 publishes **12 platform artifacts / 15 public files**: five Windows files, seven Linux files, `BUILD-METADATA.txt`, `RELEASE-NOTES.txt` and `SHA256.txt`.

The stable GitHub Packages distribution bundle is:

```text
ghcr.io/bren-wp/ghost-ftp:1.1.7
```

The OCI object mirrors `/ghostftp-release/` from the verified release assembly and is **not a runtime container**. Compatible aliases are updated only after successful publication and registry read-back.

See [GitHub Packages](docs/PACKAGES.md), [GitHub Releases](docs/GITHUB-RELEASES.md) and [Release verification](docs/RELEASE-VERIFICATION.md).

## Artifact verification

Every public release contains `SHA256.txt`. `BUILD-METADATA.txt` binds the version, release tag, source commit, platform set, Windows signing state, language count and package reference to the verified release assembly.

## Languages

Ghost FTP ships **24 languages** selectable locally, with English as the default/fallback. Localization is resolved locally; the desktop client does not send filenames, hostnames, credentials or UI strings to a translation service.

See [Localization](docs/LOCALIZATION.md).

## Build from source

Ghost FTP uses Go **1.27.1**. Root `VERSION` is the production version source of truth.

```text
go telemetry off
go test -race ./...
go vet ./...
```

Windows release-style packages:

```powershell
.\BUILD-WINDOWS.ps1
```

Canonical Linux release packages:

```bash
bash linux/BUILD.sh
```

Supplemental distro-specific CI packages:

```bash
GHOSTFTP_REQUIRE_DEB=1 GHOSTFTP_REQUIRE_RPM=1 bash linux/BUILD-DISTROS.sh
```

Official public artifacts are produced only by the canonical release workflow after exact-source audits/tests/builds succeed.

## Dependency policy

The maintained Go module has no external module requirements. Production workflows use `GOPROXY=off`, `GOSUMDB=off`, pinned GitHub Actions and dependency/platform audits.

## Documentation

Start with the [documentation index](docs/README.md). Key documents include [Architecture](docs/ARCHITECTURE.md), [Installation](docs/INSTALLATION.md), [Settings](docs/SETTINGS.md), [Reference UI](docs/REFERENCE-UI.md), [Localization](docs/LOCALIZATION.md), [Platform parity](docs/PLATFORM-PARITY.md), [Security](docs/SECURITY.md), [Privacy](docs/PRIVACY.md), [Testing](docs/TESTING.md), [Signing](docs/SIGNING.md), [GitHub Releases](docs/GITHUB-RELEASES.md), [GitHub Packages](docs/PACKAGES.md), [Release verification](docs/RELEASE-VERIFICATION.md), [Versioning](docs/VERSIONING.md), [Roadmap](docs/ROADMAP.md), [Support](docs/SUPPORT.md) and [Contributing](docs/CONTRIBUTING.md).

Historical release text remains historical and is not rewritten to imply that older immutable tags contained later changes.

## License

Ghost FTP is source-available proprietary software. Source visibility does not grant permission to redistribute, rebrand, sublicense, sell or operate derivative commercial distributions unless the license explicitly permits it.

See [LICENSE](LICENSE).

Copyright © 2026 **BRENDIGO LTD**. All rights reserved.
