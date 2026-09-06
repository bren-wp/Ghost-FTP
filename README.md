# Ghost FTP

**Ghost FTP** is a privacy-first FTP, FTPS and SFTP client for **Windows and Linux**. It is designed for dependable day-to-day server administration, clear dual-pane file management, conservative security defaults and a modern desktop workflow without application telemetry, advertising or hidden tracking.

Current Ghost FTP version: **0.2.1**

Development status: **Beta**

Ghost FTP follows the pre-stable `0.x.y` line until the Windows/Linux product, packaging and long-term compatibility gates are ready for the first stable **1.0.0** release.

## Supported desktop platforms

- **Windows** — native Win32 desktop application with x64 and x86 Setup and Portable editions.
- **Linux** — the same transfer/security engine with a native X11/XWayland-compatible graphical frontend, terminal fallback and amd64, arm64 and i386 packages.

The active application source, CI contract, documentation and release matrix are desktop-only. Windows and Linux are the only maintained application platforms.

## What 0.2.1 changes

Ghost FTP 0.2.1 is a Windows/Linux polish and reliability release driven by real runtime screenshots and connection-path regression testing.

Key changes include:

- complete **Ghost FTP** branding in the Windows workspace using the canonical packaged PE icon;
- immersive dark-mode integration for native Windows menus, combo boxes, file headers and the Site Manager;
- owner-drawn Site Manager actions matching the main premium toolbar instead of mixed bright native buttons;
- batched, non-erasing Windows resize redraws plus narrower repaint regions during connection-state changes to reduce visible flicker;
- the persisted **Connection timeout** setting now controls real Windows and Linux connection attempts instead of being bypassed by a hard-coded timeout;
- a pending Windows connection can be cancelled immediately through the visible Disconnect control rather than forcing the user to wait for timeout;
- secure Windows x86/WOW64 `Sysnative` resolution for both the OS `curl.exe` FTP/FTPS transport and Windows OpenSSH/SFTP tools, without trusting user `PATH`;
- additional Linux GUI localization through the shared 24-language registry and removal of the idle 750 ms full-window repaint when transfer state has not changed;
- continued zero-telemetry, zero-external-Go-module, explicit SFTP host-key trust and fail-closed path/credential protections;
- refreshed automated visual, transport, timeout, localization and platform-parity regression coverage.

See [CHANGELOG.md](CHANGELOG.md) for the complete version-by-version record.

## Core product principles

Ghost FTP is built around these enforced rules:

- no application telemetry, analytics, advertising SDKs or external crash-reporting SDKs;
- no fixed analytics or hidden update endpoint in the desktop runtime;
- no browser bridge or generic localhost control API between UI and transfer engine;
- fail-closed connection, path, credential and host-key validation;
- explicit SFTP host-key trust and endpoint binding;
- bounded connection timeouts, retry counts and retry delays;
- guarded atomic settings/profile writes;
- conservative overwrite recovery;
- English as the canonical language and safe fallback;
- 24 maintained UI languages;
- zero external Go modules in the desktop/core Go module;
- Windows and Linux builds produced from the same typed application engine.

## FTP, FTPS and SFTP

### FTP

Plain FTP is available for servers that require it. FTP is not encrypted, so FTPS or SFTP should be preferred whenever the server supports them.

### FTPS

Ghost FTP supports:

- explicit FTPS;
- implicit FTPS through the dedicated `ftpsi` transport mode;
- TLS certificate validation;
- normal server-side directory and file operations supported by the shared engine.

TLS verification is not silently disabled.

### SFTP

Ghost FTP supports:

- password authentication;
- private-key authentication;
- passphrase-protected private keys;
- explicit host-key fingerprint trust;
- saved fingerprint binding to the expected endpoint;
- application-created SSH configuration with proxy/jump/agent forwarding disabled;
- protected runtime credential handoff without persistent AskPass secret files.

## File-management workflow

The maintained desktop workflow includes:

- Quick Connect;
- saved Sites/profiles;
- local and remote file panes;
- local and remote navigation;
- upload and download;
- local create folder, rename and delete;
- remote create folder, rename, delete and permissions/chmod;
- validated remote permission display when the server actually supplies usable mode metadata;
- transfer queue;
- pause and resume;
- cancel and retry;
- clear completed jobs;
- bounded parallel transfer execution;
- safe conflict policies;
- connection timeout control;
- delete confirmation.

Ghost FTP deliberately does not show fake controls for backend functionality that does not exist.

## Windows experience

Windows uses a native Win32 interface and remains the reference graphical layout. The 0.2.0 workspace uses one canonical information architecture:

1. brand/status/language header;
2. Sites/profile controls;
3. Quick Connect;
4. balanced local and remote file panes;
5. direct file-operation controls;
6. transfer queue and transfer actions;
7. concise connection and queue status.

The main window is a normal resizable Windows desktop window and supports minimize, maximize, restore and DPI-aware relayout. Connection-specific controls are shown only when relevant so the interface stays clean at both compact and expanded window sizes.

Setup and Portable use the same application UI and transfer/security engine.

### Windows Setup and uninstall

Windows Setup remains a single premium installation flow. It validates its embedded payload before replacement, stages installation changes conservatively and preserves rollback behavior.

Ghost FTP does **not** ship a separate `Uninstall.exe`. The installed application registers an integrated maintenance command with Windows Installed Apps and performs uninstall through the installed executable. Portable copies cannot invoke the installed-app uninstall path.

Application files, shortcuts and Windows registrations are removed during uninstall. Saved profiles and settings are preserved by default to avoid accidental configuration loss.

## Authentic Windows UI

The repository screenshots are generated from the real production Windows executable, not hand-drawn mockups. The screenshot workflow builds the Windows x64 Portable application, launches it on a Windows runner, captures the native windows and validates the resulting PNG evidence before the repository images are refreshed.

### Main workspace

![Ghost FTP authentic Windows main workspace](docs/images/ghost-ftp-main-workspace.png)

### Site Manager

![Ghost FTP authentic Windows Site Manager](docs/images/ghost-ftp-site-manager.png)

The capture flow contains no production FTP credentials, customer data or real server secrets. See `.github/workflows/ui-screenshots.yml` and `scripts/capture_windows_screenshots.ps1`.

## Linux experience

Linux uses the same typed `internal/api.Engine`, remote/session layer, transfer scheduler, settings model, profiles, localization registry and security boundaries as Windows.

The graphical frontend provides:

- FTP/FTPS password connections;
- SFTP password connections;
- SFTP private-key authentication with optional passphrase;
- explicit host-key trust through the shared engine;
- local and remote list/navigation operations;
- create, rename, delete and remote chmod operations;
- upload/download jobs through the shared transfer scheduler;
- transfer queue controls;
- the same validated parallelism, conflict, retry, timeout and delete-confirmation settings;
- runtime language selection through the canonical 24-language registry.

Linux retains a hardened terminal interface for headless environments or explicit fallback use. The native control implementation is platform-specific, but the product actions, settings semantics, palette and security contract are shared with Windows.

## Languages

English (`en`) is the canonical source language, default language and fallback for invalid or missing locale state.

Ghost FTP currently maintains **24 languages**:

1. English (`en`)
2. Croatian (`hr`)
3. German (`de`)
4. French (`fr`)
5. Spanish (`es`)
6. Turkish (`tr`)
7. Greek (`el`)
8. Portuguese (`pt`)
9. Chinese, Simplified (`zh`)
10. Russian (`ru`)
11. Hindi (`hi`)
12. Japanese (`ja`)
13. Italian (`it`)
14. Polish (`pl`)
15. Dutch (`nl`)
16. Czech (`cs`)
17. Ukrainian (`uk`)
18. Swedish (`sv`)
19. Romanian (`ro`)
20. Hungarian (`hu`)
21. Danish (`da`)
22. Finnish (`fi`)
23. Norwegian (`no`)
24. Korean (`ko`)

Regional aliases are normalized to the canonical registry and missing translation state falls back safely to English. CI validates registry consistency, catalog parity, format verbs and translation coverage.

## Real settings and safe defaults

The canonical settings model is intentionally small and functional:

| Option | Supported range / values | Default |
| --- | --- | --- |
| Parallel transfers | 1–8 | 2 |
| Conflict policy | `skip`, `replace`, `replace_backup` | `replace_backup` |
| Automatic retries | 0–3 | 0 |
| Retry delay | 1–30 seconds | 3 seconds |
| Connection timeout | 5–60 seconds | 15 seconds |
| Confirm delete | on/off | on |
| Language | 24 registered locales | English |

`replace_backup` is the conservative overwrite default. Unknown persisted conflict states migrate back to the conservative policy instead of silently weakening recovery behavior.

## Privacy

Ghost FTP does not contain application analytics, advertising, tracking pixels, marketing SDKs or an application-controlled telemetry service.

The installed application communicates with the server address entered by the user for FTP, FTPS or SFTP transfers. CI rejects known tracking/advertising/crash SDK markers and unexpected dependency drift.

See [docs/PRIVACY.md](docs/PRIVACY.md).

## Security model

Important enforced boundaries include:

- credentials do not pass through a generic JSON command dispatcher;
- Windows saved profile secrets use OS-backed protection;
- short-lived runtime secrets are not deliberately persisted as plain-text helper files;
- SFTP host keys are explicitly trusted and endpoint-bound;
- application SFTP sessions disable proxy, jump-host and agent-forwarding inheritance;
- FTP/FTPS proxy environment state is sanitized by the application transport boundary;
- local path traversal and unsafe root deletion are rejected;
- download staging rejects unsafe substitution paths;
- local rename/delete operations use conservative path protections;
- remote session shutdown is bounded and race-protected;
- profile credentials cannot silently cross incompatible endpoint/private-key identities;
- settings/profile state uses guarded atomic writes.

See [docs/SECURITY.md](docs/SECURITY.md).

## Dependencies

The desktop/core Go module intentionally has **zero external Go modules**. CI rejects a `require`, `replace`, vendored module tree or `go.sum` dependency graph in the application module.

The current transport implementation uses operating-system transport executables rather than bundled Go libraries:

- FTP/FTPS: `curl`
- SFTP: OpenSSH `ssh` and `sftp`

Linux packages declare the corresponding runtime requirements. Windows requires suitable system-provided transport components. This distinction is documented explicitly rather than presenting OS tools as bundled application dependencies.

See [docs/DEPENDENCIES.md](docs/DEPENDENCIES.md).

## Build from source

Required project toolchain:

- Go **1.27.1**
- Python 3 for repository/release verification scripts

For local production-oriented development, disable Go toolchain telemetry:

```text
go telemetry off
```

### Windows

```text
.\BUILD-WINDOWS.ps1
```

Canonical 0.2.0 Windows packages:

```text
Ghost-FTP-0.2.0-Setup-x64.exe
Ghost-FTP-0.2.0-Setup-x86.exe
Ghost-FTP-0.2.0-Portable-x64.exe
Ghost-FTP-0.2.0-Portable-x86.exe
```

### Linux

```text
bash linux/BUILD.sh
```

The canonical Linux build creates DEB packages for amd64, arm64 and i386 plus the multiarch release bundle.

See [docs/INSTALLATION.md](docs/INSTALLATION.md) and [docs/TESTING.md](docs/TESTING.md).

## Versioning

The active release line is:

```text
0.1.0 Beta → 0.1.1 Beta → 0.2.0 Beta → 0.x.y Beta → 1.0.0 stable
```

Every `0.x.y` build is a Beta/prerelease. The first stable milestone is reserved for **1.0.0** after the maintained Windows/Linux application, packaging, security, localization, documentation and release-verification gates are complete.

Historical releases remain repository provenance; the detailed chronological record is kept in [CHANGELOG.md](CHANGELOG.md) and [docs/RELEASE-HISTORY.md](docs/RELEASE-HISTORY.md).

## Release contract

Release tags use:

```text
ghostftp-vX.Y.Z
```

The Windows/Linux release contract contains **9 platform artifacts**:

- Windows Setup x64
- Windows Setup x86
- Windows Setup x32 compatibility alias of x86
- Windows Portable x64
- Windows Portable x86
- Linux amd64 DEB
- Linux arm64 DEB
- Linux i386 DEB
- Linux multiarch ZIP

A complete release also contains:

- `RELEASE-NOTES.txt`
- `BUILD-METADATA.txt`
- `SHA256.txt`

That produces **12 public files** per complete desktop release.

Published release tags are immutable. The release workflow refuses to move an existing release tag to another commit.

## Repository quality gates

CI validates:

- Go formatting;
- unit/integration tests and race-sensitive core tests;
- `go vet`;
- repository path/case/generated-file integrity;
- Windows/Linux-only application platform contract;
- canonical version/release artifact contract;
- Beta versus stable release-channel rules;
- zero external Go module drift;
- no tracking/analytics/ads/crash SDK dependency markers;
- runtime privacy boundaries;
- credential and transport security invariants;
- localization registry/catalog parity;
- documentation links and release markers;
- Windows production build;
- Linux amd64/arm64/i386 package builds;
- authentic Windows UI capture.

## Documentation

- [Documentation index](docs/README.md)
- [Reference UI contract](docs/REFERENCE-UI.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Installation](docs/INSTALLATION.md)
- [Platform parity](docs/PLATFORM-PARITY.md)
- [Localization](docs/LOCALIZATION.md)
- [Settings](docs/SETTINGS.md)
- [Dependencies](docs/DEPENDENCIES.md)
- [Security](docs/SECURITY.md)
- [Privacy](docs/PRIVACY.md)
- [Testing](docs/TESTING.md)
- [Versioning](docs/VERSIONING.md)
- [Release history](docs/RELEASE-HISTORY.md)
- [GitHub Releases](docs/GITHUB-RELEASES.md)
- [Release verification](docs/RELEASE-VERIFICATION.md)
- [Signing](docs/SIGNING.md)
- [Roadmap](docs/ROADMAP.md)
- [Third-party notices](docs/THIRD-PARTY-NOTICES.md)
- [Contributing](docs/CONTRIBUTING.md)
- [Support](docs/SUPPORT.md)
- [Linux packaging](linux/README.md)

## Current development status

**0.2.0 Beta** is the current desktop baseline. It keeps the existing protocol/security implementation, removes retired application surfaces from the active product tree and consolidates the native UI around a cleaner Windows/Linux contract.

The project remains Beta until the first stable checklist is intentionally completed and the canonical `VERSION` advances to `1.0.0`.
