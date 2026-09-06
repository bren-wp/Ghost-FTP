# Installation

Ghost FTP 0.2.0 Beta is distributed as a desktop application for **Windows** and **Linux** only. The root `VERSION` file is the canonical version source for every application binary and package.

Every `0.x.y` release is a Beta/prerelease. The first version that may be treated as stable is **1.0.0**, after the complete desktop security, stability, packaging and release gate is intentionally satisfied.

Only Windows and Linux desktop packages belong to the active product and release contract. Historical tags/releases remain immutable historical records.

## Windows

Ghost FTP publishes Setup and Portable builds for x64 and x86 Windows systems. Setup and Portable contain the same application functionality and always use the same canonical version.

### Setup packages

Canonical names:

- `Ghost-FTP-X.Y.Z-Setup-x64.exe`
- `Ghost-FTP-X.Y.Z-Setup-x86.exe`
- `Ghost-FTP-X.Y.Z-Setup-x32.exe` — compatibility alias of the x86 Setup artifact

0.2.0 Beta names:

- `Ghost-FTP-0.2.0-Setup-x64.exe`
- `Ghost-FTP-0.2.0-Setup-x86.exe`
- `Ghost-FTP-0.2.0-Setup-x32.exe`

Setup uses the shared Ghost FTP language registry. English is the default/fallback language. The installer validates its embedded payload before activation and uses the stable `GhostFTP` technical identity while presenting **Ghost FTP** as the public product name.

The Setup UI is dependency-free native Win32 and follows the same premium dark visual direction as the application: centered layout, clear hierarchy, larger action targets, explicit primary actions and native Windows title-bar behavior where supported.

Presentation does not bypass installer safety. The installation transaction retains payload digest verification, controlled staging, path checks, rollback-aware activation, shortcut/registry updates and final readback.

### Installed Apps and uninstall

Ghost FTP 0.2.0 registers a normal Windows Installed Apps entry during Setup. The registration includes the installed version, publisher, install location, application icon and uninstall commands.

Ghost FTP does **not** ship a separate `Uninstall.exe`. Uninstallation is integrated into the installed application binary through the protected `GhostFTP.exe --uninstall` mode.

The integrated uninstall path:

- is accepted only from the canonical installed application location;
- removes application-owned shortcuts and Windows registration;
- schedules deletion of the running application binary safely after process exit;
- preserves saved profiles/settings by default so removing or upgrading the application does not silently destroy user configuration;
- is rejected by Portable copies so an arbitrary Portable executable cannot uninstall the installed product.

Setup snapshots the registry values it owns. If installation/upgrade fails before commit, the previous App Paths and Installed Apps registration can be restored as part of rollback.

### Portable packages

Canonical names:

- `Ghost-FTP-X.Y.Z-Portable-x64.exe`
- `Ghost-FTP-X.Y.Z-Portable-x86.exe`

0.2.0 Beta names:

- `Ghost-FTP-0.2.0-Portable-x64.exe`
- `Ghost-FTP-0.2.0-Portable-x86.exe`

Portable builds require no traditional installation. Place the executable in a user-writable folder and run it directly. Portable and Setup builds use the same connection, file-management, transfer, security, localization and settings engine.

### Window behavior

The Windows client is a normal desktop window and supports:

- resize;
- minimize;
- maximize;
- restore;
- DPI-aware relayout;
- a maintained minimum workspace size so controls do not collapse into unusable geometry.

### Windows runtime prerequisites

Ghost FTP does not silently download protocol components. The current transport layer uses operating-system `curl` for FTP/FTPS and OpenSSH `ssh`/`sftp` for SFTP.

On supported Windows installations these are normally available as system components. If a required executable is unavailable or does not provide the required secure capability, Ghost FTP fails with an actionable error instead of downloading an unverified substitute.

## Linux

Ghost FTP publishes Debian packages for:

- amd64;
- arm64;
- i386.

Release file names:

- `Ghost-FTP-X.Y.Z-Linux-amd64.deb`
- `Ghost-FTP-X.Y.Z-Linux-arm64.deb`
- `Ghost-FTP-X.Y.Z-Linux-i386.deb`
- `Ghost-FTP-X.Y.Z-Linux-multiarch.zip`

0.2.0 Beta names:

- `Ghost-FTP-0.2.0-Linux-amd64.deb`
- `Ghost-FTP-0.2.0-Linux-arm64.deb`
- `Ghost-FTP-0.2.0-Linux-i386.deb`
- `Ghost-FTP-0.2.0-Linux-multiarch.zip`

Example for an amd64 Debian-family system:

```text
sudo apt install ./Ghost-FTP-0.2.0-Linux-amd64.deb
```

The package installs the `ghostftp` executable and desktop/package metadata. With a local X11/XWayland-compatible display it starts the graphical frontend by default; headless environments can use the hardened terminal fallback.

Windows and Linux use the same typed `internal/api.Engine`, connection model, transfer manager, profile/settings model and security validation. Native presentation can differ where required by the operating system, but the application operations and safety boundaries remain aligned.

Linux 0.2.0 also exposes runtime selection from the same 24-language registry used by Windows, with English as the default/fallback language.

### Linux runtime prerequisites

The DEB metadata is the source of truth for Linux package dependencies. The protocol implementation uses system-provided:

- `curl` for FTP/FTPS;
- OpenSSH `ssh` and `sftp` for SFTP.

These are explicit operating-system prerequisites, not hidden bundled libraries or Go module dependencies.

## First connection

For Windows and Linux:

1. choose FTP, explicit FTPS, implicit FTPS or SFTP as appropriate;
2. enter the exact host, port and username supplied by the server provider;
3. provide the password or SFTP private-key credentials required by that server;
4. for SFTP, verify the server host-key fingerprint before first trust;
5. connect;
6. browse the local and remote panes;
7. upload/download only after the connection is confirmed.

Ghost FTP does not send credentials to a Ghost FTP account service or analytics endpoint. Connection credentials are used for the user-selected server connection.

### SFTP authentication

SFTP supports:

- password authentication;
- private-key authentication;
- a private key protected by a passphrase.

A private key is not mandatory when the server permits password authentication. Host-key verification remains active regardless of the authentication method.

### FTPS

Use FTPS when supported by the server. Certificate validation remains enabled. Ghost FTP does not add a blanket certificate/revocation bypass merely to make a failing server appear connected.

### FTP

Plain FTP is unencrypted. It remains available for compatibility with servers that provide no secure alternative, but FTPS or SFTP should be preferred whenever available.

## Settings and user data

Ghost FTP stores settings and saved profiles in the platform-specific user application-data location. State writes use guarded replacement and conservative filesystem permissions where supported.

Windows saved profile secrets use DPAPI-backed protection. Linux persisted secret envelopes use the maintained authenticated-encryption/profile security implementation and private per-user state/key permissions.

The same validated Settings model controls operational options such as:

- language;
- transfer parallelism;
- conflict behavior;
- retry count;
- retry delay;
- operation timeout;
- destructive-action confirmation.

Settings that are displayed by the UI are wired to the shared engine/configuration model rather than being presentation-only toggles.

## Release verification

Before deploying a package in a managed environment:

1. download it from the matching `ghostftp-vX.Y.Z` GitHub Release;
2. verify its SHA-256 hash against `SHA256.txt`;
3. review `BUILD-METADATA.txt` for the expected commit, version, channel and platform;
4. review `RELEASE-NOTES.txt` for compatibility and behavior changes;
5. for any `0.x.y` version, verify the GitHub Release is marked as a Beta/prerelease.

A complete Ghost FTP desktop release contains **9 platform artifacts** and **12 public files** after release notes, build metadata and checksums are included.

The production release workflow performs immediate and delayed GitHub Release readback, verifies exact asset names/sizes, compares the published checksum manifest with the locally assembled manifest and verifies the intended Beta/stable channel.

See [Release verification](RELEASE-VERIFICATION.md), [GitHub Releases](GITHUB-RELEASES.md), [Platform parity](PLATFORM-PARITY.md), [Security](SECURITY.md) and [Versioning](VERSIONING.md).
