<p align="center">
  <img src="docs/slike/byftp-zaglavlje.png" alt="ByFTP — FTP, FTPS and SFTP client" width="900">
</p>

<p align="center">
  <strong>FTP without the friction. More control over your hosting.</strong><br>
  ByFTP is Brendigo's FTP, FTPS and SFTP client for fast, private and secure website file management — with no advertising, mandatory cloud account or application telemetry.
</p>

<p align="center">
  <a href="https://github.com/bren-wp/by-ftp/releases"><strong>Download ByFTP</strong></a> ·
  <a href="https://github.com/users/bren-wp/packages?repo_name=by-ftp"><strong>Packages</strong></a> ·
  <a href="docs/SHARED-HOSTING.md"><strong>Shared hosting</strong></a> ·
  <a href="docs/INSTALACIJA.md"><strong>Installation</strong></a> ·
  <a href="docs/PODRSKA.md"><strong>Support</strong></a> ·
  <a href="docs/SIGURNOST.md"><strong>Security</strong></a>
</p>

<p align="center">
  <a href="../../actions/workflows/ci.yml"><img alt="Checks" src="../../actions/workflows/ci.yml/badge.svg"></a>
</p>

# ByFTP

ByFTP is designed for the everyday hosting workflow: connect, find `public_html`, manage files, transfer folders and keep working without unnecessary accounts, telemetry or browser-based middleware. Windows uses a native two-panel Win32 interface. Linux and macOS use a terminal interface over the same Go engine.

**Current release: 1.0.11**

## Languages

English is the canonical and default product language. ByFTP 1.0.11 also includes runtime localization for:

- Croatian — Hrvatski (`hr`)
- German — Deutsch (`de`)
- French — Français (`fr`)
- Spanish — Español (`es`)
- Turkish — Türkçe (`tr`)
- Greek — Ελληνικά (`el`)
- Portuguese — Português (`pt`)
- Simplified Chinese — 简体中文 (`zh`)
- Russian — Русский (`ru`)
- Hindi — हिन्दी (`hi`)
- Japanese — 日本語 (`ja`)

Windows can switch language from the main window without restarting. The selected language is stored in the existing local settings file. Linux/macOS use the same localization catalog and settings model. Unknown or legacy-empty locale values fall back safely to English.

## Features

- FTP, explicit FTPS, implicit FTPS and SFTP in one client.
- Native two-panel Windows file management with multi-selection.
- Upload/download of files and directory trees.
- Transfer queue with pause, resume, cancel and retry.
- Local and remote folder creation, rename and delete operations.
- CHMOD where the server supports it.
- Saved Windows profiles with DPAPI-protected secrets.
- SFTP SHA-256 host-key fingerprint verification and pinning.
- Shared-hosting-friendly FTP path semantics and MLSD → LIST fallback.
- Safer overwrite flow with staging, revalidation, backup and rollback.
- No application telemetry, analytics SDK, advertising or required Brendigo cloud account.

## What changes in 1.0.12

1.0.12 is an English-first production cleanup and localization release. It centralizes user-facing text instead of maintaining duplicated language-specific strings throughout the UI, adds twelve tested runtime languages, persists the selected locale, and adapts the Windows layout for longer translated labels.

The localization layer validates catalog completeness and formatting placeholders in tests. Known protocol/tool errors are mapped to one semantic user-error layer and translated there, so changing language does not change connection, security or transfer logic.

The release also continues the 1.0.11 UI/stability work: responsive Windows sizing, selection preservation during refresh, connection-generation guards against stale callbacks, honest partial-success reporting for remote batch operations and context-aware transfer controls.

## Shared hosting

A typical shared-hosting account needs a server name, username, password, protocol and port. The username may be in an email-like form such as `account@example.com`.

For FTP/FTPS, logical `/public_html` stays within the login/home namespace exposed by the hosting server. ByFTP first attempts machine-readable `MLSD`; when a legacy or non-standard server cannot provide a usable MLSD listing, the client falls back to `LIST` and remembers that choice for the session.

See [Shared hosting](docs/SHARED-HOSTING.md) for practical examples and troubleshooting.

## Platform support

| Platform | Distribution | Architectures | Interface |
|---|---|---|---|
| Windows 10/11 | Setup EXE, Portable EXE, ZIP | x64, x86 | Native two-panel GUI |
| Linux | DEB | amd64, arm64, i386 | Terminal |
| macOS | Universal PKG | Intel + Apple Silicon | Terminal |

### Authentication

| Method | Windows | Linux / macOS |
|---|---|---|
| FTP/FTPS password | Yes | Yes |
| SFTP private key without passphrase | Yes | Yes |
| SFTP password | Yes | No — fail closed |
| SFTP private key with passphrase | Yes | No — fail closed |
| SFTP host-key fingerprint verification | Yes | Yes |

Linux/macOS deliberately do not pass SFTP passwords or encrypted-key passphrases through unsafe command-line arguments or ordinary environment variables. Those authentication modes remain blocked until a tested secure Unix credential broker exists.

## Downloads

Official builds are published through GitHub Releases. `VERSION` is the canonical version source for runtime binaries, release artifacts and the Windows GitHub Package.

### Windows

- `ByFTP-<version>-Setup-x64.exe` — recommended for most Windows systems
- `ByFTP-<version>-Portable-x64.exe` — no installation required
- `ByFTP-<version>-Windows-x64.zip` — verified Windows bundle
- corresponding x86 packages are produced for supported 32-bit systems

### Linux

- `ByFTP-<version>-Linux-amd64.deb`
- `ByFTP-<version>-Linux-arm64.deb`
- `ByFTP-<version>-Linux-i386.deb`

### macOS

- `ByFTP-<version>-macOS-Universal.pkg`

Every release includes SHA-256 metadata and release/build metadata used by the publication verification flow.

## Transfer safety

ByFTP is designed so that an uncertain cleanup or commit state does not look like success. Current protections include:

- cryptographically random staging and temporary names;
- local symlink/junction/reparse-point validation;
- no-replace activation where the platform provides a suitable primitive;
- private byte-for-byte local upload snapshots;
- SHA-256 validation of upload snapshots before and after network reads;
- remote staging type checks when the server exposes the staging entry;
- fresh destination revalidation immediately before remote commit/rename;
- fail-closed reporting when remote staging/rollback cleanup cannot be confirmed;
- retry binding to connection identity and transfer generation;
- bounded child-process output;
- timeout/cancel propagation with process-tree cleanup for external curl/OpenSSH processes;
- FTPS certificate-revocation protection without a global revocation bypass;
- SFTP SHA-256 fingerprint pinning and private `known_hosts` lifecycle.

The safer local upload snapshot intentionally uses additional temporary disk space approximately equal to the uploaded file size.

## Privacy

ByFTP has no advertising, analytics SDK, external crash-reporting service, mandatory cloud account or fixed Brendigo API receiving user activity. Windows saved profile secrets use DPAPI. Active credentials are not placed in network-tool command-line arguments. Production builds require Go telemetry to be disabled.

See [Privacy](docs/PRIVATNOST.md) and [Security](docs/SIGURNOST.md).

## Verify SHA-256

Windows PowerShell:

```powershell
Get-FileHash .\ByFTP-<version>-Setup-x64.exe -Algorithm SHA256
```

Linux:

```bash
sha256sum ByFTP-<version>-Linux-amd64.deb
```

macOS:

```bash
shasum -a 256 ByFTP-<version>-macOS-Universal.pkg
```

## Documentation

The repository is being standardized on English as the canonical documentation language. Existing document paths remain linked during the 1.0.12 migration so external links are not broken before their replacements are committed.

- [Documentation center](docs/README.md)
- [Installation and first connection](docs/INSTALACIJA.md)
- [Shared hosting](docs/SHARED-HOSTING.md)
- [Support and troubleshooting](docs/PODRSKA.md)
- [Security](docs/SIGURNOST.md)
- [Privacy](docs/PRIVATNOST.md)
- [Testing and quality](docs/TESTIRANJE.md)
- [Release verification](docs/PROVJERA-IZDANJA.md)
- [Architecture](docs/ARHITEKTURA.md)
- [Contribution policy](docs/DOPRINOS.md)

## Important limitations

No desktop FTP/SFTP client can guarantee compatibility with every non-standard server or hosting policy. A provider may restrict writes, rename, CHMOD, connection count, TLS/SSH algorithms, passive data ports or particular directories.

Windows binaries require a real Brendigo Authenticode certificate for Verified Publisher status. macOS Developer ID signing/notarization requires real Apple credentials; the project does not simulate those trust states.

The security hardening reduces and fail-closed handles known race/lifecycle classes, but it does not claim that no unknown bug can exist on every OS/server/network combination. Production releases therefore remain gated by regression tests, the race detector, `go vet`, security/privacy audits and platform builds.

---

<p align="center"><strong>ByFTP — connect to your hosting, find your files, and keep working.</strong></p>
