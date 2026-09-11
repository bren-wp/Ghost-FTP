# Windows and Linux platform parity

Ghost FTP **0.0.4** is one desktop product with native Windows and Linux frontends. Both platforms use the **same typed `internal/api.Engine`** and the same protocol, transfer, profile, settings, localization, Remote Edit and security layers.

Parity means equivalent supported behavior and protocol/security semantics with honest native-platform UX. Native operating-system implementation details may differ where Windows and Linux require different primitives, but a supported product action must not exist as a decorative or dead control on either platform.

## Shared protocol contract

Both platforms support:

- FTP;
- FTPS;
- SFTP password authentication;
- SFTP private-key authentication;
- SFTP key passphrase handling;
- SFTP host-key fingerprint trust;
- local/remote navigation;
- upload/download/tree transfers;
- remote file operations where supported;
- connection timeouts;
- privacy-safe diagnostics.

Fresh/quick-connect state resolves to **explicit FTPS on port 21** on both platforms. Plain FTP remains an explicit compatibility option. A secure transport failure is never silently retried as a weaker protocol.

On Linux, automatic password and private-key-passphrase delivery through OpenSSH AskPass is available only when the running Ghost FTP executable and credential-bearing helper boundary have the maintained trusted root-controlled provenance. Package-installed builds satisfy that boundary. User-writable Portable/per-user execution remains usable for workflows that do not require automatic AskPass secret delivery, but Ghost FTP fails closed rather than exposing password/passphrase capability through a mutable helper path.

## Shared profile/settings model

Windows and Linux use the same typed configuration/profile model. Platform-specific secret protection is intentionally different, but saved credentials remain opt-in and local.

Both native profile-save paths require an explicit credential-persistence decision instead of silently persisting newly entered passwords or private-key passphrases. Linux uses a bounded second-confirmation window before passing newly entered secrets to the protected profile store and clears the plaintext UI fields after the operation; Windows uses its maintained current-user protected secret boundary. Account identity changes remain fail-closed so stored credentials cannot silently cross protocol/host/port/username identities.

Settings normalization, conflict policy, retry behavior, parallelism, timeout, language, appearance and directional bandwidth ceilings remain shared contracts. Upload/download bandwidth values use binary KiB/s, reserve `0` for unlimited, and are validated by the shared configuration layer rather than frontend shadow state. Compatibility JSON fields are migration state, not justification for duplicate UI controls.

## Appearance

**Classic Light is the primary fresh/fallback appearance.**

- **Windows** provides the maintained Classic Light / Dark appearance path. An explicitly persisted Dark selection remains respected and is initialized coherently with native title-bar/control theming.
- **Linux** exposes the same validated Light / Dark `Appearance` setting and applies the persisted palette before the first native frame is rendered. Saving a new appearance applies the shared source-defined palette immediately to the maintained X11/XWayland-compatible frontend.

Both implementations use local source-defined colors and add no theme service, browser runtime, telemetry or network dependency.

## 24-language parity

Ghost FTP ships one **24-language** local registry. English is the default/fallback. Both frontends consume normalized locale codes from the same catalog, and missing optional text falls back safely without online translation.

Security/privacy-sensitive credential-persistence prompts and bandwidth labels are catalog-backed rather than hardcoded into one Windows path.

## Transfer parity

Both platforms route transfers through the same transfer manager and remote abstraction. Shared behavior includes queued/running/terminal states, pause/resume/cancel/retry/clear lifecycle, four-way queued **Top / Up / Down / Bottom** priority reordering, connection-generation binding, truthful progress/speed/ETA snapshots, retry classification, local containment, upload-source snapshot validation, staged/rollback-oriented remote operations, cleanup and terminal-state correctness.

Bandwidth policy is also shared: upload and download ceilings are independent aggregate directional budgets. The transfer scheduler divides the configured budget conservatively across configured worker slots, and each attempt snapshots its effective allowance. FTP/FTPS enforce the result through curl `limit-rate`; SFTP uses OpenSSH `sftp -l` with conservative unit conversion. The UI does not emulate throttling with a timer or busy-wait loop.

Tree-download directory preparation is anchored to opened filesystem roots so boundary or ancestor pathname replacement cannot redirect local directory creation.

## Remote Edit parity

Windows and Linux expose built-in Remote Edit through the same shared engine contract for FTP, FTPS and SFTP.

Shared safeguards include:

- one supported regular remote text file at a time;
- bounded content size;
- UTF-8/text validation and binary rejection;
- LF/CRLF/CR preservation and mixed-ending rejection;
- SHA-256 revision/conflict detection;
- transactional upload/read-back verification;
- remote permission preservation where the server reports a trustworthy mode;
- metadata refresh after a successful save.

The native presentation differs intentionally: Windows uses an application-owned native text editor dialog; Linux uses the maintained X11/XWayland-compatible editor overlay. Both keep the user workflow deliberately compact with **Save / Reload / Close** rather than adding a permanent third file pane.

## Connection-manager parity

The frontends use the shared remote manager rather than independent session implementations. Regression coverage exercises successful manager connect/list/operation/disconnect behavior, invalid FTP credentials and FTPS-to-plaintext failure. Connection identity/generation prevents stale work from silently attaching to a later connection.

## File-management parity

Both frontends expose local/remote panes, navigation, selection, refresh, create, rename, delete, upload, download and supported remote editing. Action availability remains truthful and backed by the same Core behavior.

File ordering uses the same shared `itemlist.SortBy` implementation. Both platforms support Name, Type, Size and Modified ordering in both panes; the remote pane additionally supports Permissions. Ascending/descending ordering retains the shared directories-first rule, unknown metadata sorts conservatively, and Linux restores the selected visible item by name after a sort just as Windows keeps its list selection stable through its native control lifecycle.

Both also expose non-destructive current-folder filtering, bounded recursive local/server search and conservative directory comparison. Filtering and sorting compose over the authoritative loaded directory snapshot rather than mutating it. Synchronized comparison navigation is available only for exact paired ordinary directories proven safe on both sides and performs fresh listings before committing pane paths.

Native keyboard focus/accelerator presentation may differ between Win32 and X11/XWayland, but the supported file actions, sorting, filtering, search, comparison, bookmarks, queue actions and settings behaviors remain available through maintained native controls on both platforms.

## Navigation parity

Both frontends expose reusable local and remote bookmarks and profile start directories through the shared non-secret persistence model. Remote navigation state is bound to protocol, host, port and username. Bookmark/start-directory activation performs fresh navigation and revalidates active connection identity before committing server pane state; Linux also restores the previously verified local base when a selected profile local start cannot be listed.

Quick Connect does not create hidden bookmarks or hidden saved-site state.

## Security parity

Both platforms preserve:

- FTPS certificate and hostname validation;
- SFTP host-key trust;
- no silent secure-to-plain downgrade;
- validated paths and bounded recursive operations;
- symlink/reparse-aware local safety appropriate to the platform;
- protected credential handling with bounded secret lifetime;
- privacy-safe diagnostics without intentional credential reproduction.

SFTP protected-secret ownership distinguishes transient/session-owned material from credentials borrowed from stored profiles. Cancel/expiry/mismatch and failed setup paths must not retain newly owned secret blobs longer than necessary, and cleanup must not invalidate borrowed profile credentials.

Linux additionally requires trusted executable provenance at both sides of the AskPass boundary. The helper path used for credential delivery must be root-controlled and bound to the running executable identity, and the immediate OpenSSH parent must resolve to a trusted root-controlled `ssh`/`sftp` executable.

## Windows-specific implementation

Windows uses native Win32 UI, DPI-aware layout, native dialogs and the current-user Windows saved-secret protection boundary.

The main profile-save flow and Site Manager use the same explicit credential-save consent semantics. Profile identity/path data can be saved without persisting newly entered credentials when the user declines consent.

**Production Authenticode is optional.** When a trusted protected signing identity is configured, generated Windows artifacts must verify successfully; when it is absent, official publication remains explicitly unsigned and records:

```text
WINDOWS_AUTHENTICODE=unsigned
```

A generated/self-signed development certificate is never substituted for trusted production publisher identity.

The canonical Windows build still produces verified x64 and x86 native Setup plus Portable payloads as internal staging artifacts. Public 0.0.4 distribution exposes only:

```text
Ghost-FTP-0.0.4-Setup.exe
Ghost-FTP-0.0.4-Portable.exe
```

The x86-compatible bootstrap uses `GetNativeSystemInfo`-derived architecture rather than environment variables, selects the embedded native x64/x86 payload, verifies staged bytes and performs no runtime download. Architecture-specific staging executables must not leak into the public release directory.

## Linux-specific implementation

Linux uses the maintained native X11/XWayland-compatible frontend and platform-local saved-secret/storage protections.

The canonical 0.0.4 release path is `linux/BUILD-DISTROS.sh`:

- Debian DEB: `amd64`, `arm64`, `i386`;
- Ubuntu DEB: `amd64`, `arm64`, `i386`;
- Fedora RPM: `x86_64`, `aarch64`, `i686`;
- distro-neutral Portable tar.gz: `amd64`, `arm64`, `i386`.

Exactly one Linux production executable is compiled per Go architecture and reused across its matching Debian, Ubuntu, Fedora and Portable variants. `.github/workflows/linux-distro-packages.yml` rebuilds those artifacts from the exact source head, checks metadata and verifies byte-for-byte executable parity.

`.github/workflows/linux-distro-install.yml` independently verifies real package-manager install/remove lifecycle and startup of the installed production GUI under local Xvfb on **Debian 13 amd64**, **Ubuntu 26.04 LTS amd64** and **Fedora 44 x86_64**. It verifies runtime dependencies, package ownership and uninstall residue without weakening production filesystem safety checks.

Native distro-install verification is intentionally **x86-64 only**. The arm64/aarch64 and i386/i686 artifacts retain exact-head build, metadata, extraction and byte-parity coverage; the project does not claim native package-manager/runtime installation coverage for those architectures until such a gate exists.

The legacy generic `linux/BUILD.sh` path remains CI compatibility coverage for DEB/portable parity but is not the canonical 0.0.4 public release allow-list.

Portable archives intentionally preserve binary parity with package builds, but extraction into a user-writable directory does not inherit package-manager root-controlled executable provenance. Therefore Linux Portable/per-user execution does not claim automatic SFTP password or private-key-passphrase AskPass support under the hardened same-UID local-attacker model.

## Android source parity boundary

Android is an active native source surface tied to root `VERSION`, with installable development APK output, real Files/Sites/Bookmarks/Transfers/Settings/About surfaces, FTP and strict explicit FTPS, SAF-scoped local storage, bounded FTP response parsing and staged transfer commit/cancellation behavior.

Android is not represented as desktop 1:1 UI and is not included in the Windows/Linux public release allow-list. SFTP remains hidden on Android until strict host-key identity verification has a maintained native implementation. This boundary is intentional and fail-closed rather than a decorative unsupported protocol option.

## Release parity

The production workflow independently builds and verifies both public desktop platform families before publication. A successful Windows build cannot substitute for a failed Linux build, and vice versa. Android has an independent exact-head lint/APK gate and authentic emulator UI evidence but does not enlarge the public release asset set.

The current public 0.0.4 contract requires **14 platform artifacts / 17 public files**: two universal Windows files, twelve Linux distro/Portable files and three release metadata/verification files.

The current release publishes its verified distribution bundle at `ghcr.io/bren-wp/ghost-ftp:0.0.4`. Latest-only retention preserves that exact-version package and removes superseded package versions only after the new release is verified.

## Definition of parity complete

A cross-platform feature or packaging claim is complete when:

1. shared Core/API semantics are implemented once where appropriate;
2. each platform exposes only behavior its native frontend can implement truthfully;
3. platform-specific implementation differences are documented rather than hidden behind dead controls;
4. security/privacy boundaries remain equivalent;
5. localization/fallback works;
6. platform and packaging tests pass at the claimed coverage level;
7. documentation distinguishes build verification, native installation verification and release publication;
8. authentic UI evidence and release metadata match the shipped implementation.

See [Architecture](ARCHITECTURE.md), [Installation](INSTALLATION.md), [Settings](SETTINGS.md), [Reference UI](REFERENCE-UI.md), [Testing](TESTING.md), [Signing](SIGNING.md) and [Security](SECURITY.md).
