# Ghost FTP

**Ghost FTP** is a privacy-first native desktop file-transfer client for **Windows and Linux**, developed and published by **BRENDIGO LTD**. It provides one professional dual-pane workstation for **FTP, FTPS and SFTP**, with local profiles, encrypted saved-secret handling, bounded transfer management, secure release verification and no application telemetry.

Current Ghost FTP version: **1.0.0**  
Development status: **Stable**  
Release channel: **Stable**  
First stable release: **Ghost FTP 1.0.0**  
Default language: **English**  
Selectable local languages: **24**

- Releases: https://github.com/bren-wp/Ghost-FTP/releases
- Packages: https://github.com/users/bren-wp/packages?repo_name=Ghost-FTP
- Repository: https://github.com/bren-wp/Ghost-FTP

![Ghost FTP main workspace](docs/images/ghost-ftp-main-workspace.png)

## 1.0.0 stable release

Ghost FTP 1.0.0 is the first release published as a normal stable GitHub Release rather than a prerelease. It consolidates the 0.2.x stabilization line and makes the verified Windows/Linux release contract the maintained production baseline.

The stable baseline includes:

- native Windows and maintained Linux desktop frontends backed by the same typed Core/API model;
- FTP, Explicit FTPS, Implicit FTPS and SFTP connection workflows;
- strict host, path, protocol and remote-operation validation;
- SFTP host-key trust and fingerprint validation;
- local-only profile storage with platform-specific saved-secret protection;
- privacy-safe connection diagnostics that classify failures without exposing credentials;
- transfer queue lifecycle with truthful progress, speed and ETA state;
- upload-source snapshot protection and remote commit/revalidation safeguards;
- symlink/reparse-aware local filesystem hardening;
- transactional Windows Setup, rollback, integrated uninstall and Portable builds;
- deterministic Windows/Linux production CI and security/privacy/documentation audits;
- stable GitHub Packages publication of the verified release bundle.

## Privacy by design

Ghost FTP does not include application analytics, advertising, tracking pixels, fingerprinting, automatic crash upload, a mandatory product account or hidden profile synchronization. Go telemetry is explicitly disabled in production CI and release workflows.

Normal network activity is limited to user-directed FTP/FTPS/SFTP operations and the operating-system tools required for those protocols. Connection errors are converted into privacy-safe user-facing categories; passwords, private-key passphrases and protected profile secrets are not intentionally copied into diagnostics.

Saved credentials are opt-in. Windows uses the current-user Windows protection boundary; Linux uses local authenticated encryption with user-private key material. Session-only credentials remain in runtime state unless the user deliberately saves a profile.

See [Privacy](docs/PRIVACY.md) and [Security](docs/SECURITY.md).

## Security model

The maintained security boundary includes:

- host, port, path and protocol validation before connection/transfer;
- TLS certificate/hostname validation for FTPS with no silent downgrade;
- explicit treatment of plain FTP as an unencrypted transport;
- SFTP host-key fingerprint policy and private-key validation;
- bounded process execution and environment handling for system transfer tools;
- staged upload/download behavior and rollback-oriented destination handling;
- remote destination revalidation before final commit where supported;
- local path containment and destructive-operation safeguards;
- resilient profile/settings writes with bounded recovery behavior;
- no private signing material committed to the repository;
- zero external Go module requirements in the maintained source tree.

Security-sensitive behavior is covered by Go regression tests plus repository-level Python audits. See [Security](docs/SECURITY.md), [Testing](docs/TESTING.md) and [Dependencies](docs/DEPENDENCIES.md).

## Desktop workflow

Ghost FTP uses the familiar professional two-pane model:

- **Local** pane for files on the current computer;
- **Remote** pane for the connected server;
- **Site Manager** for saved connection profiles;
- **Transfers** for queued/running/completed operations;
- connection diagnostics and status surfaces;
- keyboard-first navigation, sorting, selection and file actions;
- configurable appearance, language, transfer and connection preferences.

The Windows frontend uses native Win32 drawing and controls. The Linux frontend uses the maintained X11/XWayland-compatible native path. Both consume shared Core behavior rather than separate protocol engines.

![Ghost FTP Site Manager](docs/images/ghost-ftp-site-manager.png)

## Supported protocols

### FTP

Standard FTP is available for compatibility. It is unencrypted and should be used only when the server/environment explicitly requires it.

### FTPS

Ghost FTP supports TLS-protected FTP modes and preserves certificate/hostname validation. Failed TLS negotiation is not silently converted into plain FTP.

### SFTP

SFTP uses SSH transport semantics with host-key verification. Password and key-based authentication are supported by the maintained system-tool integration and validation layer.

## Windows installation

Choose the architecture and packaging mode that matches the machine:

```text
Ghost-FTP-1.0.0-Setup-x64.exe
Ghost-FTP-1.0.0-Setup-x86.exe
Ghost-FTP-1.0.0-Setup-x32.exe
Ghost-FTP-1.0.0-Portable-x64.exe
Ghost-FTP-1.0.0-Portable-x86.exe
```

`x32` is a compatibility alias of the verified x86 Setup build; it is not a separate architecture build.

Setup installs per user, registers integrated maintenance/uninstall information and uses a transaction/rollback path for replacement. Portable runs without installation registration and keeps its portable data boundary separate.

Stable Windows publication remains gated on a configured trusted Authenticode identity in the protected release environment.

See [Installation](docs/INSTALLATION.md) and [Signing](docs/SIGNING.md).

## Linux installation

Stable Linux artifacts are:

```text
Ghost-FTP-1.0.0-Linux-amd64.deb
Ghost-FTP-1.0.0-Linux-arm64.deb
Ghost-FTP-1.0.0-Linux-i386.deb
Ghost-FTP-1.0.0-Linux-multiarch.zip
```

The DEB metadata is generated from the root `VERSION` file and verified in CI/release jobs before publication.

See [Linux documentation](linux/README.md).

## Releases and Packages

The canonical user-installable files are attached to the official GitHub Release. The stable workflow publishes **9 platform artifacts** plus release metadata, notes and `SHA256.txt`, for **12 public files** in total.

Stable releases also publish an OCI **distribution bundle** to GitHub Packages:

```text
ghcr.io/bren-wp/ghost-ftp:1.0.0
```

The GHCR package mirrors `/ghostftp-release/` from the verified release assembly. It is **not a runtime container**; it is a versioned distribution surface for automation, mirrors and integrity verification. Stable aliases include `1.0`, `1` and `latest`.

See [GitHub Packages](docs/PACKAGES.md), [GitHub Releases](docs/GITHUB-RELEASES.md) and [Release verification](docs/RELEASE-VERIFICATION.md).

## Artifact verification

Every public release includes `SHA256.txt`. Verify downloaded files before deployment, especially when a file has been mirrored outside GitHub.

The release also includes `BUILD-METADATA.txt`, which binds the version, tag, source commit, platform set, signing state, language count and package reference to the release assembly.

For automated environments, the GHCR distribution bundle adds an OCI manifest digest on top of the per-file SHA-256 manifest.

## Localization

Ghost FTP ships **24 selectable local languages** with English as the default/fallback. Localization is resolved locally; the desktop client does not send filenames, hostnames, credentials or UI strings to a translation service.

See [Localization](docs/LOCALIZATION.md).

## Build from source

Ghost FTP uses Go **1.27.1**. The root `VERSION` file is the only production version source of truth.

Basic checks:

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

System FTP/SFTP capabilities are treated as explicit runtime dependencies rather than silently bundled network SDKs. See [Dependencies](docs/DEPENDENCIES.md) and [Third-party notices](docs/THIRD-PARTY-NOTICES.md).

## Documentation

Start with the [documentation index](docs/README.md). Key documents include:

- [Architecture](docs/ARCHITECTURE.md)
- [Installation](docs/INSTALLATION.md)
- [Settings](docs/SETTINGS.md)
- [Localization](docs/LOCALIZATION.md)
- [Platform parity](docs/PLATFORM-PARITY.md)
- [Security](docs/SECURITY.md)
- [Privacy](docs/PRIVACY.md)
- [Testing](docs/TESTING.md)
- [Signing](docs/SIGNING.md)
- [GitHub Releases](docs/GITHUB-RELEASES.md)
- [GitHub Packages](docs/PACKAGES.md)
- [Release verification](docs/RELEASE-VERIFICATION.md)
- [Versioning](docs/VERSIONING.md)
- [Roadmap](docs/ROADMAP.md)
- [Support](docs/SUPPORT.md)
- [Contributing](docs/CONTRIBUTING.md)

Historical release notes remain in [Release history](docs/RELEASE-HISTORY.md) and [CHANGELOG](CHANGELOG.md); old version references in those historical sections are retained intentionally.

## License

Ghost FTP is source-available proprietary software. Source visibility does not grant permission to redistribute, rebrand, sublicense, sell or operate derivative commercial distributions unless the license explicitly permits it.

See [LICENSE](LICENSE) for the controlling terms.

Copyright © 2026 **BRENDIGO LTD**. All rights reserved.
