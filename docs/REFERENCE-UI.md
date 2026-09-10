# Ghost FTP desktop reference UI

Ghost FTP **0.0.2** uses a focused native two-pane desktop layout on Windows and Linux. The same typed Core owns protocol, profile, transfer and Remote Edit behavior; platform frontends differ only where the operating system requires native presentation details.

This is a source/runtime contract, not a mockup specification. Visible controls must map to real engine capabilities and real state.

## Canonical workspace

The maintained working surface consists of profile/application actions, Quick Connect, Local and Remote file panes, transfer actions, the transfer queue and a restrained status/version surface. Windows Setup and Portable package the same application executable and therefore use the same workspace, localization, appearance and action-state logic after startup.

The Remote side exposes normal file management plus **Edit** for one supported regular remote text file. Remote Edit must not create a permanent third file pane.

## Appearance contract

**Classic Light is the fresh-install, missing-state and invalid-state primary appearance.** An explicitly persisted Dark selection remains respected on Windows. Appearance is one canonical decision rather than a collection of overlapping cosmetic switches.

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

### Dark — explicit Windows choice

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

On Windows, appearance is applied coherently so title bar, menus, native controls, headers and owner-drawn controls do not enter mixed Light/Dark state. The Linux graphical frontend keeps the maintained native palette contract without exposing a fake control whose backend lifecycle is incomplete.

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

## Settings surface

Windows Settings is one application-owned modal surface rather than a chain of unrelated prompts. It presents appearance, transfer concurrency, connection timeout, retry policy, destination conflict policy and delete confirmation together.

Numeric values are checked against the same `internal/config` bounds used by persisted settings normalization. Invalid input keeps the dialog open, shows localized corrective text and restores focus to the invalid field. **OK** returns one complete candidate settings value; **Cancel** or title-bar **X** discards the pending values.

## Site Manager

![Ghost FTP Site Manager](images/ghost-ftp-site-manager.png)

Site Manager provides a compact profile navigator and detail surface while preserving the same credential-consent and trust semantics as the main connection workflow. Usernames, passwords, passphrases, private-key paths, fingerprints and local/remote profile paths are not rendered in the left navigation.

## Main Workspace

![Ghost FTP Main Workspace](images/ghost-ftp-main-workspace.png)

Actions are enabled from real state. A disabled operation remains disabled regardless of whether the user reaches it through a button, menu, list gesture or keyboard shortcut. Stale asynchronous callbacks are invalidated by connection-generation state.

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

## File panes and transfer queue

Both panes use the shared engine and filesystem/remote validation layers. Permissions are shown only when the server provides real permission metadata. Sorting, navigation and selection restoration preserve the model identity used by rename, delete, upload, download and edit operations.

Pause, resume, cancel, retry and clear-finished operate through the canonical transfer manager. Progress, speed, ETA and byte counts may be shown only when backed by real transfer state.

## Responsive behavior

Windows startup/minimum geometry respects the active monitor work area. Mixed-DPI transitions use the destination monitor, negative monitor origins remain valid and compact work areas must not force the application outside usable bounds.

Linux preserves the same major workflow priorities at supported sizes without exposing controls whose backend behavior is incomplete.

## Settings and About evidence

![Ghost FTP Settings](images/ghost-ftp-settings.png)

![Ghost FTP About](images/ghost-ftp-about.png)

About displays the runtime product/version identity generated from canonical build `VERSION`; the current release identity is **Ghost FTP 0.0.2**.

## Authentic screenshot evidence

Repository screenshots under `docs/images/` are generated from the **real production Windows x64 Portable executable** by `.github/workflows/ui-screenshots.yml`. Mockups, image-generation output and manually composed approximations are not accepted as production UI evidence.

The currently persisted repository image set was generated by authentic UI workflow run **#201** from source commit `a1e9635f5724ea8b53afca9830f28f7fa9159798` and persisted by screenshot commit `29f3a9a069df37107772265987ecfd251b645c3e`:

- `docs/images/ghost-ftp-main-workspace.png` — SHA-256 `659caccf3fab3e9add23424e45709f541c902219ba5212f6287ab5ed25c8cb6e`;
- `docs/images/ghost-ftp-site-manager.png` — SHA-256 `63e9292530030afab7a95b21788d9ec1da80b6f7bca1ba0ce8a132fd665602a9`;
- `docs/images/ghost-ftp-settings.png` — SHA-256 `b46b8c9c0730e96b1a0ed9ba54e84633eb6f2030271407f0944d433046c0c870`;
- `docs/images/ghost-ftp-about.png` — SHA-256 `1d1b6487be473e3f59af2620584cf09e2ef3225d30712818a9cb8ca1fa492ca4`.

Those hashes identify the checked-in image objects; they are not proof for a later changed UI/source head. A release-prep or public UI/runtime change must obtain authentic capture evidence from its own exact gated source. The screenshot workflow verifies PNG format, plausible dimensions, visual non-degeneracy, file size and SHA-256, and persists maintained images only under its guarded eligible-branch policy.

## Accessibility and usability

- visible keyboard focus where native controls support it;
- readable contrast in Light and Dark appearances;
- clear destructive-action confirmation when enabled;
- localized user-facing labels through the maintained 24-language catalog;
- no credential or server-secret text in screenshots/documentation evidence.

## Change rule

A desktop UI change is acceptable only when it maps to real product state/capability, stays behind the shared engine/security boundary, remains usable at supported DPI/window sizes, maintains truthful enabled/disabled state, respects localization and privacy, and passes regression tests plus authentic Windows capture.

See [Settings](SETTINGS.md), [Platform parity](PLATFORM-PARITY.md), [Testing](TESTING.md) and [Privacy](PRIVACY.md).
