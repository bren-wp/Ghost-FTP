# Windows and Linux platform parity

Ghost FTP currently maintains two desktop application platforms: **Windows** and **Linux**. Both editions use the same transfer, profile, settings, localization and security core. Windows uses the native Win32 reference frontend; Linux now uses a dependency-free native X11/XWayland graphical frontend and retains the hardened terminal frontend for headless or explicit fallback use.

The active maturity baseline is **0.1.0 Beta**. Functional parity work completed before the version reset is preserved; changing the active version line does not remove or downgrade existing capabilities.

The project does not claim pixel-identical visual parity where it does not yet exist. See [Desktop reference UI](REFERENCE-UI.md).

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
| Endpoint-bound saved profile metadata | Yes | Yes |
| Windows DPAPI secret persistence | Yes | No |

Linux intentionally does not emulate Windows DPAPI with plaintext or weak reversible storage. Linux profile metadata is protected by the documented Linux profile envelope, while already-used password/passphrase material is not reconstructed merely to make `profile-save` convenient.

When a Linux user explicitly accepts a new SFTP host key, the accepted public fingerprint is retained in the in-process connection metadata so a later `profile-save` can persist the verified endpoint pin. Passwords and private-key passphrases are removed from the session config after connection and are not printed by profile commands.

## Windows reference-shell boundary

Windows Setup and Windows Portable package the same application executable/source. Once the app starts, both expose the same:

- deep navy reference shell;
- left profile/navigation sidebar;
- menu and top action toolbar;
- Connection Log and Quick Connect cards;
- Local/Remote file cards;
- remote search;
- transfer queue;
- live localization;
- command/action-state validation;
- Site Manager and settings surfaces.

The graphical shell is presentation only. Toolbar/menu actions route to the same canonical connection, file-operation and transfer code paths used elsewhere in the application.

## Windows Site Manager parity boundary

Windows provides a graphical Site Manager because it is the native GUI reference frontend. Site Manager does not create a separate connection stack.

It uses the same profile store and connection engine for:

- saved site selection;
- protocol, server and port;
- username and protected password handling;
- local and remote start paths;
- SFTP private key and key passphrase;
- endpoint-bound stored credentials;
- normal connection validation and SFTP host trust.

The one-click **Sites** toolbar button and the application menu open the same Site Manager implementation. Quick connection uses the normal connection path after transferring entered values to the main connection state.

Linux exposes the same underlying profile/connection capabilities through its graphical profile controls and through terminal fallback commands; neither path creates a second connection stack.

## File-operation parity

| Capability | Windows | Linux |
| --- | --- | --- |
| Remote list/navigation | GUI | `ls`, `cd`, `pwd` |
| Remote create folder | GUI | `mkdir` |
| Remote rename | GUI | `rename` |
| Remote delete | GUI | `delete` |
| Remote permissions/chmod | GUI | `chmod` |
| Remote mode visibility when server supplies it | Permissions column | Listing metadata/command output path |
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

Linux local commands call `Engine.LocalList`, `LocalMkdir`, `LocalRename` and `LocalDelete`. They therefore retain the same no-follow/no-replace/root-delete protections used by the Windows local panel instead of falling back to arbitrary shell commands.

`get` and `put` resolve relative local paths from the Linux local panel directory rather than from the process working directory. Relative remote paths resolve from the active remote directory.

`gettree` and `puttree` call `Engine.AddTreeTransfer`; they inherit the shared tree planner's bounded depth/item count, symlink handling, path validation and normal transfer queue/conflict policy behavior.

## Remote permission metadata

The shared `model.Item` can carry optional remote permission display metadata.

Ghost FTP currently derives it from real server listing data:

- UNIX-style FTP `LIST` modes;
- SFTP `ls -la` modes through the shared UNIX listing parser;
- MLSD `unix.mode` when provided.

MLSD `perm=` capabilities are not misrepresented as POSIX modes. Unknown or malformed values remain empty.

The Windows Remote card displays this as the fifth **Permissions** column. The local card intentionally remains four columns because local authorization semantics are not inferred through the remote listing model.

## Remote-search parity boundary

The current Windows remote search is a local in-memory filter over the already loaded directory. It does not create another remote search protocol or third-party query service.

Linux users can navigate/list through the graphical file panel or terminal fallback. Any graphical search enhancement must reuse the same item model and must not introduce a separate server/indexing service.

## Delete confirmation parity

The canonical `confirm-delete` setting is enforced on both frontends. Linux remote `delete` and local `ldelete` fail closed unless the user explicitly confirms the operation when confirmation is enabled.

Disabling confirmation is itself a validated persisted settings change (`set confirm-delete false`). The terminal does not silently bypass the setting for convenience.

Windows destructive controls use the same canonical settings/action-state boundary and are disabled when the current selection/connection state does not permit the requested operation.

## Transfer queue parity

Windows exposes queue actions as buttons and the transfer list. Linux exposes the same transfer-manager actions in its GUI and preserves these terminal fallback commands:

- `jobs`
- `pause`
- `resume`
- `cancel <id>`
- `retry <id>`
- `clear`

The command surface does not create a separate queue implementation.

The graphical UI does not fabricate transfer speed, ETA or remaining-byte values merely to look like another client. Such columns may be added only when the shared transfer model exposes real data.

## Saved-profile parity

Linux graphical profile controls and terminal profile commands use the shared protected profile store:

- `profiles` lists public profile metadata;
- `profile-show <id>` displays non-secret profile state;
- `profile-save <name>` stores the active endpoint, verified SFTP fingerprint when applicable, private-key path and current local/remote working paths;
- `profile-remove <id>` removes a profile through the shared store.

`profile-save` intentionally does **not** copy an already-used password or passphrase from the active connection. Those secrets are cleared after authentication, so the command never reconstructs or prints them merely to create a profile.

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

Linux can edit the same validated settings through its graphical Settings overlay and can also inspect/modify them through the terminal fallback:

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

The canonical registry contains 24 languages and English is the default/fallback. Windows supports live language switching and localized Setup primary flows. Linux resolves the same stored locale for its graphical and terminal surfaces and uses the same translation catalogs/fallback normalization.

CI verifies:

- English is first/default;
- all 24 catalogs exist;
- catalog keys and format verbs are valid;
- minimum real translation coverage is maintained;
- Windows live localization remains wired;
- Windows Setup primary copy remains localized;
- Linux runtime language switching remains wired.

The Windows UI regression suite additionally rejects a return of hardcoded Croatian action strings that would overwrite another selected runtime locale.

Command names remain stable English technical tokens so scripts/documentation do not change when the UI language changes.

## Visual presentation

### Windows

Windows is the graphical reference GUI. It uses:

- native Win32/DWM/common-control facilities;
- high-DPI scaling;
- deep navy/graphite surfaces and blue-violet accent hierarchy;
- owner-drawn toolbar/action buttons;
- persistent left sidebar;
- Connection Log and Quick Connect cards;
- balanced local/server dual-pane workspace;
- remote in-memory search;
- real remote Permissions column when metadata exists;
- full-width transfer queue;
- one-click Sites access and native Site Manager;
- native dialogs and file/folder pickers.

The refined layout is reapplied after resize, DPI, protocol and language changes so the application does not fall back to obsolete geometry after startup.

### Linux

Linux now provides a real graphical desktop presentation over the same core. The frontend speaks directly to the local X11/XWayland display protocol using the Go standard library and the existing typed `api.Engine`; it does not bundle GTK, Qt, Electron, a webview, a tracking runtime or another transfer engine. The hardened terminal frontend remains available for headless systems or explicit `GHOSTFTP_UI=terminal` use.

The GUI exposes Quick Connect, SFTP host-key trust, profiles, local/server panes, file and tree transfers, transfer queue controls, local/remote file operations, remote permissions and validated transfer settings. Destructive actions retain the shared confirm-delete policy.

**Linux is not claimed to be pixel-identical to native Win32.** It follows the same Ghost FTP brand palette and interaction hierarchy while preserving native platform/runtime constraints and small auditable amd64, arm64 and i386 packages.

The Linux desktop requires a local X11-compatible display (native X11 or XWayland). Protocol transport prerequisites remain the system `curl` and OpenSSH tools documented elsewhere.

## Authentic UI evidence

Windows production validation includes a dedicated workflow that builds the real x64 Portable executable and captures the native main workspace and Site Manager with `PrintWindow(PW_RENDERFULLCONTENT)`.

The workflow verifies PNG signature, dimensions, size and SHA-256 and refuses to persist a capture if the branch moved after the source commit. Mockup/image-generation output is not accepted as production UI evidence.

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

Windows/reference regressions verify:

- canonical sidebar/toolbar/cards remain wired;
- toolbar actions mirror canonical action state;
- remote search retains a full unfiltered model;
- local file columns remain four-column reference order;
- remote file columns remain five-column reference order with Permissions;
- permissions are backed by validated LIST/SFTP/MLSD `unix.mode` data;
- live language switching updates the Permissions heading;
- hardcoded Croatian file-action strings do not return;
- authentic capture remains buildable on a real Windows runner.

The repository/platform audits ensure retired application targets do not return and both maintained platform production builds remain present.

## Parity change rule

Any new core transfer or connection option must answer three questions before merge:

1. Is it implemented once in the shared core rather than duplicated per UI?
2. Can both Windows and Linux invoke it safely?
3. Is its behavior covered by platform-neutral tests or explicit platform parity checks?

If an option is temporarily exposed on only one frontend, documentation and CI must identify the gap rather than claiming parity that does not exist.
