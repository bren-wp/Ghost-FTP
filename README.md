# Ghost FTP

**Ghost FTP** is a privacy-first native desktop file-transfer client for **Windows and Linux**, developed and published by **BRENDIGO LTD**. It provides one professional dual-pane workstation for **FTP, FTPS and SFTP**, with local profiles, protected saved-secret handling, bounded transfer management, secure release verification and no application telemetry.

- Current Ghost FTP version: **1.1.2**
- Development status: **Stable**
- Release channel: **Stable**
- First stable release: **Ghost FTP 1.0.0**
- Default language: **English**
- Selectable local languages: **24 languages**

- Releases: https://github.com/bren-wp/Ghost-FTP/releases
- Packages: https://github.com/users/bren-wp/packages?repo_name=Ghost-FTP
- Repository: https://github.com/bren-wp/Ghost-FTP

![Ghost FTP main workspace](docs/images/ghost-ftp-main-workspace.png)

## 1.1.2 stable native-UI and localization hardening release

Ghost FTP 1.1.2 is a Windows/Linux quality and native-UI hardening release on top of the secure 1.1.x transfer backend. It removes duplicated application navigation, improves native Light/Dark dialog consistency, fixes localization gaps and compact-control clipping, and strengthens authentic production screenshot evidence without expanding the supported platform or protocol surface.

Highlights:

- a **canonical left application sidebar** now provides Language, Site Manager, Settings, Diagnostics and About navigation on Windows;
- the duplicated native top application menu and its obsolete renderer were removed rather than maintained as a second navigation surface;
- Prompt, option/language/settings and About surfaces use the shared native Light/Dark-aware dialog shell;
- runtime About uses a dedicated application-owned card with **BRENDIGO LTD** publisher metadata, public **Ghost FTP** branding and only official `brendigo.com` / `brendigo.com/kontakt` destinations;
- runtime About does not use WebView, GitHub links or hidden network requests;
- all **24 supported languages** have non-empty FTPS explicit/implicit labels and Site Manager/Diagnostics navigation coverage, with English fallback;
- compact Windows actions adapt between icon+label, label-only and intentional icon-only presentation instead of producing accidental clipped labels;
- the About layout reserves sufficient heading/body space for long localized text;
- authentic production Windows screenshots are rebuilt from the real x64 Portable executable for Main Workspace, Site Manager, Settings and About;
- no external Go module dependency, telemetry, analytics, advertising or tracking was added, and the FTP/FTPS/SFTP security backend was not weakened.

The 1.1.2 line retains the established safe defaults from 1.1.1: Classic Light is the fresh/fallback appearance, explicit FTPS on port 21 is the fresh quick-connect protocol, and credential persistence remains explicit and local.

## Privacy by design

Ghost FTP does not include application analytics, advertising, tracking pixels, fingerprinting, automatic crash upload, a mandatory product account or hidden profile synchronization. Go telemetry is explicitly disabled in production CI and release workflows.

Normal network activity is limited to user-directed FTP/FTPS/SFTP operations and the operating-system tools required for those protocols. Connection errors are converted into privacy-safe user-facing categories; passwords, private-key passphrases and protected profile secrets are not intentionally copied into diagnostics.

Saved credentials are opt-in. On Windows, both the main Save Profile flow and Site Manager require explicit consent before newly entered credentials are persisted. Windows uses the current-user Windows protection boundary. Linux persistent profile state remains local and platform-protected according to the documented Linux storage model; session-only credentials are not intentionally promoted into persistent profile state.

See [Privacy](docs/PRIVACY.md) and [Security](docs/SECURITY.md).

## Security model

The maintained security boundary includes:

- host, port, path and protocol validation before connection/transfer;
- **FTPS as the fresh connection default** while retaining explicit plain FTP compatibility;
- TLS certificate/hostname validation for FTPS with no silent downgrade;
- explicit treatment of plain FTP as an unencrypted transport;
- SFTP host-key fingerprint policy and private-key validation;
- ownership-aware lifetime management for protected SFTP password/passphrase handles;
- bounded process execution and environment handling for system transfer tools;
- staged upload/download behavior and rollback-oriented destination handling;
- remote destination revalidation before final commit where supported;
- local path containment and destructive-operation safeguards;
- resilient profile/settings writes with bounded recovery behavior;
- no private signing material committed to the repository;
- zero external Go module requirements in the maintained source tree.

Security-sensitive behavior is covered by Go regression tests plus repository-level Python audits. Real loopback FTP tests cover both the transport lifecycle and the production connection manager. See [Security](docs/SECURITY.md), [Testing](docs/TESTING.md) and [Dependencies](docs/DEPENDENCIES.md).

## Desktop workflow

Ghost FTP uses the familiar professional two-pane model:

- **Local** pane for files on the current computer;
- **Remote** pane for the connected server;
- **Site Manager** for saved connection profiles;
- **Transfers** for queued/running/completed operations;
- connection diagnostics and status surfaces;
- keyboard-first navigation, sorting, selection and file actions;
- a deliberately compact set of language, appearance, transfer and connection preferences.

On Windows, application-level navigation is intentionally centralized in the left sidebar. The operational FTP workspace continues to expose only genuine connection, file and transfer actions, avoiding duplicate Connect/Save/Delete/Transfer commands in navigation chrome.

The Windows frontend uses native Win32 drawing and controls. The Linux frontend uses the maintained X11/XWayland-compatible native path. Both consume shared Core behavior rather than separate protocol engines.

![Ghost FTP Site Manager](docs/images/ghost-ftp-site-manager.png)

## Appearance

**Classic Light is the primary Ghost FTP 1.1.2 appearance.** Fresh installs and invalid/missing appearance state resolve to Classic Light. Windows users who explicitly choose Dark keep that persisted preference.

Appearance remains intentionally a single two-choice Windows setting rather than multiple overlapping switches for background, accent, icon and control colors. The selected Windows appearance is applied on the next application start so the complete native control tree is created consistently, avoiding half-themed controls and repaint races.

Prompt, option/language/settings and About dialogs share an application-owned Light/Dark-aware native shell. This avoids mixed surfaces such as a dark title bar with a stock white body while preserving native controls and avoiding WebView-based UI.

Classic Light uses neutral near-white surfaces, subtle grey borders, dark readable text and a restrained blue selection/accent treatment. It is inspired by the clarity of classic professional FTP clients, but uses Ghost FTP's own iconography, branding and palette rather than copying third-party assets.

## Supported protocols

### FTPS — fresh default

Ghost FTP uses explicit FTPS on port 21 as the fresh/quick-connect default on Windows and Linux. TLS certificate/hostname validation is preserved. Failed TLS negotiation is not silently converted into plain FTP.

### SFTP

SFTP uses SSH transport semantics with host-key verification. Password and key-based authentication are supported by the maintained system-tool integration and validation layer.

### FTP — explicit compatibility

Standard FTP remains available when a legacy server/environment explicitly requires it. It is unencrypted and is not selected as the fresh default.

## Windows installation

Choose the architecture and packaging mode that matches the machine:

```text
Ghost-FTP-1.1.2-Setup-x64.exe
Ghost-FTP-1.1.2-Setup-x86.exe
Ghost-FTP-1.1.2-Setup-x32.exe
Ghost-FTP-1.1.2-Portable-x64.exe
Ghost-FTP-1.1.2-Portable-x86.exe
```

`x32` is a compatibility alias of the verified x86 Setup build; it is not a separate architecture build.

Setup installs per user, registers integrated maintenance/uninstall information and uses a transaction/rollback path for replacement. Portable runs without installation registration and keeps its portable data boundary separate.

Production Authenticode signing is optional. When a trusted certificate is configured in protected Actions secrets, Windows artifacts are signed and verified; when it is absent, the official release remains explicitly unsigned and records that state in `BUILD-METADATA.txt`. Ghost FTP never generates a self-signed certificate and presents it as a trusted production publisher identity.

See [Installation](docs/INSTALLATION.md) and [Signing](docs/SIGNING.md).

## Linux installation

Stable Linux artifacts are:

```text
Ghost-FTP-1.1.2-Linux-amd64.deb
Ghost-FTP-1.1.2-Linux-arm64.deb
Ghost-FTP-1.1.2-Linux-i386.deb
Ghost-FTP-1.1.2-Linux-multiarch.zip
```

The DEB metadata is generated from the root `VERSION` file and verified in CI/release jobs before publication.

See [Linux documentation](linux/README.md).

## Releases and Packages

The canonical user-installable files are attached to the official GitHub Release. The stable workflow publishes **9 platform artifacts** plus release metadata, notes and `SHA256.txt`, for **12 public files** in total.

Stable releases also publish an OCI **distribution bundle** to GitHub Packages:

```text
ghcr.io/bren-wp/ghost-ftp:1.1.2
```

The GHCR package mirrors `/ghostftp-release/` from the verified release assembly. It is **not a runtime container**; it is a versioned distribution surface for automation, mirrors and integrity verification. For the 1.1 stable line, successful publication updates `1.1`, `1` and `latest` only after the exact release workflow passes and registry read-back succeeds.

See [GitHub Packages](docs/PACKAGES.md), [GitHub Releases](docs/GITHUB-RELEASES.md) and [Release verification](docs/RELEASE-VERIFICATION.md).

## Artifact verification

Every public release includes `SHA256.txt`. Verify downloaded files before deployment, especially when a file has been mirrored outside GitHub.

The release also includes `BUILD-METADATA.txt`, which binds the version, tag, source commit, platform set, signing state, language count and package reference to the release assembly.

For automated environments, the GHCR distribution bundle adds an OCI manifest digest on top of the per-file SHA-256 manifest.

## Languages

Ghost FTP ships **24 languages** selectable locally, with English as the default/fallback. Localization is resolved locally; the desktop client does not send filenames, hostnames, credentials or UI strings to a translation service. The maintained 24-language surface includes FTPS explicit/implicit mode labels, Site Manager/Diagnostics navigation and credential-persistence consent.

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
- [Reference UI](docs/REFERENCE-UI.md)
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
