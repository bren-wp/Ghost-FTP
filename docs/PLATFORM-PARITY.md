# Windows and Linux platform parity

Ghost FTP **0.0.1 Beta** is one desktop product with native Windows and Linux frontends. Both platforms use the **same typed `internal/api.Engine`** and the same protocol, transfer, profile, settings, localization, Remote Edit and security layers.

Parity means equivalent protocol/security semantics and honest native-platform UX, not pixel-identical widgets or a requirement to expose a control before its backend lifecycle is complete.

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

Settings normalization, conflict policy, retry behavior, parallelism, timeout and language remain shared contracts. Compatibility JSON fields are migration state, not justification for duplicate UI controls.

## Appearance

**Classic Light is the primary fresh/fallback appearance.**

- **Windows** provides the maintained Classic Light / Dark appearance path. An explicitly persisted Dark selection remains respected and is initialized coherently with native title-bar/control theming.
- **Linux** uses the maintained native palette contract without exposing a fake appearance control whose backend lifecycle is incomplete.

Both implementations use local source-defined colors and add no theme service, browser runtime, telemetry or network dependency.

## 24-language parity

Ghost FTP ships one **24-language** local registry. English is the default/fallback. Both frontends consume normalized locale codes from the same catalog, and missing optional text falls back safely without online translation.

Security/privacy-sensitive credential-persistence prompts are catalog-backed rather than hardcoded into one Windows path.

## Transfer parity

Both platforms route transfers through the same transfer manager and remote abstraction. Shared behavior includes queued/running/terminal states, pause/resume/cancel/retry/clear lifecycle, connection-generation binding, truthful progress/speed/ETA snapshots, retry classification, local containment, upload-source snapshot validation, staged/rollback-oriented remote operations, cleanup and terminal-state correctness.

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

Both frontends expose local/remote panes, navigation, selection, refresh, create, rename, delete, upload, download and supported remote editing. Native keyboard/sorting details may differ, but action availability must remain truthful and backed by the same Core behavior.

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

Windows uses native Win32 UI, DPI-aware layout, native dialogs and the current-user Windows saved-secret protection boundary. Production packages include x64/x86 Setup and Portable binaries.

The main profile-save flow and Site Manager use the same explicit credential-save consent semantics. Profile identity/path data can be saved without persisting entered credentials when the user declines consent.

**Production Authenticode is optional.** When a trusted protected signing identity is configured, generated Windows artifacts must verify successfully; when it is absent, official publication remains explicitly unsigned and records:

```text
WINDOWS_AUTHENTICODE=unsigned
```

A generated/self-signed development certificate is never substituted for trusted production publisher identity.

The canonical Windows build produces x64 and x86 Setup plus x64 and x86 Portable executables. The public `x32` Setup filename is a byte-identical compatibility alias of the verified x86 Setup artifact; it is not a third Windows architecture build.

## Linux-specific implementation

Linux uses the maintained native X11/XWayland-compatible frontend and platform-local saved-secret/storage protections.

The canonical release path uses `linux/BUILD.sh` to build generic DEBs and package-manager-neutral `.tar.gz` archives for `amd64`, `arm64` and `i386`. CI proves that each generic DEB and portable archive contains the same compiled `ghostftp` executable byte-for-byte.

The maintained source also has a separate distro-specific packaging contract in `linux/BUILD-DISTROS.sh`:

- Debian DEB: `amd64`, `arm64`, `i386`;
- Ubuntu DEB: `amd64`, `arm64`, `i386`;
- Fedora RPM: `x86_64`, `aarch64`, `i686`;
- distro-neutral Portable tar.gz: `amd64`, `arm64`, `i386`.

`.github/workflows/linux-distro-packages.yml` rebuilds those artifacts from the exact source head, checks metadata and verifies byte-for-byte executable parity across matching Debian, Ubuntu, Fedora and Portable packages.

`.github/workflows/linux-distro-install.yml` independently verifies a real package-manager install/remove lifecycle and startup of the installed production GUI under local Xvfb on **Debian 13 amd64**, **Ubuntu 26.04 LTS amd64** and **Fedora 44 x86_64**. It verifies runtime dependencies, package ownership and uninstall residue without weakening production filesystem safety checks.

Native distro-install verification is intentionally **x86-64 only**. The arm64/aarch64 and i386/i686 artifacts retain exact-head build, metadata, extraction and byte-parity coverage; the project does not claim native package-manager/runtime installation coverage for those architectures until such a gate exists.

The distro-specific CI package family is supplemental. It is verified build/install coverage, but it is **not yet part of the canonical release allow-list**.

Portable archives intentionally preserve binary parity with package builds, but extraction into a user-writable directory does not inherit package-manager root-controlled executable provenance. Therefore Linux Portable/per-user execution does not claim automatic SFTP password or private-key-passphrase AskPass support under the hardened same-UID local-attacker model.

## Release parity

The production workflow independently builds and verifies both platform families before publication. A successful Windows build cannot substitute for a failed Linux build, and vice versa.

The current public 0.0.1 contract requires **12 platform artifacts / 15 public files**: five Windows files, seven Linux files and three release metadata/verification files.

Supplemental Debian/Ubuntu/Fedora/Portable distro-specific CI artifacts do not change those public release counts until the canonical release workflow explicitly stages, allow-lists, hashes, publishes and reads them back for a future version.

Pre-1.0 Beta releases do not claim a Stable GHCR distribution bundle. The latest-only retention workflow removes obsolete package versions after a newly published release is verified.

## Definition of parity complete

A cross-platform feature or packaging claim is complete when:

1. shared Core/API semantics are implemented once where appropriate;
2. each platform exposes only behavior its native frontend can implement truthfully;
3. platform-specific differences are documented rather than hidden behind dead controls;
4. security/privacy boundaries remain equivalent;
5. localization/fallback works;
6. platform and packaging tests pass at the claimed coverage level;
7. documentation distinguishes build verification, native installation verification and release publication;
8. authentic UI evidence and release metadata match the shipped implementation.

See [Architecture](ARCHITECTURE.md), [Installation](INSTALLATION.md), [Settings](SETTINGS.md), [Reference UI](REFERENCE-UI.md), [Testing](TESTING.md), [Signing](SIGNING.md) and [Security](SECURITY.md).
