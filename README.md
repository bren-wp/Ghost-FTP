# Ghost FTP

**Ghost FTP** is a privacy-first FTP, FTPS and SFTP desktop client focused on dependable transfers, conservative security defaults and a clean professional workflow.

Current Ghost FTP version: **2.0.0**

The 2.x product line is intentionally focused on two application platforms:

- **Windows** — native Win32 desktop UI, x64 and x86, with setup and portable packages.
- **Linux** — the same shared transfer/security engine, packaged for amd64, arm64 and i386, with a hardened terminal interface.

Android, iOS and macOS application targets were retired from the active source tree for 2.0. Their historical source and releases remain available through immutable Git history and published 1.x tags.

The repository also contains the existing **Ghost FTP Web companion** source. It is maintained separately from the Windows/Linux desktop release contract and is not counted as an application platform artifact in the 2.x desktop release.

## Product principles

Ghost FTP is built around a small set of non-negotiable rules:

- no application telemetry, analytics, advertising or crash-reporting SDKs;
- no fixed analytics/update endpoint in the desktop runtime;
- no generic localhost API or browser IPC between the UI and transfer engine;
- fail-closed connection, path, credential and host-key validation;
- explicit SFTP host-key trust;
- bounded transfer retries and timeouts;
- atomic state writes and conservative overwrite recovery;
- reproducible Windows/Linux release artifacts with SHA-256 verification;
- English as the canonical/default language with more than 20 maintained languages;
- no third-party Go modules in the desktop/core module.

## Protocols

### SFTP

SFTP uses OpenSSH semantics and supports:

- password authentication;
- private-key authentication;
- private keys protected with a passphrase;
- explicit host-key fingerprint trust;
- saved fingerprint binding to the expected endpoint;
- proxy/jump/agent forwarding disabled by the application-created SSH configuration;
- protected runtime credential handoff without writing an AskPass secret file.

### FTPS

Explicit FTPS is supported with certificate validation enabled. Ghost FTP does not silently disable TLS certificate revocation checks.

### FTP

Plain FTP is supported for compatibility with servers that require it, but it is unencrypted and should only be used when the server cannot provide FTPS or SFTP.

Implicit FTPS is not part of the supported desktop protocol contract.

## Windows experience

The Windows edition is the reference graphical experience. It includes:

- premium dark graphite/navy interface;
- local and remote file panels;
- high-DPI-aware layout and typography;
- saved connection profiles;
- protocol/host/port/user/password controls;
- SFTP private-key and passphrase controls;
- upload and download actions;
- remote create, rename, delete and permissions operations;
- local create, rename and delete operations;
- transfer queue with pause, resume, cancel, retry and clear-finished actions;
- conflict policies: `skip`, `replace`, and `replace_backup`;
- automatic retry, retry-delay, parallelism and connection-timeout settings;
- live language switching;
- localized Windows Setup flow;
- x64 and x86 setup/portable packages.

The 2.0 visual refresh uses a deeper graphite/navy surface system, stronger action hierarchy and refined owner-drawn buttons without introducing a GUI framework dependency.

## Linux experience

Linux uses the same `internal/api`, remote/session, transfer, settings, profile, localization and security layers as Windows. The 2.0 terminal frontend closes important historical parity gaps:

- SFTP now accepts **password authentication** when no private key is supplied;
- SFTP accepts a **private key plus optional passphrase** instead of rejecting passphrases;
- FTP/FTPS password authentication remains available;
- transfer jobs can be listed, paused, resumed, cancelled, retried and cleared;
- core transfer settings can be inspected and changed through the shared validated settings store;
- saved profiles can be inspected;
- remote list, navigation, create, rename, delete and chmod operations use the same engine boundaries;
- upload/download jobs use the same transfer scheduler and conflict policy as Windows;
- runtime language can be changed using the canonical localization registry.

The Windows and Linux editions now share the **functional core contract**, but the presentation layer is intentionally different: Windows is a native Win32 GUI and Linux is a terminal UI. A pixel-identical cross-platform GUI would require introducing or maintaining a GUI/runtime dependency, which the current dependency policy deliberately avoids.

## Languages

Ghost FTP is English-first and currently exposes **24 languages**:

1. English (`en`) — canonical source and default fallback
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

Regional aliases such as Norwegian `nb`/`nn` and Simplified Chinese aliases normalize to the canonical registry. Missing/invalid locale state falls back safely to English.

CI validates the language registry, catalog parity, format verbs, translation coverage, Windows live localization, Windows Setup localization and Linux runtime localization.

## Transfer options

The canonical settings model contains:

| Option | Supported range / values | Safe default |
| --- | --- | --- |
| Parallel transfers | 1–8 | 2 |
| Conflict policy | `skip`, `replace`, `replace_backup` | `replace_backup` |
| Automatic retries | 0–3 | 0 |
| Retry delay | 1–30 seconds | 3 seconds |
| Connection timeout | 5–60 seconds | 15 seconds |
| Confirm delete | on/off | on |
| Language | 24 registered locales | English |

`replace_backup` is the conservative overwrite default. Persisted unknown conflict states migrate back to this policy rather than silently weakening recovery behavior.

## Privacy

Ghost FTP does not contain application analytics, advertising, tracking pixels, crash-reporting SDKs or an application-controlled telemetry service.

The desktop runtime deliberately blocks fixed HTTP(S) URLs in `cmd/` and `internal/`, and CI runs a privacy audit that rejects known telemetry/vendor markers and unexpected network-client imports.

The application communicates with the server address entered by the user for the purpose of FTP/FTPS/SFTP file transfer. Build and release workflows use GitHub infrastructure only for source/build/release automation; that is separate from the installed application's runtime behavior.

See [docs/PRIVACY.md](docs/PRIVACY.md).

## Security model

Important enforced boundaries include:

- credentials never pass through a generic JSON dispatcher;
- Windows saved profile secrets use DPAPI-backed protection;
- non-Windows runtime secrets use short-lived protected in-process storage;
- SFTP AskPass credentials are not written to disk;
- SFTP host keys are explicitly trusted and endpoint-bound;
- proxy, jump-host, agent-forwarding and ambient SSH state are disabled for application sessions;
- FTP/FTPS proxy environment variables are sanitized;
- download staging paths reject symlink/reparse-point substitution;
- local delete/rename paths use no-follow and no-replace protections;
- filesystem-root recursive deletion is blocked;
- remote session shutdown is bounded and race-protected;
- profile credentials cannot silently cross account/endpoint/private-key identity boundaries;
- settings/profile state uses guarded atomic writes.

See [docs/SECURITY.md](docs/SECURITY.md) for the maintained threat and invariant documentation.

## Dependencies

The desktop/core Go module intentionally has **zero external Go modules**: no `require`, `replace`, vendored module tree or `go.sum` dependency graph is accepted by CI.

That does **not** mean the current transport implementation has zero operating-system prerequisites. The desktop transport layer currently uses OS-provided tools:

- FTP/FTPS: `curl`
- SFTP: OpenSSH `ssh` and `sftp`

Windows normally provides suitable system components on supported installations; Linux packages declare the required runtime packages. Ghost FTP does not bundle those projects as hidden third-party libraries.

This distinction is documented and audited so releases do not make a misleading “zero runtime dependencies” claim. See [docs/DEPENDENCIES.md](docs/DEPENDENCIES.md).

## Build from source

Required desktop toolchain:

- Go **1.27.1**
- Python 3 for repository/release verification scripts

Before a production build, disable Go telemetry:

```text
go telemetry off
```

### Windows

Use:

```text
.\BUILD-WINDOWS.ps1
```

The canonical build creates setup and portable x64/x86 artifacts in `dist/` and injects the root `VERSION` into the binaries.

### Linux

Use:

```text
bash linux/BUILD.sh
```

The canonical Linux build creates DEB packages for amd64, arm64 and i386.

See [docs/INSTALLATION.md](docs/INSTALLATION.md) and [docs/TESTING.md](docs/TESTING.md).

## Releases

Ghost FTP follows semantic versioning. Removing application platforms is a breaking public contract change, therefore the Windows/Linux consolidation starts at **2.0.0** instead of being published as another 1.x patch.

Release tags use:

```text
ghostftp-vX.Y.Z
```

The 2.x release contract contains **9 platform artifacts**:

- Windows Setup x64
- Windows Setup x86
- Windows Setup x32 compatibility alias of x86
- Windows Portable x64
- Windows Portable x86
- Linux amd64 DEB
- Linux arm64 DEB
- Linux i386 DEB
- Linux multiarch ZIP

Each release also contains:

- `RELEASE-NOTES.txt`
- `BUILD-METADATA.txt`
- `SHA256.txt`

That produces **12 public files** per complete desktop release.

Published tags/releases are treated as immutable historical provenance. The release workflow refuses to move an existing release tag to another commit.

See [CHANGELOG.md](CHANGELOG.md), [docs/RELEASE-HISTORY.md](docs/RELEASE-HISTORY.md), [docs/GITHUB-RELEASES.md](docs/GITHUB-RELEASES.md) and [docs/RELEASE-VERIFICATION.md](docs/RELEASE-VERIFICATION.md).

## Repository quality gates

CI runs fail-closed checks for:

- Go formatting, race tests and vet;
- repository path/case/generated-file integrity;
- Windows/Linux-only application platform contract;
- version/release artifact contract;
- no external Go module dependency drift;
- no telemetry/analytics/ads/crash SDK dependency markers;
- runtime privacy boundaries;
- credential and transport security invariants;
- localization parity and translation coverage;
- documentation links and release markers;
- Web companion integrity;
- Windows production build;
- Linux amd64/arm64/i386 package build.

## Documentation

- [Documentation index](docs/README.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Installation](docs/INSTALLATION.md)
- [Platform parity](docs/PLATFORM-PARITY.md)
- [Localization](docs/LOCALIZATION.md)
- [Settings](docs/SETTINGS.md)
- [Dependencies](docs/DEPENDENCIES.md)
- [Security](docs/SECURITY.md)
- [Privacy](docs/PRIVACY.md)
- [Testing](docs/TESTING.md)
- [Release history](docs/RELEASE-HISTORY.md)
- [GitHub Releases](docs/GITHUB-RELEASES.md)
- [Release verification](docs/RELEASE-VERIFICATION.md)
- [Signing](docs/SIGNING.md)
- [Shared-hosting/Web companion](docs/SHARED-HOSTING.md)
- [Roadmap](docs/ROADMAP.md)
- [Third-party notices](docs/THIRD-PARTY-NOTICES.md)
- [Contributing](docs/CONTRIBUTING.md)
- [Support](docs/SUPPORT.md)
- [Linux packaging](linux/README.md)
- [Web companion](GhostFTP%20WEB/README.md)

## Status of the 2.0 line

The `2.0.0` development line is the Windows/Linux consolidation and quality release. A version becomes a published release only after the complete CI and release gate passes and the immutable `ghostftp-v2.0.0` release is created from the validated `main` commit.
