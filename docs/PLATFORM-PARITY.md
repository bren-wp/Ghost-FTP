# Windows and Linux platform parity

Ghost FTP currently maintains two desktop application platforms: **Windows** and **Linux**. Both editions use the same transfer, profile, settings, localization and security core. The presentation layer differs intentionally: Windows is a native Win32 graphical application, while Linux uses a hardened terminal interface.

The active maturity baseline is **0.1.0 Beta**. Functional parity work completed before the version reset is preserved; changing the active version line does not remove or downgrade existing capabilities.

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

Earlier repository work corrected a Linux defect where the terminal frontend required an SFTP private key and rejected a non-empty key passphrase. That completed fix remains part of the 0.1.0 Beta baseline. Linux now passes the same password/key/passphrase model to the shared engine as Windows.

When a Linux user explicitly accepts a new SFTP host key, the accepted public fingerprint is retained in the in-process connection metadata so a later `profile-save` can persist the verified endpoint pin. Passwords and private-key passphrases are still removed from the session config after connection and are not printed by profile commands.

## Windows Site Manager parity boundary

Windows additionally provides a graphical Site Manager because it is the native GUI reference frontend. Site Manager does not create a separate connection stack.

It uses the same profile store and connection engine for:

- saved site selection;
- protocol, server and port;
- username and protected password handling;
- local and remote start paths;
- SFTP private key and key passphrase;
- endpoint-bound stored credentials;
- normal connection validation and SFTP host trust.

The one-click **Sites** toolbar button and the application menu open the same Site Manager implementation. Quick connection uses the normal `connectNow()` path after transferring the entered values to the main connection state.

Linux exposes the same underlying profile/connection capabilities through terminal commands rather than reproducing the Windows dialog.

## File-operation parity

| Capability | Windows | Linux |
| --- | --- | --- |
| Remote list/navigation | GUI | `ls`, `cd`, `pwd` |
| Remote create folder | GUI | `mkdir` |
| Remote rename | GUI | `rename` |
| Remote delete | GUI | `delete` |
| Remote permissions/chmod | GUI | `chmod` |
| Local list/navigation | GUI | `lls`, `lcd`, `lpwd` |
| Local create folder | GUI | `lmkdir` |
| Local rename | GUI | `lrename` |
| Local delete | GUI | `ldelete` |
| Upload file | GUI | `put` |
| Download file | GUI | `get` |
| Upload folder/tree | GUI/shared tree planner | `puttree` |
| Download folder/tree | GUI/shared tree planner | `gettree` |
| Upload/download scheduling | Shared engine | Shared engine |
| Tree-transfer limits/security | Shared engine | Shared engine |

Linux local commands call `Engine.LocalList`, `LocalMkdir`, `LocalRename` and `LocalDelete`. They therefore retain the same no-follow/no-replace/root-delete protections used by the Windows local panel instead of falling back to shell commands.

`get` and `put` resolve relative local paths from the Linux local panel directory rather than from the process working directory. Relative remote paths resolve from the active remote directory. This makes the two-panel terminal workflow predictable and consistent with the Windows file-manager model.

`gettree` and `puttree` call `Engine.AddTreeTransfer`; they inherit the shared tree planner's bounded depth/item count, symlink handling, path validation and normal transfer queue/conflict policy behavior.

## Delete confirmation parity

The canonical `confirm-delete` setting is enforced on both frontends. Linux remote `delete` and local `ldelete` fail closed unless the user explicitly confirms the operation when confirmation is enabled.

Disabling confirmation is itself a validated persisted settings change (`set confirm-delete false`). The terminal does not silently bypass the setting for convenience.

## Transfer queue parity

Windows exposes queue actions as buttons and the transfer list. Linux exposes the same transfer-manager actions as commands:

- `jobs`
- `pause`
- `resume`
- `cancel <id>`
- `retry <id>`
- `clear`

The command surface does not create a separate queue implementation.

## Saved-profile parity

Linux profile commands use the same protected profile store as Windows:

- `profiles` lists public profile metadata;
- `profile-show <id>` displays non-secret profile state, including whether a password/passphrase exists;
- `profile-save <name>` stores the active endpoint, verified SFTP fingerprint when applicable, private-key path and current local/remote working paths;
- `profile-remove <id>` removes a profile through the shared store.

`profile-save` intentionally does **not** copy an already-used password or passphrase from the active connection. Those secrets are cleared after authentication, so the command never reconstructs or prints them merely to create a profile. Credential persistence remains an explicit protected-profile operation rather than an implicit side effect of connecting.

On Windows, selecting a saved profile or Site Manager entry also does not reveal a protected saved password or private-key passphrase as plaintext. Credential reuse stays inside the protected profile-resolution boundary.

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

Command names remain stable English technical tokens so scripts/documentation do not change when the UI language changes. User-facing status/error copy continues to use the localization layer and English fallback when a reviewed translation is unavailable.

## Visual presentation

### Windows

Windows is the graphical reference GUI. It uses:

- a native Win32 window;
- high-DPI scaling;
- graphite/navy dark surfaces;
- owner-drawn action buttons;
- a balanced local/server dual-pane workspace;
- a visible status/session strip;
- direct upload/download controls between panes;
- a full-width transfer queue;
- connection/profile controls;
- one-click Sites access and native Site Manager;
- native dialogs and file/folder pickers.

The refined layout is reapplied after resize, DPI, protocol and language changes so the application does not fall back to obsolete geometry after startup.

### Linux

Linux is currently a terminal presentation over the same core. The prompt exposes both remote and local working directories, and the command groups are separated into Remote, Local, Files, Queue, Profiles and Options to keep the expanded capability discoverable.

This presentation keeps the application free of a bundled cross-platform GUI framework and allows amd64, arm64 and i386 package targets to remain small and auditable.

A future Linux graphical frontend is acceptable only if it can preserve the project's dependency, security and reproducibility requirements. A visual change must never fork the transfer/security engine.

## Active version and packaging parity

The root `VERSION` file is shared by both maintained platforms.

During the pre-stable line, `0.x.y` builds are Beta releases. Windows Setup, Windows Portable and Linux packages all use the same numeric version. The first stable version is reserved as **1.0.0**.

See [Versioning policy](VERSIONING.md).

## Retired application platforms

Android, iOS and macOS application targets are not part of the active Windows/Linux source/build matrix. Historical commits and releases remain immutable and may still contain those platform sources and artifacts.

The existing Web companion remains in the repository as a separate source surface. It is not counted as a Windows/Linux desktop application platform artifact in current releases.

## Parity regression coverage

Linux-specific regressions verify:

- remote relative paths resolve from the active remote directory;
- local relative paths resolve from the active local panel directory;
- absolute local paths remain absolute;
- delete confirmation is bypassed only when the validated setting explicitly disables it;
- empty, negative or unrecognized confirmations remain fail-closed;
- explicit affirmative confirmation is accepted through the localization-aware affirmative parser.

Windows production validation additionally exercises native package builds, control creation, Site Manager construction and authentic UI capture on a real Windows runner.

The repository/platform audits ensure retired application targets do not return and both maintained platform production builds remain present.

## Parity change rule

Any new core transfer or connection option must answer three questions before merge:

1. Is it implemented once in the shared core rather than duplicated per UI?
2. Can both Windows and Linux invoke it safely?
3. Is its behavior covered by platform-neutral tests or explicit platform parity checks?

If an option is temporarily exposed on only one frontend, documentation and CI must identify the gap rather than claiming parity that does not exist.
