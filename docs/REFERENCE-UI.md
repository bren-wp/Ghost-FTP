# Ghost FTP desktop reference UI

This document defines the maintained visual and interaction contract for the Ghost FTP desktop workspace introduced on the 0.1.0 Beta line.

It is a **source/runtime contract**, not a mockup specification. Controls shown by the application must map to real engine capabilities and real state. Decorative controls that imply unsupported backend behavior are not acceptable.

## Canonical Windows workspace

The Windows desktop application is the current graphical reference implementation. Windows Setup and Windows Portable package the same Ghost FTP application executable, so the installed and portable editions use the same workspace code, theme, localization and action-state logic.

The reference shell is organized as follows:

1. persistent left navigation/profile sidebar;
2. application menu and top action toolbar;
3. connection-log card;
4. Quick Connect card;
5. equal Local and Remote file-manager cards;
6. remote-search field;
7. full-width transfer-queue card;
8. quiet connection/version status surface.

The intended visual character is a dense professional desktop file manager using deep navy surfaces, cool blue borders, muted secondary text and a blue-violet primary accent. The UI remains native Win32 and does not load a web UI, tracking runtime or third-party GUI framework.

## Menu contract

The top-level menu order follows the graphical reference:

- File;
- View;
- Transfers;
- Servers;
- Tools;
- Help.

Every menu action routes to the same command handlers and engine validation used by the corresponding button or panel action. Menu actions must never create a second connection, transfer, rename or delete implementation.

## Toolbar contract

The primary toolbar exposes the real actions currently available through the application:

- Connect;
- Disconnect;
- Upload;
- Download;
- Refresh;
- New folder;
- Rename;
- Delete;
- Site Manager;
- Settings;
- Diagnostics.

Toolbar controls mirror canonical action state. A disabled canonical operation must remain disabled through the toolbar as well; the toolbar is not an authorization or validation bypass.

The button renderer uses Windows-provided Segoe Fluent/MDL2-compatible glyph resources and the native application drawing layer. Ghost FTP does not bundle an external icon font merely for this toolbar.

## Quick Connect

Quick Connect uses the normal connection path and normal input validation. It is not a lightweight or less-secure alternate connector.

On sufficiently wide windows the primary connection fields fit on one row. At narrower supported widths the identity/password controls wrap to a second row so fields do not overlap or become unusable. SFTP-only private-key/passphrase controls appear only when SFTP is selected.

## Connection log

The connection-log surface is local to the running application. It is not a remote log collector.

The Windows status/log helper:

- strips line-breaking control characters from individual status messages;
- bounds individual entries;
- keeps a bounded recent-history window;
- adds local display timestamps;
- never creates a telemetry or diagnostics upload endpoint.

Diagnostics are generated locally from current application state. The Diagnostics action does not send the report to Ghost FTP, an analytics provider or a crash-reporting service.

## Local and Remote panels

Both file cards use the shared engine model and existing filesystem/remote validation layers.

The local visual column order is:

1. Name;
2. Size;
3. Type;
4. Modified.

The remote visual column order is:

1. Name;
2. Size;
3. Type;
4. Modified;
5. Permissions.

The logical list-view column storage may differ from the visual order where required to preserve existing model/index behavior. The reference layout explicitly sets the visual order rather than rewriting item semantics.

## Real permission metadata

The **Permissions** column is backed by actual remote listing metadata; it must not be filled with invented values.

Ghost FTP currently accepts displayable permission metadata from:

- UNIX-style FTP `LIST` entries such as `-rw-r--r--` or `drwxr-xr-x`;
- SFTP `ls -la` output, which is parsed through the same bounded UNIX listing parser;
- MLSD `unix.mode` when a server actually supplies that extension.

The display normalizer accepts only bounded POSIX-style symbolic modes or three/four-digit octal values. MLSD `perm=` capability strings such as `adfrw` are **not** presented as POSIX permissions because they describe supported operations, not the file mode.

Servers that do not expose a usable mode simply produce an empty Permissions cell. That is more accurate than fabricating `644`, `755` or another guessed value.

## Remote search

The current Windows remote search filters the already loaded remote directory model in memory. It does not send search text to a third-party service and does not create extra remote-server requests while typing.

The full unfiltered directory model is retained separately so clearing or changing the search query restores correct item/index semantics for download, rename and delete actions.

## Transfer queue

The queue remains the shared transfer manager rendered in the reference shell. Pause, resume, cancel, retry and clear-finished actions continue to operate through the canonical queue implementation.

A visual field may be added only when the transfer model exposes the corresponding real state. In particular, speed, ETA or byte-remaining columns must not be shown with fabricated values solely to resemble another FTP client.

## Theme palette

The native Windows palette is intentionally defined in source rather than downloaded from a theme service. The current reference values are:

| Role | RGB |
| --- | --- |
| Window | `5, 17, 29` |
| Panel | `7, 25, 39` |
| List | `8, 28, 43` |
| Border | `28, 62, 86` |
| Primary text | `224, 237, 255` |
| Muted text | `126, 161, 201` |
| Accent | `96, 126, 255` |
| Strong accent | `110, 84, 255` |
| Success | `57, 216, 166` |
| Warning | `247, 190, 72` |

Theme changes must preserve readable contrast, high-DPI behavior and disabled/action-state clarity.

## Localization

English is the canonical/default/fallback language. The shell/menu/toolbar/columns participate in the maintained 24-language runtime selection.

The remote Permissions heading is localized through the same `common.permissions` catalog key used by the permissions action. File-action dialogs and status messages must not silently overwrite the selected locale with hardcoded Croatian or English strings.

## Authentic screenshot evidence

Repository screenshots under `docs/images/` are generated from the real production Windows x64 Portable executable by the dedicated Actions workflow.

The workflow:

1. disables Go telemetry;
2. builds the production Windows package;
3. starts the real Portable executable;
4. captures native windows with `PrintWindow(PW_RENDERFULLCONTENT)`;
5. verifies PNG signature, dimensions, size and SHA-256;
6. persists the images only if the capture commit is still the branch head.

Mockup or image-generation output is not accepted as evidence for the production UI.

## Windows Setup / Portable equivalence

Setup and Portable are packaging variants of one Windows application source. The workspace, colors, command routing, profile model, localization and privacy behavior therefore remain identical after the executable starts.

Installer-only screens are a separate native Setup surface and do not fork the application workspace implementation.

## Linux presentation boundary

Linux currently uses the same transfer/security/settings/profile engine with a hardened terminal frontend. It is **not** currently pixel-identical to the Windows graphical shell.

This is an explicit documented gap, not a hidden claim of visual parity. A future Linux graphical implementation must satisfy all of the following before it can be called reference-equivalent:

- no fork of the connection/transfer/security engine;
- no telemetry or tracking dependency;
- no hidden web/service runtime;
- reproducible amd64/arm64/i386 packaging as applicable;
- explicit documentation of every OS GUI runtime prerequisite;
- equivalent action-state and destructive-operation safeguards;
- authentic runtime capture/testing rather than mockups.

The project does not call a cross-platform GUI dependency “zero dependency.” If a Linux GUI requires a system toolkit or display protocol library, that prerequisite must be documented accurately and reviewed under the dependency policy.

## Change rule

A desktop UI change is acceptable only when:

1. it maps to real product state or capability;
2. it does not bypass the shared engine/security boundary;
3. it remains usable under supported DPI/window sizes;
4. it respects the selected runtime language;
5. it does not add tracking or an undocumented runtime dependency;
6. regression tests and authentic Windows capture remain green.
