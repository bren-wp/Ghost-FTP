# Ghost FTP desktop reference UI

This document defines the maintained visual and interaction contract for **Ghost FTP 1.1.0 Stable** and later compatible desktop releases.

It is a **source/runtime contract**, not a mockup specification. Controls shown by the application must map to real engine capabilities and real state. Decorative controls that imply unsupported backend behavior are not acceptable.

## Canonical workspace

Ghost FTP keeps a dense professional two-pane file-transfer layout rather than adding dashboard cards or decorative navigation that competes with file operations.

The maintained working surface consists of:

1. profile/application actions;
2. Quick Connect;
3. Local and Remote file panes;
4. transfer actions between those panes;
5. the transfer queue;
6. a restrained status/version surface.

Windows Setup and Portable package the same application executable and therefore use the same workspace, localization, appearance and action-state logic after startup.

## Appearance contract

Ghost FTP 1.1.0 adds **Classic Light** without creating a collection of overlapping cosmetic switches.

### Dark

The established dark appearance remains available on Windows.

| Role | RGB |
| --- | --- |
| Window | `8, 10, 15` (`#080A0F`) |
| Panel | `15, 19, 28` (`#0F131C`) |
| List | `21, 26, 37` (`#151A25`) |
| Border | `37, 45, 60` (`#252D3C`) |
| Primary text | `244, 247, 255` (`#F4F7FF`) |
| Muted text | `142, 153, 173` (`#8E99AD`) |
| Accent | `82, 119, 245` (`#5277F5`) |
| Strong accent | `114, 147, 255` (`#7293FF`) |
| Success | `74, 215, 155` (`#4AD79B`) |
| Warning | `242, 186, 85` (`#F2BA55`) |
| Danger | `255, 100, 118` (`#FF6476`) |
| Selection | `29, 42, 74` (`#1D2A4A`) |

### Classic Light

Classic Light follows the clarity and information density associated with traditional professional FTP clients, but does not copy FileZilla artwork, branding or proprietary assets.

| Role | RGB |
| --- | --- |
| Window | `243, 244, 246` (`#F3F4F6`) |
| Panel | `255, 255, 255` (`#FFFFFF`) |
| List | `255, 255, 255` (`#FFFFFF`) |
| Border | `200, 205, 214` (`#C8CDD6`) |
| Primary text | `31, 35, 40` (`#1F2328`) |
| Muted text | `102, 112, 133` (`#667085`) |
| Accent | `63, 99, 221` (`#3F63DD`) |
| Strong accent | `37, 75, 199` (`#254BC7`) |
| Success | `27, 127, 75` (`#1B7F4B`) |
| Warning | `154, 103, 0` (`#9A6700`) |
| Danger | `198, 40, 40` (`#C62828`) |
| Selection | `220, 232, 255` (`#DCE8FF`) |

Theme data is local source data only. No remote stylesheet, font service, theme API, analytics endpoint or browser runtime is loaded.

On Windows, the appearance choice is applied on the next start so title bar, menus, native controls, headers and owner-drawn controls are initialized consistently. The settings UI explicitly communicates this behavior.

The Linux graphical frontend uses Classic Light as the canonical 1.1 palette. It deliberately does not expose a runtime appearance toggle until complete native switching can be implemented without mixed-state rendering or race-prone global palette mutation.

## Menu and action contract

Menu/button surfaces are alternate entry points to the same canonical commands. They must not create second implementations of connect, transfer, rename, delete, Site Manager or settings behavior.

Actions are enabled from real state. A disabled operation must remain disabled regardless of whether the user reaches it through a button, menu, list gesture or keyboard shortcut.

## Icons

Windows toolbar/buttons use operating-system Segoe Fluent/MDL2-compatible glyph resources and the native drawing layer. File panes use the operating-system file image list where available.

Rules:

- no remote icon CDN;
- no external icon-font dependency solely for toolbar decoration;
- one semantic glyph per action;
- destructive actions must not reuse a positive/transfer glyph;
- disabled icons/text must remain readable in both maintained Windows appearances;
- packaging variants must not ship different workspace icon sets.

The product logo comes from the same canonical executable resource used by production Setup/Portable packaging so documentation and runtime branding cannot drift to a second manually maintained logo source.

## Quick Connect

Quick Connect uses the normal connection path and normal validation. It is not a lightweight or less-secure alternate connector.

On sufficiently wide windows the main connection fields fit on one row. At narrower supported widths the controls reflow so fields do not overlap or become unusable. SFTP private-key/passphrase controls are enabled only for the relevant protocol state.

## Local and Remote panes

Both panes use the shared engine and filesystem/remote validation layers.

The local visual columns are Name, Size, Type and Modified. The remote pane adds Permissions when the server provides real permission metadata.

Permissions must never be fabricated. Servers without usable POSIX-style mode information produce an empty Permissions cell rather than a guessed value.

Sorting, double-click navigation, keyboard actions and selection restoration must preserve the item/model identity used by rename, delete, upload and download operations.

## Transfer queue

Pause, resume, cancel, retry and clear-finished actions operate through the canonical transfer manager. Visual progress, speed, ETA or byte counts may be shown only when backed by real transfer state.

The queue must not invent values simply to resemble another FTP client.

## No duplicated options

The UI may expose one control per canonical behavior. Compatibility fields in persisted JSON are not justification for duplicate controls. Destination conflict handling, for example, is represented by one conflict-policy choice even though legacy mirror fields remain for backward-compatible state migration.

Appearance follows the same rule: one Dark/Classic Light decision on Windows, not separate switches for window background, panels, lists, icon colors and accent colors.

## Localization

English is the default/fallback language. The maintained catalog contains 24 selectable local languages.

Appearance labels/help are local strings and do not require a translation service. File actions, dialogs, menus, status text and settings must not silently replace the selected locale with hardcoded developer text.

## Authentic screenshot evidence

Repository screenshots under `docs/images/` are generated from the **real production Windows x64 Portable executable** by `.github/workflows/ui-screenshots.yml`.

The workflow:

1. disables Go telemetry;
2. builds the production Windows package;
3. starts the real Portable executable;
4. captures the native main workspace and Site Manager window;
5. verifies PNG signature, plausible dimensions, file size and SHA-256;
6. persists the images only when the capture commit is still the branch head.

Mockups, image-generation output and manually composed approximations are not accepted as production UI evidence.

## Windows Setup / Portable equivalence

Setup and Portable are packaging variants of one Windows application source. Workspace layout, appearance, commands, profile model, localization and privacy behavior therefore remain identical after startup.

Installer-only screens are a separate native Setup surface and do not fork the application workspace implementation.

## Linux presentation boundary

Linux uses the same transfer/security/settings/profile engine with a dependency-free X11/XWayland graphical frontend. It is native-platform different from Win32 rather than pretending to be pixel-identical.

The Linux GUI remains under the same product boundary: no fork of the connection/transfer/security engine, no telemetry, no hidden browser/service runtime, shared destructive-operation safeguards, and reproducible amd64/arm64/i386 packaging. The terminal frontend remains available for headless or explicit fallback use.

## Change rule

A desktop UI change is acceptable only when:

1. it maps to real product state or capability;
2. it does not bypass the shared engine/security boundary;
3. it remains usable at supported DPI/window sizes;
4. every visible action has a valid enabled/disabled state;
5. labels and icons remain semantically consistent and non-duplicated;
6. it respects the selected runtime language;
7. it does not add tracking or an undocumented runtime dependency;
8. regression tests and authentic Windows capture remain green.
