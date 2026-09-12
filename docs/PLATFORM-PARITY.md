# Windows and Linux platform parity

Ghost FTP **0.0.5** is one desktop product with native Windows and Linux frontends. Both platforms use the **same typed `internal/api.Engine`** and the same protocol, transfer, profile, settings, localization, Remote Edit and security layers.

Parity means equivalent supported behavior and protocol/security semantics with honest native-platform UX. Platform primitives may differ, but a supported action must not exist as a decorative/dead control on either desktop frontend.

## Shared protocol contract

Both desktop platforms support FTP, explicit FTPS, SFTP password authentication, SFTP private-key authentication, SFTP key passphrase handling, strict SFTP host-key fingerprint trust, local/remote navigation, transfer trees, remote file operations where supported, connection timeouts and privacy-safe diagnostics.

Fresh Quick Connect resolves to explicit FTPS on port 21. Plain FTP is explicit compatibility. Secure-transport failure is never silently retried through a weaker protocol.

Linux automatic SFTP password/private-key-passphrase delivery requires the maintained trusted root-controlled executable/helper/parent provenance. User-writable Portable/per-user execution remains usable for workflows that do not require that automatic AskPass secret delivery, but Ghost FTP fails closed rather than exposing secrets through a mutable helper path.

## Shared profile/settings model

Windows and Linux use the same typed configuration/profile model. Saved credentials are opt-in and local. Account identity changes remain fail-closed so credentials cannot silently cross protocol/host/port/username identity.

Settings normalization, conflict policy, retry behavior, parallelism, timeout, language, appearance and directional bandwidth ceilings are shared contracts. Upload/download values use binary KiB/s, reserve `0` for unlimited and are validated by shared configuration.

## Appearance and 24-language parity

Classic Light is the fresh/fallback appearance; Dark is a maintained explicit choice. Windows uses native theme integration while Linux applies the source-defined selected palette before first paint. Neither platform loads remote theme services/fonts/styles.

Ghost FTP ships one **24-language** local registry with English default/fallback. Security/privacy-sensitive prompts and transfer-setting labels are catalog-backed.

## Transfer and file-management parity

Both platforms route transfers through the same manager and remote abstraction: pause/resume/cancel/retry/clear lifecycle, queued **Top / Up / Down / Bottom** ordering, connection-generation binding, truthful progress/speed/ETA, retry classification, path confinement and staged transfer commit/rollback semantics.

Bandwidth ceilings are independent aggregate directional budgets. FTP/FTPS use curl `limit-rate`; SFTP uses OpenSSH `sftp -l` with conservative unit conversion.

Both frontends expose local/remote panes, refresh/navigation, create/rename/delete, upload/download, shared sorting, current-folder filtering, bounded recursive search, conservative directory comparison and synchronized navigation for safely proven paired ordinary directories.

## Remote Edit parity

Windows and Linux expose built-in Remote Edit through the same engine contract for FTP, FTPS and SFTP. Safeguards include bounded text size, UTF-8/binary validation, line-ending preservation, SHA-256 revision/conflict detection, transaction-style upload/read-back, trustworthy permission preservation and metadata refresh.

0.0.5 adds stricter Windows editor-session lifecycle ownership: open/save/reload cycles remain one serialized session and cannot be re-entered into a parallel stale editor session while asynchronous work is completing.

## Navigation parity

Both frontends expose local/remote bookmarks and profile start directories through shared non-secret persistence. Remote navigation is account/session-bound and performs fresh listing before visible state is committed.

## Security parity

Both desktop platforms preserve FTPS certificate/hostname validation, SFTP host-key trust, no silent secure-to-plain downgrade, validated paths/bounded recursive operations, protected credential handling and privacy-safe diagnostics. Linux additionally enforces trusted AskPass executable/parent provenance.

## Windows-specific implementation

Windows uses native Win32 UI, DPI-aware layout and the current-user saved-secret protection boundary.

Ghost FTP 0.0.5 adds re-entry/lifecycle guards around profile persistence, local/remote file mutations and Remote Edit sessions. Duplicate or stale commands cannot bypass in-flight mutation state, and nested application-owned modal loops preserve process-level shutdown semantics rather than swallowing `WM_QUIT`.

**Production Authenticode is optional.** When a trusted protected signing identity is configured, artifacts must verify; when it is absent, official publication records:

```text
WINDOWS_AUTHENTICODE=unsigned
```

Public Windows distribution exposes only:

```text
Ghost-FTP-0.0.5-Setup.exe
Ghost-FTP-0.0.5-Portable.exe
```

Verified native x64/x86 payloads remain internal. The bootstrap uses `GetNativeSystemInfo`, verifies staged bytes and performs no runtime download.

## Linux-specific implementation

The canonical 0.0.5 release path is `linux/BUILD-DISTROS.sh`:

- Debian DEB: `amd64`, `arm64`, `i386`;
- Ubuntu DEB: `amd64`, `arm64`, `i386`;
- Fedora RPM: `x86_64`, `aarch64`, `i686`;
- Portable tar.gz: `amd64`, `arm64`, `i386`.

One production executable is compiled per Go architecture and reused across matching distro/Portable variants. Native install/remove/runtime/GUI coverage is **x86-64 only** on **Debian 13 amd64**, **Ubuntu 26.04 LTS amd64** and **Fedora 44 x86_64**. Other canonical architectures retain build, metadata, extraction and byte-parity verification.

## Android source parity boundary

Android is an active native development surface tied to root `VERSION`, with Files/Sites/Bookmarks/Transfers/Settings/About, FTP + strict explicit FTPS, SAF-scoped storage, bounded parsing and staged transfer lifecycle. 0.0.5 additionally binds pending connection sessions to the owning Activity lifecycle and rejects stale callbacks after destruction/recreation.

Android SFTP remains hidden until strict host-key identity verification has a maintained implementation. Android is not included in the Windows/Linux public release allow-list.

## Release parity

A successful Windows build cannot substitute for a failed Linux build and vice versa. Android has an independent exact-head APK/runtime gate but does not enlarge the public release asset set.

The current public 0.0.5 contract requires **14 platform artifacts / 17 public files** and publishes `ghcr.io/bren-wp/ghost-ftp:0.0.5` as a distribution bundle.

See [Architecture](ARCHITECTURE.md), [Installation](INSTALLATION.md), [Settings](SETTINGS.md), [Reference UI](REFERENCE-UI.md), [Testing](TESTING.md), [Signing](SIGNING.md) and [Security](SECURITY.md).
