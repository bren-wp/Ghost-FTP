# Ghost FTP

**Ghost FTP** is a privacy-first native desktop file-transfer client for **Windows and Linux**, developed and published by **BRENDIGO LTD**. It provides a professional dual-pane workstation for **FTP, FTPS and SFTP**, local profiles, protected saved-secret handling, bounded transfer management, secure release verification and no application telemetry.

- Current Ghost FTP version: **1.1.5**
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

## 1.1.5 stable identity, quality and documentation release

Ghost FTP 1.1.5 is a backward-compatible Windows/Linux maintenance release focused on public product identity correctness, release-documentation integrity, UI branding consistency and evidence-based repository cleanup.

Highlights:

- **ghostftp.com** is the canonical product website;
- **brendigo.com** is the author/publisher website and **BRENDIGO LTD** remains the publisher identity;
- Windows About separates the product website from the publisher website and support destination;
- public localized strings normalize the technical `GhostFTP` identifier to the user-facing **Ghost FTP** name;
- all 24 supported languages are regression-tested against public-brand leakage and format drift;
- Linux DEB metadata uses `Homepage: https://ghostftp.com` and BRENDIGO LTD Maintainer identity;
- stale release-specific test duplication was removed while evergreen desktop-quality coverage was retained;
- current Installation, Packages, Support and release-verification documentation is bound to root `VERSION` by regression tests;
- authentic Windows x64 Portable screenshots are required for Main Workspace, Site Manager, Settings and About because public Settings/About branding changed;
- no external Go module dependency, telemetry, analytics, advertising, tracking or hidden product network service is added.

The established secure defaults remain unchanged: explicit FTPS on port 21 is the fresh quick-connect protocol, secure transports never silently downgrade to plain FTP, saved credentials remain explicit/local, and Classic Light remains the fresh/fallback Windows appearance while an explicitly saved Dark preference is preserved.

## Privacy by design

Ghost FTP does not include application analytics, advertising, tracking pixels, fingerprinting, automatic crash upload, a mandatory product account or hidden profile synchronization. Go telemetry is explicitly disabled in production CI and release workflows.

Normal network activity is limited to user-directed FTP/FTPS/SFTP operations and the operating-system tools required for those protocols. Connection errors are converted into privacy-safe user-facing categories; passwords, private-key passphrases and protected profile secrets are not intentionally copied into diagnostics.

Saved credentials are opt-in and local. Windows uses the current-user Windows protection boundary. Linux persistent profile state remains local and platform-protected according to the documented Linux storage model.

See [Privacy](docs/PRIVACY.md) and [Security](docs/SECURITY.md).

## Security model

The maintained security boundary includes:

- host, port, path and protocol validation before connection or transfer;
- **FTPS as the fresh connection default** while retaining explicit plain FTP compatibility;
- TLS certificate/hostname validation for FTPS with no silent downgrade;
- SFTP host-key fingerprint policy and private-key validation;
- ownership-aware lifetime management for protected SFTP password/passphrase handles;
- bounded process execution and sanitized environment handling for system transfer tools;
- staged upload/download behavior with rollback-oriented destination handling;
- root-bound local download activation through Go `os.Root`;
- remote destination and directory-type revalidation where supported;
- local path containment and destructive-operation safeguards;
- resilient profile/settings writes with bounded recovery behavior;
- no private signing material committed to the repository;
- zero external Go module requirements in the maintained source tree.

Security-sensitive behavior is covered by Go regression tests plus repository-level Python audits. Real loopback FTP tests cover both transport lifecycle and production connection management.

## Desktop workflow

Ghost FTP uses the professional two-pane file-transfer model:

- **Local** pane for files on the current computer;
- **Remote** pane for the connected server;
- **Site Manager** for saved connection profiles and safe profile duplication;
- **Transfers** for queued, running and completed operations;
- connection diagnostics and status surfaces;
- keyboard-first navigation, sorting, selection and file actions;
- language, appearance, transfer and connection preferences.

On Windows, application navigation is centralized in the left sidebar. The operational workspace exposes genuine connection, file and transfer actions without maintaining duplicate command surfaces.

The Windows frontend uses native Win32 drawing and controls. The Linux frontend uses the maintained X11/XWayland-compatible native path. Both consume the same typed Core behavior rather than separate protocol engines.

![Ghost FTP Site Manager](docs/images/ghost-ftp-site-manager.png)

## Appearance and local icons

**Classic Light is the primary Ghost FTP 1.1.5 appearance.** Fresh installs and invalid/missing appearance state resolve to Classic Light. Windows users who explicitly choose Dark keep that persisted preference.

The native Windows icon path is local to the operating system: Segoe Fluent Icons is preferred when available and Segoe MDL2 Assets is the compatibility fallback. Ghost FTP does not fetch an icon font or UI library from the network.

## Supported protocols

### FTPS — fresh default

Ghost FTP uses explicit FTPS on port 21 as the fresh/quick-connect default on Windows and Linux. TLS certificate/hostname validation is preserved. Failed TLS negotiation is not silently converted into plain FTP.

### SFTP

SFTP uses SSH transport semantics with host-key verification. Password and key-based authentication are supported by the maintained system-tool integration and validation layer. Batch operands are escaped so remote filenames are treated literally rather than as command options or glob patterns.

### FTP — explicit compatibility

Standard FTP remains available when a legacy server explicitly requires it. It is unencrypted and is not selected as the fresh default.

## Windows installation

```text
Ghost-FTP-1.1.5-Setup-x64.exe
Ghost-FTP-1.1.5-Setup-x86.exe
Ghost-FTP-1.1.5-Setup-x32.exe
Ghost-FTP-1.1.5-Portable-x64.exe
Ghost-FTP-1.1.5-Portable-x86.exe
```

`x32` is a compatibility alias of the verified x86 Setup build; it is not a separate architecture build.

Setup installs per user and uses transaction/rollback-oriented replacement. Portable runs without installation registration. Production Authenticode signing is optional: when a trusted certificate is configured in protected Actions secrets, Windows artifacts are signed and verified; when it is absent, the official release remains explicitly unsigned and records that state in `BUILD-METADATA.txt`.

See [Installation](docs/INSTALLATION.md) and [Signing](docs/SIGNING.md).

## Linux installation

```text
Ghost-FTP-1.1.5-Linux-amd64.deb
Ghost-FTP-1.1.5-Linux-arm64.deb
Ghost-FTP-1.1.5-Linux-i386.deb
Ghost-FTP-1.1.5-Linux-multiarch.zip
```

The DEB metadata is generated from root `VERSION`, uses the product homepage `https://ghostftp.com`, identifies BRENDIGO LTD as publisher/maintainer and is verified before publication.

See [Linux documentation](linux/README.md).

## Releases and Packages

The canonical user-installable files are attached to the official GitHub Release. The stable workflow publishes **9 platform artifacts** plus release metadata, notes and `SHA256.txt`, for **12 public files** in total.

Stable releases also publish an OCI **distribution bundle** to GitHub Packages:

```text
ghcr.io/bren-wp/ghost-ftp:1.1.5
```

The package mirrors `/ghostftp-release/` from the verified release assembly and is **not a runtime container**. Successful stable publication updates `1.1`, `1` and `latest` only after registry publication and read-back succeed.

See [GitHub Packages](docs/PACKAGES.md), [GitHub Releases](docs/GITHUB-RELEASES.md) and [Release verification](docs/RELEASE-VERIFICATION.md).

## Artifact verification

Every public release includes `SHA256.txt`. `BUILD-METADATA.txt` binds the version, release tag, source commit, platform set, Windows signing state, language count and package reference to the release assembly.

For automated environments, the GHCR distribution bundle adds an OCI manifest digest on top of the per-file SHA-256 manifest.

## Languages

Ghost FTP ships **24 languages** selectable locally, with English as the default/fallback. Localization is resolved locally; the desktop client does not send filenames, hostnames, credentials or UI strings to a translation service.

Public localization is normalized at the application translation boundary so compatibility strings may retain the internal `GhostFTP` identifier without leaking it into user-facing branding.

See [Localization](docs/LOCALIZATION.md).

## Build from source

Ghost FTP uses Go **1.27.1**. The root `VERSION` file is the only production version source of truth.

```text
go telemetry off
go test -race ./...
go vet ./...
```

Windows release-style packages:

```powershell
.\BUILD-WINDOWS.ps1
```

Linux packages:

```bash
bash linux/BUILD.sh
```

Official public artifacts are produced only by the repository release workflow after the complete audit/test/build contract succeeds.

## Dependency policy

The maintained Go module has no external module requirements. Production workflows run with `GOPROXY=off` and `GOSUMDB=off`, use pinned GitHub Actions revisions and audit dependency/platform drift.

## Documentation

Start with the [documentation index](docs/README.md). Key documents include [Architecture](docs/ARCHITECTURE.md), [Installation](docs/INSTALLATION.md), [Settings](docs/SETTINGS.md), [Reference UI](docs/REFERENCE-UI.md), [Localization](docs/LOCALIZATION.md), [Platform parity](docs/PLATFORM-PARITY.md), [Security](docs/SECURITY.md), [Privacy](docs/PRIVACY.md), [Testing](docs/TESTING.md), [Signing](docs/SIGNING.md), [GitHub Releases](docs/GITHUB-RELEASES.md), [GitHub Packages](docs/PACKAGES.md), [Release verification](docs/RELEASE-VERIFICATION.md), [Versioning](docs/VERSIONING.md), [Roadmap](docs/ROADMAP.md), [Support](docs/SUPPORT.md) and [Contributing](docs/CONTRIBUTING.md).

Historical release notes remain in [Release history](docs/RELEASE-HISTORY.md), the [CHANGELOG](CHANGELOG.md) and immutable published GitHub releases/tags.

## License

Ghost FTP is source-available proprietary software. Source visibility does not grant permission to redistribute, rebrand, sublicense, sell or operate derivative commercial distributions unless the license explicitly permits it.

See [LICENSE](LICENSE) for the controlling terms.

Copyright © 2026 **BRENDIGO LTD**. All rights reserved.
