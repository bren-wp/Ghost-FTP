<p align="center">
  <img src="build/icon.png" alt="ByFTP" width="128" height="128">
</p>

<h1 align="center">ByFTP</h1>

<p align="center">
  <strong>Fast, private and native FTP / FTPS / SFTP for Windows.</strong><br>
  A focused desktop file-transfer client by Brendigo — built for developers, agencies, hosting administrators and anyone who wants direct server access without a browser shell, telemetry or cloud account.
</p>

<p align="center">
  <a href="https://github.com/bren-wp/by-ftp/releases"><strong>Releases</strong></a> ·
  <a href="https://github.com/users/bren-wp/packages?repo_name=by-ftp"><strong>Packages</strong></a> ·
  <a href="LICENSE"><strong>License</strong></a> ·
  <a href="PRIVACY.md"><strong>Privacy</strong></a> ·
  <a href="SECURITY.md"><strong>Security</strong></a>
</p>

<p align="center">
  <a href="../../actions/workflows/ci.yml"><img alt="CI" src="../../actions/workflows/ci.yml/badge.svg"></a>
</p>

## Transfer files. Keep control.

ByFTP is a native Windows x64 client for **FTP, FTPS and SFTP**. It combines a familiar two-panel file manager with a hardened transfer engine, multi-file workflows and a privacy-first architecture.

There is no embedded browser, no localhost web dashboard, no analytics SDK, no advertising layer and no account required. ByFTP connects to the server **you choose** and keeps application data local to your Windows profile.

**Current release: 2.12.0**

## Download

### [Releases](https://github.com/bren-wp/by-ftp/releases)

**Recommended for Windows users.** Official ByFTP release downloads are published on the GitHub Releases page. Release downloads can include:

- Portable x64 EXE
- Setup x64 EXE
- Uninstaller
- SHA-256 checksums
- build/security verification report

### [Packages](https://github.com/users/bren-wp/packages?repo_name=by-ftp)

GitHub Packages is reserved for packaged ByFTP distributions and related official artifacts where that distribution format is appropriate.

> Only unmodified builds published by Brendigo through an official ByFTP/Brendigo channel are official releases.

### Why ByFTP?

- **Native Windows experience** — dark Win32 desktop UI with Fluent-style system icons and proper DPI scaling.
- **FTP + FTPS + SFTP in one client** — including explicit/implicit FTPS and SFTP key authentication.
- **Built for real file work** — multi-select, whole-folder transfers, batch operations, permissions and a persistent transfer queue.
- **Safer transfer behavior** — transactional staging, rollback, path revalidation, symlink/junction protection and cross-server retry blocking.
- **Privacy by design** — no telemetry, no analytics, no automatic external API calls and no persistent runtime activity log.
- **Local credential protection** — saved profiles are protected with Windows DPAPI and sensitive values are kept out of process command lines.
- **Small and focused** — no Electron/WebView application shell and no external Go runtime dependencies.

## Features

### Connections

- FTP
- FTPS Explicit
- FTPS Implicit
- SFTP
- password authentication
- private-key authentication with passphrase support
- SFTP host-key fingerprint confirmation and pinning
- configurable 5–60 second connection timeout
- saved connection profiles protected locally with Windows DPAPI

### File manager

- two-panel local/server workflow
- Ctrl/Shift multi-select
- double-click navigation and quick transfers
- upload/download individual files or complete directory trees
- create folders
- rename
- delete
- refresh/navigation controls
- remote permissions (CHMOD), including batch selection
- Windows file-type and folder icons
- protection against path traversal, reserved Windows names, symlinks, junctions and reparse-point escapes

### Transfer queue

- 1–8 parallel transfers
- pause / resume
- cancel selected or multiple jobs
- retry selected failed/cancelled jobs
- optional transient-only automatic retry
- configurable retry delay
- **Skip existing files** mode
- cross-server retry protection
- runtime local-root revalidation before every attempt
- worker panic containment so one unexpected job failure does not take down the whole application

## Privacy-first architecture

ByFTP is deliberately designed without telemetry or cloud control-plane dependencies.

It does **not** include:

- analytics or usage tracking
- advertising SDKs
- third-party crash reporting
- background account synchronization
- automatic update API calls
- browser/localhost control servers
- persistent runtime activity/error logging

Normal ByFTP network traffic is intended for the **FTP/FTPS/SFTP destination selected by the user**. Windows itself can still perform operating-system services such as DNS resolution, antivirus/EDR inspection or firewall processing outside ByFTP's control.

Saved connection profiles are protected locally with Windows DPAPI. SFTP trust material is session-scoped, sensitive SSH metadata is minimized on process command lines and ByFTP blocks inherited proxy/SSH helper paths that could redirect a connection unexpectedly.

Read the complete model in [PRIVACY.md](PRIVACY.md) and [SECURITY.md](SECURITY.md).

## Security highlights

- SFTP host-key pinning tied to the verified key/algorithm
- System32-only Windows curl/OpenSSH discovery
- proxy, SSH agent, ProxyJump/ProxyCommand and external helper inheritance blocked
- password/passphrase values excluded from command-line arguments
- no early decryption of saved credentials in the connection manager
- safe local-child validation for server-controlled file names
- junction/symlink/reparse-point traversal protections
- transactional upload/download staging and rollback
- cryptographically random internal staging names
- secure ByFTP-owned directory creation
- connection-generation isolation for queued transfer batches
- cross-server transfer retry blocking
- installer payload size + SHA-256 validation
- installer upgrade rollback for binaries and Registry state
- guarded uninstaller path validation

## Designed for

**Web developers & agencies** — quickly upload releases, edit hosting files and manage multiple client servers.

**Hosting & server administrators** — work across FTP, FTPS and SFTP from one native Windows interface.

**Privacy-conscious teams** — use a transfer client that does not require a vendor account, analytics pipeline or cloud dashboard.

**Portable workflows** — run the portable build when you do not want a traditional installation.

## Requirements

### Running ByFTP

- Windows 10 or Windows 11 x64
- Windows system `curl.exe` for FTP/FTPS
- Windows OpenSSH Client for SFTP

### Building ByFTP

Production builds require:

- Go **1.26.5+**
- Python 3
- Windows x64

The production build intentionally avoids dependency downloads:

```text
GOTOOLCHAIN=local
GOPROXY=off
GOSUMDB=off
```

The project currently has **no external Go modules**.

## Build from source

The source is published for transparency, security review and authorized development. **ByFTP is not open-source software.** Modification, redistribution, rebranding, derivative works and reuse of the source code require prior written permission from Brendigo. See [LICENSE](LICENSE).

For authorized builds, from PowerShell on Windows:

```powershell
.\BUILD-WINDOWS.ps1
```

or:

```cmd
BUILD-WINDOWS.cmd
```

The production pipeline runs privacy checks, unit tests, `go vet`, Windows builds, PE resource injection, mitigation verification and SHA-256 generation. Output is written to `dist/`.

## Quality gates

Core checks:

```powershell
go test ./...
go vet ./...
python scripts/audit_privacy.py
```

GitHub Actions additionally runs race tests and the Windows production build path.

See [TESTING.md](TESTING.md) and [RELEASE-CHECKLIST.md](RELEASE-CHECKLIST.md).

## Repository structure

```text
cmd/
  byftp/        desktop application + controlled SFTP AskPass mode
  installer/    per-user Windows installer
  uninstaller/  guarded Windows uninstaller
internal/
  api/          typed in-process application engine
  config/       profiles, settings and local protection
  desktop/      native Win32 UI
  localfs/      local filesystem operations
  remote/       FTP / FTPS / SFTP adapters
  security/     validation and Windows security helpers
  transfer/     queue, retry and worker lifecycle
scripts/        privacy, build, payload and PE verification tooling
build/          ByFTP icon resources
.github/        CI, CODEOWNERS and issue templates
```

## License

**Copyright © 2026 Brendigo. All rights reserved.**

ByFTP is **proprietary/source-available software, not open-source software**. The source can be viewed for transparency, security review and evaluation, but the license does not grant a general right to modify, redistribute, rebrand, sell, sublicense, create derivative works from, or reuse the code in another project.

Read the complete [Brendigo ByFTP Proprietary Source License](LICENSE).

Because this repository is public, GitHub's own Terms of Service may allow GitHub users to view and fork the repository through GitHub's platform functionality. That platform permission does not grant broader modification, redistribution, commercialization or rebranding rights under the ByFTP license.

## Authorized contributions

Issues and feature suggestions are welcome. Source modifications or pull-request contributions should be made only when expressly requested or authorized by Brendigo and remain subject to [LICENSE](LICENSE) and [CONTRIBUTING.md](CONTRIBUTING.md).

**Never** include real passwords, private keys, production hostnames, account names or customer server information in issues, screenshots, test fixtures or pull requests.

## Documentation

- [Architecture](ARCHITECTURE.md)
- [Security](SECURITY.md)
- [Privacy](PRIVACY.md)
- [Testing](TESTING.md)
- [Release checklist](RELEASE-CHECKLIST.md)
- [Signing](SIGNING.md)
- [Changelog](CHANGELOG.md)
- [Support](SUPPORT.md)
- [Contributing](CONTRIBUTING.md)
- [Third-party notices](THIRD-PARTY-NOTICES.md)

## Security reports

Please follow [SECURITY.md](SECURITY.md). Do not publish credentials or sensitive server information in a public GitHub issue.

---

<p align="center">
  <a href="https://github.com/bren-wp/by-ftp/releases"><strong>Download ByFTP</strong></a><br><br>
  <strong>ByFTP</strong><br>
  Native file transfer for Windows · Built by Brendigo
</p>
