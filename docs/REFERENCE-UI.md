# Ghost FTP native reference UI

Ghost FTP **0.0.5** uses focused native two-pane desktop layouts on Windows and Linux and a purpose-built native mobile workspace on Android. The same typed desktop Core owns protocol, profile, transfer, bandwidth policy and Remote Edit behavior on Windows/Linux; Android has its own bounded FTP/FTPS/SAF runtime while preserving project security/privacy boundaries.

This is a source/runtime contract, not a mockup specification. Visible controls must map to real capability and state.

## Canonical desktop workspace

The Windows/Linux surface consists of application/profile actions, Quick Connect, Local and Remote file panes, transfer actions, transfer queue and restrained status/version surfaces. The Remote side exposes normal file management plus **Remote Edit** for one supported regular text file.

## Appearance contract

Classic Light is the fresh/missing/invalid-state primary appearance. Dark is the maintained explicit choice. Both palettes are local source data; no remote stylesheet/font/theme/analytics service is loaded.

Windows applies appearance coherently across title bar, menus and controls. Linux applies the validated persisted palette before first frame and applies a newly saved palette immediately.

## Native dialog and lifecycle contract

Ghost FTP-owned Windows dialogs visually belong to the active appearance and must not terminate the application message loop when they close.

Maintained rules include:

- only the main desktop window owns process-level shutdown lifecycle;
- Prompt, Option, Settings, information cards and Remote Edit close only their bounded modal/session loop;
- application-owned dialogs use the active Ghost FTP owner where available;
- controls/fonts/geometry scale from current DPI;
- nested modal loops preserve a received process shutdown request;
- 0.0.5 mutation/session guards prevent stale command re-entry while asynchronous profile, file or Remote Edit work is active.

Linux overlays own only their bounded overlay lifecycle and cannot fabricate engine state.

## Settings surfaces

Windows Settings presents appearance, concurrency, independent upload/download bandwidth ceilings, connection timeout, retry policy, destination conflict policy and delete confirmation in one validated native surface. Bandwidth fields state `KiB/s` and `0 = unlimited`. Linux exposes the same shared settings policy through bounded native controls.

## Site Manager and saved profiles

![Ghost FTP Site Manager](images/ghost-ftp-site-manager.png)

Site Manager/profile workflows preserve explicit credential-consent and trust semantics. Windows and Linux can save non-secret profile state independently from newly entered credentials. 0.0.5 additionally prevents parallel Windows encrypted profile persistence mutations.

## Main Workspace

![Ghost FTP Main Workspace](images/ghost-ftp-main-workspace.png)

Actions are enabled from real state. A disabled operation remains disabled regardless of whether the user attempts to reach it by button, menu, list gesture or keyboard. Stale async callbacks are invalidated by connection/session identity where the runtime is asynchronous.

## File panes and transfer queue

Both desktop panes use shared engine/filesystem/remote validation layers. Sorting is shared for Name, Type, Size and Modified plus remote Permissions; directories remain first and filter+sort operates over copies of authoritative loaded snapshots.

Pause/resume/cancel/retry/clear operate through the transfer manager. Four-way queued **Top / Up / Down / Bottom** reorders queued scheduler slots only. Progress/speed/ETA may be shown only when backed by real state.

0.0.5 guards Windows local/remote create-directory, rename, delete and permission mutations against overlapping duplicate/stale commands without globally locking unrelated opposite-side workflow.

## Built-in Remote Editor

Remote Edit remains deliberately compact: one Edit action, one native/modal editor, **Save / Reload / Close**, dirty-state indication and conflict/error status. Windows uses an application-owned native editor; Linux uses the maintained X11/XWayland-compatible overlay.

0.0.5 keeps the Windows editor session busy from initial open through save/reload continuations until terminal close/error, preventing parallel re-entry during async cycles.

## Android native workspace

Android 0.0.5 source exposes real **Files**, **Sites**, **Bookmarks**, **Transfers**, **Settings** and **About** surfaces plus semantic navigation. Small screens use an application drawer; wider layouts use a visible sidebar. System-bar insets keep controls outside reserved system UI.

Android supports FTP and strict explicit FTPS. Local access uses Android Storage Access Framework capabilities; passwords remain memory-only in the current source line; saved sites contain non-secret connection/navigation metadata; transfers use staged final-name commit/cancellation gating.

In 0.0.5 an in-flight connection belongs to one Activity instance. Destruction/recreation aborts its pending session and stale completion callbacks cannot reactivate an obsolete Activity UI.

SFTP remains absent from the Android protocol selector until strict native host-key identity verification exists.

## Settings and About evidence

![Ghost FTP Settings](images/ghost-ftp-settings.png)

![Ghost FTP About](images/ghost-ftp-about.png)

About displays runtime product/version identity generated from canonical build `VERSION`; current desktop production identity is **Ghost FTP 0.0.5**, while the Android development identity carries the `-dev` suffix.

## Authentic screenshot evidence

`.github/workflows/ui-screenshots.yml` captures real exact-head runtime UI. **Mockups, image-generation output and manually composed approximations are not accepted** as production UI evidence.

The maintained evidence contract is:

- Windows — 5 images: Main Workspace, Site Manager, Bookmarks, Settings, About;
- Linux — 3 images: Main Workspace, Bookmarks, Settings;
- Android — 7 images: Files, Navigation, Sites, Bookmarks, Transfers, Settings, About.

The final verifier requires 15 runtime images, verifies source/workflow identity, filenames, byte counts and SHA-256, and emits the read-only `ghostftp-authentic-ui-verified-bundle`. The workflow does not commit/push evidence back to the tested branch.

Repository-local documentation assets remain:

- `images/ghost-ftp-main-workspace.png`;
- `images/ghost-ftp-site-manager.png`;
- `images/ghost-ftp-settings.png`;
- `images/ghost-ftp-about.png`.

See [Settings](SETTINGS.md), [Platform parity](PLATFORM-PARITY.md), [Testing](TESTING.md) and [Privacy](PRIVACY.md).
