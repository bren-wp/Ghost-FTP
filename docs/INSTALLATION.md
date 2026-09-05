# Installation

Ghost FTP is distributed as a desktop application for **Windows** and **Linux**.

The active desktop development baseline is **0.1.0 Beta**. Every `0.x.y` package remains a Beta build until the product intentionally reaches the first stable **1.0.0** release.

Android, iOS and macOS application packages are not part of the active distribution. Historical releases remain available through immutable release/tag history and are not deleted by the current version reset.

## Windows

Ghost FTP publishes both Setup and Portable builds for x64 and x86 Windows systems.

Setup and Portable are two packaging forms of the same release. They always use the same canonical version from the root `VERSION` file.

### Setup packages

Canonical names:

- `Ghost-FTP-X.Y.Z-Setup-x64.exe`
- `Ghost-FTP-X.Y.Z-Setup-x86.exe`
- `Ghost-FTP-X.Y.Z-Setup-x32.exe` — compatibility alias of the x86 Setup artifact

Current Beta examples:

- `Ghost-FTP-0.1.0-Setup-x64.exe`
- `Ghost-FTP-0.1.0-Setup-x86.exe`

First stable examples:

- `Ghost-FTP-1.0.0-Setup-x64.exe`
- `Ghost-FTP-1.0.0-Setup-x86.exe`

The Setup flow is localized through the same canonical language registry used by the application. English is the default/fallback. The installer validates its embedded payload before installation and registers the application using the stable `GhostFTP` technical identity while presenting **Ghost FTP** as the public product name.

The first language/option surface is a dependency-free native Win32 dialog with a consistent Ghost FTP shell: centered placement, clear header/instruction hierarchy, larger action targets, a default primary action and best-effort modern Windows dark-title-bar/rounded-corner hints. The same visual helper is used by native edit prompts so Setup and the application do not drift into unrelated dialog styles.

These visual changes do **not** weaken the installer boundary. Payload digest verification, staging, no-redirect path checks, rollback-aware activation, registry/application-path changes and final readback remain separate from presentation code. Normal confirmation/completion/error steps continue to use native Windows Task Dialog behavior where available.

### Portable packages

Canonical names:

- `Ghost-FTP-X.Y.Z-Portable-x64.exe`
- `Ghost-FTP-X.Y.Z-Portable-x86.exe`

Current Beta examples:

- `Ghost-FTP-0.1.0-Portable-x64.exe`
- `Ghost-FTP-0.1.0-Portable-x86.exe`

First stable examples:

- `Ghost-FTP-1.0.0-Portable-x64.exe`
- `Ghost-FTP-1.0.0-Portable-x86.exe`

Portable builds do not require a traditional installation. Place the executable in a user-writable folder and run it directly.

### Beta identification

The package filename and Windows file/product metadata keep the numeric semantic version (`0.1.0`, `0.2.0`, and so on). User-facing product/version surfaces may append **Beta** for any `0.x.y` build.

GitHub Releases for `0.x.y` are published as prereleases. This prevents a Beta package from being represented as the first stable release merely because the binary itself is production-build quality.

### Windows runtime prerequisites

Ghost FTP does not download protocol components in the background. The current transport implementation expects suitable operating-system `curl` and OpenSSH components. On supported Windows installations these are normally present as system components.

If a required transport executable is unavailable, the connection must fail with an actionable error rather than silently fetching third-party software.

## Linux

Ghost FTP publishes Debian packages for:

- amd64
- arm64
- i386

Release file names:

- `Ghost-FTP-X.Y.Z-Linux-amd64.deb`
- `Ghost-FTP-X.Y.Z-Linux-arm64.deb`
- `Ghost-FTP-X.Y.Z-Linux-i386.deb`
- `Ghost-FTP-X.Y.Z-Linux-multiarch.zip`

For the current Beta baseline, replace `X.Y.Z` with `0.1.0`.

Install a matching package using the distribution package manager, for example:

```text
sudo apt install ./Ghost-FTP-0.1.0-Linux-amd64.deb
```

The package installs the `ghostftp` executable and Linux desktop/package metadata. The current Linux frontend is terminal-based but uses the same transfer/security engine as Windows.

Linux exposes remote and local working directories, local file-management commands, single-file transfer commands, bounded directory/tree transfer commands, transfer queue controls, validated settings and saved-profile metadata operations. See [Windows and Linux platform parity](PLATFORM-PARITY.md) and [Linux packaging/usage](../linux/README.md).

### Linux runtime prerequisites

The DEB metadata is the source of truth for package dependencies. The current protocol implementation requires system-provided:

- `curl` for FTP/FTPS;
- OpenSSH `ssh` and `sftp` for SFTP.

These are system packages, not hidden bundled libraries.

## First connection

For every maintained desktop platform:

1. choose the protocol;
2. enter the exact server host, port and username;
3. provide password or SFTP key credentials as appropriate;
4. verify the SFTP host fingerprint when the server is first trusted;
5. begin browsing/transferring only after the connection succeeds.

### SFTP authentication

SFTP supports:

- password authentication;
- private-key authentication;
- private key protected by a passphrase.

A private key is not mandatory when the server supports password authentication.

### FTPS

Use explicit FTPS when supported by the server. Certificate validation remains enabled.

### FTP

Plain FTP is unencrypted. Use it only when the server does not provide a secure alternative.

## Settings and user data

Ghost FTP stores settings/profiles in the platform-specific user application-data location. Files are created with conservative permissions where supported and state writes use guarded atomic replacement.

Saved secrets follow platform security handling. Windows uses DPAPI-backed protection; Linux runtime secrets are protected within the process and saved profile handling is governed by the platform/profile crypto implementation.

The Linux `profile-save` command intentionally saves public connection metadata/current paths without reconstructing a password/passphrase that has already been cleared after authentication. This keeps profile creation from turning into an implicit secret-persistence side effect.

## Release verification

Before deploying an installer/package in a managed environment:

1. download the release artifact from the matching `ghostftp-vX.Y.Z` release;
2. verify its SHA-256 hash against `SHA256.txt`;
3. review `BUILD-METADATA.txt` for the expected commit/version/release channel/platform;
4. review `RELEASE-NOTES.txt` for behavior and compatibility changes;
5. for a `0.x.y` build, verify that the GitHub Release is marked as a prerelease/Beta.

The production release workflow performs its own immediate and delayed GitHub Release readback, verifies exact asset names/sizes, compares the published `SHA256.txt` with the locally assembled manifest and verifies the expected Beta/stable channel before publication is allowed to report success.

See [Release verification](RELEASE-VERIFICATION.md) and [Versioning policy](VERSIONING.md).

## Web companion

The repository also contains a Web companion for shared-hosting deployment. It is not installed as a Windows/Linux desktop package and is not counted in the desktop application artifact contract. See [Shared hosting](SHARED-HOSTING.md).
