# Ghost FTP native reference UI

Ghost FTP **0.0.4** uses focused native two-pane desktop layouts on Windows and Linux and a purpose-built native mobile workspace on Android. The same typed desktop Core owns protocol, profile, transfer, bandwidth policy and Remote Edit behavior on Windows/Linux; Android has its own bounded FTP/FTPS/SAF runtime while preserving the project security/privacy contract.

This is a source/runtime contract, not a mockup specification. Visible controls must map to real capabilities and real state.

## Canonical desktop workspace

The maintained Windows/Linux working surface consists of profile/application actions, Quick Connect, Local and Remote file panes, transfer actions, the transfer queue and a restrained status/version surface. Windows universal Setup and Portable launch the matching verified native application payload and therefore use the same workspace, localization, appearance and action-state logic after startup.

The Remote side exposes normal file management plus **Edit** for one supported regular remote text file. Remote Edit must not create a permanent third file pane.

## Appearance contract

**Classic Light is the fresh-install, missing-state and invalid-state primary appearance.** An explicitly persisted Dark selection remains respected on both Windows and Linux. Appearance is one canonical validated setting rather than a collection of overlapping cosmetic switches.

Classic Light deliberately avoids pure white as the dominant application surface. It uses cool neutral layers, while Dark uses the maintained navy/charcoal hierarchy.

### Classic Light — primary

| Role | RGB |
| --- | --- |
| Window | `238, 241, 245` (`#EEF1F5`) |
| Panel | `246, 248, 251` (`#F6F8FB`) |
| List | `250, 251, 253` (`#FAFBFD`) |
| Border | `199, 206, 216` (`#C7CED8`) |
| Primary text | `32, 37, 43` (`#20252B`) |
| Muted text | `101, 112, 131` (`#657083`) |
| Accent | `63, 99, 221` (`#3F63DD`) |
| Strong accent | `37, 75, 199` (`#254BC7`) |
| Selection | `220, 232, 255` (`#DCE8FF`) |

### Dark — maintained explicit choice

| Role | RGB |
| --- | --- |
| Window | `11, 15, 23` (`#0B0F17`) |
| Panel | `18, 24, 36` (`#121824`) |
| List | `22, 29, 42` (`#161D2A`) |
| Border | `44, 54, 72` (`#2C3648`) |
| Primary text | `242, 245, 250` (`#F2F5FA`) |
| Muted text | `151, 163, 184` (`#97A3B8`) |
| Accent | `91, 124, 250` (`#5B7CFA`) |
| Strong accent | `122, 152, 255` (`#7A98FF`) |
| Selection | `32, 47, 80` (`#202F50`) |

Theme data is local source data only. No remote stylesheet, font service, theme API, analytics endpoint or browser runtime is loaded.

On Windows, appearance is applied coherently so title bar, menus, native controls, headers and owner-drawn controls do not enter mixed Light/Dark state. On Linux, the validated persisted appearance selects the same shared source palette before the first frame is rendered, and saving Settings applies the new palette immediately to the maintained X11/XWayland-compatible frontend.

## Native dialog contract

Ghost FTP-owned Windows dialogs visually belong to the active application appearance and must not terminate the application message loop when they close.

Maintained rules include:

- only the main desktop window owns process-level `WM_QUIT`/`PostQuitMessage` lifecycle;
- auxiliary Prompt, Option, Settings and information-card windows close only their own bounded modal loop;
- application-owned dialogs use the active Ghost FTP owner when available and temporarily disable that owner while a modal decision is pending;
- controls, fonts and geometry scale from the current Windows DPI;
- dialog title bars follow the active Light/Dark state on supported Windows builds;
- localized action labels remain caller/catalog owned.

Closing **Nova mapa**, **Preimenuj**, **Postavke**, **Dijagnostika** or **O programu** with the title-bar X closes that surface only. It must not close Ghost FTP.

Linux settings, bookmarks, recursive-search, comparison and Remote Edit overlays own only their bounded overlay lifecycle and must not fabricate state outside their backing Engine/runtime operations.

## Settings surfaces

Windows Settings is one application-owned modal surface rather than a chain of unrelated prompts. It presents appearance, transfer concurrency, independent upload and download bandwidth ceilings, connection timeout, retry policy, destination conflict policy and delete confirmation together.

Bandwidth fields use explicit `KiB/s` units and `0 = unlimited`, and all numeric values are checked against the same `internal/config` bounds used by persisted settings normalization. Invalid input keeps the dialog open, shows localized corrective text and restores focus to the invalid field. **OK** returns one complete candidate settings value; **Cancel** or title-bar **X** discards the pending values.

The Linux Settings overlay exposes the same shared appearance and runtime policy through bounded native controls. Numeric bandwidth controls use maintained presets while preserving the same validated setting range and `0 = unlimited` semantics.

## Site Manager and saved profiles

![Ghost FTP Site Manager](images/ghost-ftp-site-manager.png)

Site Manager/profile workflows preserve explicit credential-consent and trust semantics. Usernames, passwords, passphrases, private-key paths, fingerprints and local/remote profile paths are not rendered in a navigation list as accidental disclosure.

Windows and Linux can save non-secret profile state independently of newly entered credentials. Linux 0.0.4 additionally requires a bounded second confirmation before newly entered password/private-key-passphrase material is sent to protected profile persistence, then clears the plaintext UI fields.

## Main Workspace

![Ghost FTP Main Workspace](images/ghost-ftp-main-workspace.png)

Actions are enabled from real state. A disabled operation remains disabled regardless of whether the user reaches it through a button, menu, list gesture or keyboard shortcut. Stale asynchronous callbacks are invalidated by connection/session identity where the maintained runtime exposes asynchronous work.

## File panes, sorting and transfer queue

Both desktop panes use the shared engine and filesystem/remote validation layers. Permissions are shown only when the server provides real permission metadata.

Windows and Linux use the shared item sorter for Name, Type, Size and Modified on both panes; the Remote pane additionally supports Permissions. Sorting is ascending/descending, directories remain first, unknown metadata sorts conservatively and filter+sort operates over a copy of the authoritative loaded snapshot. Row-indexed actions resolve only through the currently visible slice.

Pause, resume, cancel, retry and clear-finished operate through the canonical transfer manager. Four-way queued **Top / Up / Down / Bottom** actions reorder only queued scheduler slots and do not mutate running/terminal history. Progress, speed, ETA and byte counts may be shown only when backed by real transfer state. Bandwidth ceilings are enforced in the transport path rather than by repaint timing or a UI-only speed cap.

Bookmarks/profile start directories are navigation state, not hidden transfer authority. Server targets are account/session bound and are freshly listed before visible pane state is committed.

## Built-in Remote Editor

Remote Edit is deliberately simple:

- one Edit action from the Remote file surface;
- one native/modal editor surface;
- **Save**, **Reload** and **Close**;
- dirty-state indication;
- clear conflict/error status;
- keyboard support including save/reload shortcuts;
- no external editor process and no permanent extra application panel.

Windows uses an application-owned native text editor dialog. Linux uses the maintained X11/XWayland-compatible editor overlay. Both use the same Remote Edit engine for bounded open/save, UTF-8/text validation, conflict detection, line-ending preservation, permission handling and read-back verification. After a verified successful save, the remote list refreshes size/mtime metadata while retaining the edited-file selection.

## Android native workspace

Android 0.0.4 source exposes real **Files**, **Sites**, **Bookmarks**, **Transfers**, **Settings** and **About** surfaces plus semantic navigation. Small-screen navigation uses an application drawer; wider tablet layouts retain a visible sidebar. System-bar insets are respected on current target SDK behavior so interactive controls remain outside reserved system UI.

Android supports FTP and strict explicit FTPS. Local files are exposed only through Android Storage Access Framework capabilities. Passwords remain memory-only in the current Android source line, saved sites contain non-secret connection/navigation metadata, and transfers use staged final-name commit plus cancellation gating.

SFTP is intentionally absent from the Android protocol selector until strict native host-key identity verification exists. That omission is a security boundary, not an unfinished disabled control.

## Responsive behavior

Windows startup/minimum geometry respects the active monitor work area. Mixed-DPI transitions use the destination monitor, negative monitor origins remain valid and compact work areas must not force the application outside usable bounds.

Linux preserves the same major workflow priorities at supported sizes. Dynamic file-pane controls must not overlap recursive-search/comparison ownership of the same control region.

Android switches between drawer and persistent-sidebar navigation according to the maintained width threshold and applies system-bar insets to the root shell.

## Settings and About evidence

![Ghost FTP Settings](images/ghost-ftp-settings.png)

![Ghost FTP About](images/ghost-ftp-about.png)

About displays runtime product/version identity generated from canonical build `VERSION`; the current source/release candidate identity is **Ghost FTP 0.0.4** for desktop production builds, while the maintained Android development identity carries the `-dev` suffix.

## Authentic screenshot evidence

The maintained exact-head workflow `.github/workflows/ui-screenshots.yml` captures **real runtime UI** on Windows, Linux and Android. Mockups, image-generation output and manually composed approximations are not accepted as production UI evidence.

The current evidence contract is:

- Windows — 5 images: Main Workspace, Site Manager, Bookmarks, Settings, About;
- Linux — 3 images: Main Workspace, Bookmarks, Settings;
- Android — 7 images: Files, Navigation, Sites, Bookmarks, Transfers, Settings, About.

The final evidence job checks out the exact source SHA, downloads all three platform capture artifacts and runs `scripts/assemble_ui_evidence.py`. The verifier requires exactly **15 runtime images**, validates source/workflow identity, expected filenames, recorded byte counts and per-file SHA-256, then uploads `ghostftp-authentic-ui-verified-bundle`.

The evidence workflow has read-only repository contents permission. It does **not** commit or push screenshots, does not use `github-actions[bot]` to move the tested head, and does not treat an older evidence SHA as proof for a newer source revision.

Android capture navigation is semantic accessibility navigation only: exact `Open navigation`, exact `Navigate to <section>` nodes and exact post-click section-title assertions. Screenshot color/geometry/hard-coded coordinate fallbacks are not acceptable runtime evidence.

The repository-local images below remain the documentation visual set:

- `images/ghost-ftp-main-workspace.png`;
- `images/ghost-ftp-site-manager.png`;
- `images/ghost-ftp-settings.png`;
- `images/ghost-ftp-about.png`.

Those checked-in images are documentation assets, not a substitute for the fresh exact-head cross-platform evidence artifact required for a changed UI/release candidate.

## Accessibility and usability

- visible keyboard focus where native controls support it;
- readable contrast in Light and Dark desktop appearances;
- semantic Android navigation labels for authentic accessibility-based capture;
- clear destructive-action confirmation when enabled;
- localized desktop user-facing labels through the maintained 24-language catalog;
- no credential or server-secret text in screenshots/documentation evidence.

## Change rule

A maintained UI change is acceptable only when it maps to real product state/capability, stays behind the relevant engine/security boundary, remains usable at supported DPI/window/device sizes, maintains truthful enabled/disabled state, respects localization/privacy requirements and passes regression tests plus exact-head authentic runtime capture on every affected evidence platform.

See [Settings](SETTINGS.md), [Platform parity](PLATFORM-PARITY.md), [Testing](TESTING.md) and [Privacy](PRIVACY.md).
