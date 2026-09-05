# Windows and Linux platform parity

Ghost FTP 2.x supports two application platforms: **Windows** and **Linux**. Both editions use the same transfer, profile, settings, localization and security core. The presentation layer differs intentionally: Windows is a native Win32 graphical application, while Linux uses a hardened terminal interface.

The objective is functional parity without introducing a third-party cross-platform GUI runtime solely to make the interfaces visually identical.

## Shared core

Both Windows and Linux execute through the same typed `internal/api.Engine` boundary and share:

- connection validation;
- FTP, explicit FTPS and SFTP sessions;
- SFTP host-key trust;
- protected credential handling;
- connection/profile identity binding;
- remote file operations;
- local filesystem protections;
- transfer scheduling;
- conflict policy handling;
- retries and retry delays;
- connection timeout settings;
- parallel transfer settings;
- directory/tree transfer planning;
- path traversal/symlink/reparse-point protections;
- bounded disconnect/session shutdown;
- 24-language runtime registry;
- persistent settings/profile stores.

There is no alternate Linux protocol implementation and no alternate Windows transfer scheduler.

## Authentication parity

| Capability | Windows | Linux |
| --- | --- | --- |
| FTP password | Yes | Yes |
| Explicit FTPS password | Yes | Yes |
| SFTP password | Yes | Yes |
| SFTP private key | Yes | Yes |
| SFTP key passphrase | Yes | Yes |
| SFTP host fingerprint confirmation | Yes | Yes |
| Saved endpoint-bound credentials | Core support | Core support |

Ghost FTP 2.0 corrected a Linux defect from the 1.x line where the terminal frontend required an SFTP private key and rejected a non-empty key passphrase. Linux now passes the same password/key/passphrase model to the shared engine as Windows.

## File-operation parity

| Capability | Windows | Linux |
| --- | --- | --- |
| Remote list/navigation | GUI | `ls`, `cd`, `pwd` |
| Remote create folder | GUI | `mkdir` |
| Remote rename | GUI | `rename` |
| Remote delete | GUI | `delete` |
| Remote permissions/chmod | GUI | `chmod` |
| Upload file | GUI | `put` |
| Download file | GUI | `get` |
| Upload/download scheduling | Shared engine | Shared engine |
| Tree-transfer engine | Shared engine | Shared engine |

The same validation and transfer code owns the operation regardless of presentation.

## Transfer queue parity

Windows exposes queue actions as buttons and the transfer list. Linux exposes the same transfer-manager actions as commands:

- `jobs`
- `pause`
- `resume`
- `cancel <id>`
- `retry <id>`
- `clear`

The command surface does not create a separate queue implementation.

## Settings parity

The canonical settings model is shared. Current validated values are:

| Setting | Range / values | Default |
| --- | --- | --- |
| Language | 24 supported locales | `en` |
| Parallel transfers | 1–8 | 2 |
| Conflict policy | `skip`, `replace`, `replace_backup` | `replace_backup` |
| Automatic retries | 0–3 | 0 |
| Retry delay | 1–30 s | 3 s |
| Connection timeout | 5–60 s | 15 s |
| Confirm delete | boolean | enabled |

Linux can inspect settings using `settings` and modify supported values with:

```text
set parallelism <1-8>
set conflict <skip|replace|replace_backup>
set retries <0-3>
set retry-delay <1-30>
set timeout <5-60>
set confirm-delete <true|false>
```

All writes pass through `Engine.SetSettings` and the same fail-closed validation/migration used by Windows.

## Localization parity

The canonical registry contains 24 languages and English is the default/fallback. Windows supports live language switching and localized Setup primary flows. Linux resolves the same stored locale, can change it at runtime and uses the same translation catalogs for terminal operations.

CI verifies:

- English is first/default;
- all 24 catalogs exist;
- catalog keys and format verbs are valid;
- minimum real translation coverage is maintained;
- Windows live localization remains wired;
- Windows Setup primary copy remains localized;
- Linux runtime language switching remains wired.

## Visual presentation

### Windows

Windows is the premium reference GUI. It uses:

- a native Win32 window;
- high-DPI scaling;
- graphite/navy dark surfaces;
- owner-drawn action buttons;
- local/remote file panels;
- connection/profile controls;
- transfer queue controls;
- native dialogs and file/folder pickers.

### Linux

Linux is currently a terminal presentation over the same core. This keeps the application free of a bundled cross-platform GUI framework and allows amd64, arm64 and i386 package targets to remain small and auditable.

A future Linux graphical frontend is acceptable only if it can preserve the project's dependency, security and reproducibility requirements. A visual change must never fork the transfer/security engine.

## Retired application platforms

Android, iOS and macOS application targets were retired from the active 2.x source tree. Historical 1.x commits and releases remain immutable and may still contain those platform sources and artifacts.

The existing Web companion remains in the repository as a separate source surface. It is not counted as a Windows/Linux desktop application platform artifact in 2.x releases.

## Parity change rule

Any new core transfer or connection option must answer three questions before merge:

1. Is it implemented once in the shared core rather than duplicated per UI?
2. Can both Windows and Linux invoke it safely?
3. Is its behavior covered by platform-neutral tests or explicit platform parity checks?

If an option is temporarily exposed on only one frontend, documentation and CI must identify the gap rather than claiming parity that does not exist.
