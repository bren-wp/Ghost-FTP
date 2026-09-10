# Windows and Linux platform parity

Ghost FTP **0.0.3** is one desktop product with native Windows and Linux frontends. Both platforms use the **same typed `internal/api.Engine`** and the same protocol, transfer, profile, settings, localization, Remote Edit and security layers.

Parity means equivalent protocol/security semantics and honest native-platform UX, not pixel-identical widgets.

## Shared protocol contract

Both platforms support FTP, explicit FTPS, **SFTP password** authentication, SFTP private-key authentication, **SFTP key passphrase** handling, host-key fingerprint trust, local/remote navigation, upload/download/tree transfers, remote operations, privacy-safe diagnostics and bounded connection behavior. Fresh connections use explicit FTPS on port 21; failed secure negotiation never silently downgrades to plain FTP.

## Shared settings and transfer behavior

Windows and Linux use the same typed settings model for parallelism, connection timeout, retries, conflict policy and independent upload/download bandwidth ceilings. Bandwidth values use KiB/s with `0 = unlimited`. The scheduler applies a conservative aggregate directional budget and the real transport enforces the resulting per-attempt cap.

Both platforms route transfers through the shared transfer manager with queue lifecycle, connection-generation binding, retry classification, local containment, staging/rollback and truthful progress/speed/ETA behavior.

## Remote Edit and file-management parity

Both frontends expose built-in Remote Edit through the same bounded text/revision/conflict/read-back engine. Both support non-destructive current-folder filtering, bounded recursive local/server search, conservative directory comparison and synchronized navigation only for safely proven paired ordinary directories.

## 24-language parity

Ghost FTP ships one **24-language** local registry. English is the default/fallback and both frontends consume normalized locale codes from the same catalog without online translation.

## Security parity

Both platforms preserve FTPS certificate/hostname validation, SFTP host-key trust, no silent downgrade, validated paths, symlink/reparse-aware local safety, protected credential handling and privacy-safe diagnostics.

Linux additionally enforces trusted executable provenance around credential-bearing OpenSSH AskPass. Package-installed root-controlled builds can satisfy that boundary; user-writable Portable/per-user execution fails closed for automatic SFTP password/passphrase delivery rather than exposing secrets through a mutable helper path.

## Windows-specific implementation

Windows uses native Win32 UI, DPI-aware layout and current-user saved-secret protection. Public 0.0.3 downloads are exactly two universal executables:

```text
Ghost-FTP-0.0.3-Setup.exe
Ghost-FTP-0.0.3-Portable.exe
```

The canonical builder still creates x64/x86 native Setup/Portable payloads internally. The public bootstrap selects the native payload using Windows system architecture information, verifies staged bytes and performs no runtime download.

**Production Authenticode is optional.** Configured trusted signatures must verify; otherwise official publication records:

```text
WINDOWS_AUTHENTICODE=unsigned
```

## Linux-specific implementation

The canonical release path is `linux/BUILD-DISTROS.sh`:

- Debian DEB: `amd64`, `arm64`, `i386`;
- Ubuntu DEB: `amd64`, `arm64`, `i386`;
- Fedora RPM: `x86_64`, `aarch64`, `i686`;
- distro-neutral Portable tar.gz: `amd64`, `arm64`, `i386`.

`.github/workflows/linux-distro-packages.yml` checks package metadata and byte-for-byte binary parity across matching package variants. `.github/workflows/linux-distro-install.yml` performs real install/remove/runtime/GUI smoke on **Debian 13 amd64**, **Ubuntu 26.04 LTS amd64** and **Fedora 44 x86_64**.

Native package-manager/runtime coverage remains **x86-64 only**. Other architectures retain build, metadata, extraction and parity coverage.

## Release parity

The current public 0.0.3 contract requires **14 platform artifacts / 17 public files**: two universal Windows executables, twelve canonical Linux packages/archives and three release metadata/verification files. A successful Windows build cannot substitute for a failed Linux build, and vice versa.

The verified distribution bundle is `ghcr.io/bren-wp/ghost-ftp:0.0.3`. Latest-only retention removes superseded public release/package identities only after successor verification succeeds.

## Definition of parity complete

A cross-platform feature or packaging claim is complete only when shared Core behavior, platform UI, security/privacy boundaries, localization, tests, package/build verification, documentation and authentic evidence agree on the claimed behavior.

See [Architecture](ARCHITECTURE.md), [Installation](INSTALLATION.md), [Settings](SETTINGS.md), [Reference UI](REFERENCE-UI.md), [Testing](TESTING.md), [Signing](SIGNING.md) and [Security](SECURITY.md).
