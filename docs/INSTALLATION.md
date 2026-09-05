# Installation

Ghost FTP 2.x is distributed as a desktop application for **Windows** and **Linux**.

Android, iOS and macOS application packages are not part of the active 2.x distribution. Historical 1.x releases remain available through immutable release/tag history.

## Windows

Ghost FTP publishes both Setup and portable builds for x64 and x86 Windows systems.

### Setup packages

- `Ghost-FTP-X.Y.Z-Setup-x64.exe`
- `Ghost-FTP-X.Y.Z-Setup-x86.exe`
- `Ghost-FTP-X.Y.Z-Setup-x32.exe` — compatibility alias of the x86 setup artifact

The Setup flow is localized through the same canonical language registry used by the application. English is the default/fallback. The installer validates its embedded payload before installation and registers the application using the stable `GhostFTP` technical identity while presenting **Ghost FTP** as the public product name.

### Portable packages

- `Ghost-FTP-X.Y.Z-Portable-x64.exe`
- `Ghost-FTP-X.Y.Z-Portable-x86.exe`

Portable builds do not require a traditional installation. Place the executable in a user-writable folder and run it directly.

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

Install a matching package using the distribution package manager, for example:

```text
sudo apt install ./Ghost-FTP-X.Y.Z-Linux-amd64.deb
```

The package installs the `ghostftp` executable and Linux desktop/package metadata. The current Linux frontend is terminal-based but uses the same transfer/security engine as Windows.

### Linux runtime prerequisites

The DEB metadata is the source of truth for package dependencies. The current protocol implementation requires system-provided:

- `curl` for FTP/FTPS;
- OpenSSH `ssh` and `sftp` for SFTP.

These are system packages, not hidden bundled libraries.

## First connection

For every platform:

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

## Release verification

Before deploying an installer/package in a managed environment:

1. download the release artifact from the matching `ghostftp-vX.Y.Z` release;
2. verify its SHA-256 hash against `SHA256.txt`;
3. review `BUILD-METADATA.txt` for the expected commit/version/platform;
4. review `RELEASE-NOTES.txt` for behavior and compatibility changes.

See [Release verification](RELEASE-VERIFICATION.md).

## Web companion

The repository also contains a Web companion for shared-hosting deployment. It is not installed as a Windows/Linux desktop package and is not counted in the 2.x desktop application artifact contract. See [Shared hosting](SHARED-HOSTING.md).
