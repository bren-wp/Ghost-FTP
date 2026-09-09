# Ghost FTP desktop reference UI

This document defines the maintained visual and interaction contract for **Ghost FTP 1.1.8 Stable** and later compatible desktop releases.

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

**Classic Light is the fresh-install, missing-state and invalid-state primary appearance in 1.1.8.** An explicitly persisted Dark selection remains respected on Windows. Appearance is one canonical decision rather than a collection of overlapping cosmetic switches.

The 1.1.8-maintained palette deliberately avoids pure white as the dominant application surface. Classic Light uses cool neutral layers, while Dark uses a restrained navy/charcoal hierarchy so long file-management sessions remain readable without flattening panels into one undifferentiated background.

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
| Success | `27, 127, 75` (`#1B7F4B`) |
| Warning | `154, 103, 0` (`#9A6700`) |
| Danger | `198, 40, 40` (`#C62828`) |
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
| Success | `74, 215, 155` (`#4AD79B`) |
| Warning | `242, 186, 85` (`#F2BA55`) |
| Danger | `255, 104, 120` (`#FF6878`) |
| Selection | `32, 47, 80` (`#202F50`) |

Theme data is local source data only. No remote stylesheet, font service, theme API, analytics endpoint or browser runtime is loaded.

On Windows, an explicit appearance choice is applied on the next start so title bar, menus, native controls, headers and owner-drawn controls are initialized consistently. Fresh/fallback state resolves to Classic Light before the native control tree is created.

The Linux graphical frontend uses Classic Light as the canonical palette. It deliberately does not expose a runtime appearance toggle until complete native switching can be implemented without mixed-state rendering or race-prone global palette mutation.

## Native dialog contract

Ghost FTP-owned Windows dialogs must visually belong to the active application appearance and must not terminate the application message loop when they close.

Maintained rules are:

- only the main desktop window owns process-level `WM_QUIT`/`PostQuitMessage` lifecycle;
- auxiliary Prompt, Option, Settings and information-card windows close only their own bounded modal loop;
- application-owned dialogs use the active Ghost FTP owner window when available and temporarily disable that owner while a modal decision is pending;
- dialog client dimensions are converted to true outer Windows dimensions so title bars and frames do not clip footer controls;
- controls, fonts and geometry scale from the current Windows DPI;
- dialog title bars follow the active Light/Dark state on supported Windows builds;
- Diagnostics uses the Ghost FTP information shell rather than an unrelated bright stock surface during a Dark session;
- action labels remain caller/localization-owned, so a selected locale must not fall back to English `Cancel` on an otherwise localized dialog.

Closing **Nova mapa**, **Preimenuj**, **Postavke**, **Dijagnostika** or **O programu** with the title-bar X must close that surface only. It must not close the whole Ghost FTP application.

## Settings surface

Windows Settings is one application-owned modal surface rather than a wizard-like chain of unrelated prompts. It presents the current canonical settings together:

- appearance;
- parallel transfer count;
- connection timeout;
- automatic retry count;
- retry delay;
- destination conflict policy;
- delete confirmation.

Numeric values are validated against the same `internal/config` bounds used by persisted settings normalization. Invalid input keeps Settings open, shows localized corrective text and restores focus to the invalid field instead of discarding the rest of the user's pending choices.

**OK** returns one complete candidate settings value to the desktop layer and the existing typed engine persists it. **Cancel** or title-bar **X** discards the pending dialog values and returns to the main workspace. The serialized settings schema is unchanged; compatibility mirror fields are not promoted into duplicate controls.

## Menu and action contract

Menu/button surfaces are alternate entry points to the same canonical commands. They must not create second implementations of connect, transfer, rename, delete, Site Manager or settings behavior.

Actions are enabled from real state. A disabled operation must remain disabled regardless of whether the user reaches it through a button, menu, list gesture or keyboard shortcut.

Connection attempts have a visible cancel/disconnect path; users must not be trapped behind a long timeout. Stale asynchronous callbacks are invalidated by connection-generation state so an old attempt cannot repaint or attach itself to a newer session.

## Site Manager left navigation

The Windows Site Manager left panel is the canonical compact profile navigator. It is deliberately not a second dashboard or a duplicate application menu.

Its maintained behavior is:

- **Quick Connect** is always the first navigation row and remains the no-profile connection path;
- saved profiles use a compact two-line presentation: profile name on the primary line and `PROTOCOL · host` on the secondary line;
- duplicate profile names remain distinguishable by their protocol/host secondary line;
- the selected row uses the canonical selection surface plus a restrained accent rail, and keyboard focus remains visibly indicated;
- the list keeps native ListBox keyboard navigation and selection semantics rather than replacing them with decorative custom widgets;
- Classic Light uses the normal native list treatment; `DarkMode_Explorer` is applied only when the active Windows appearance is Dark;
- usernames, passwords, passphrases, private-key paths, fingerprints and local/remote profile paths are not rendered in the left navigation;
- Save, Delete and Connect remain the existing canonical Site Manager actions in the detail pane instead of being duplicated as sidebar shortcuts.

The navigation row model is independent of Win32 drawing so regression tests can verify ordering, disambiguation and privacy-safe content without relying on screenshot pixels alone.

## Icons

Windows toolbar/buttons use operating-system Segoe Fluent/MDL2-compatible glyph resources and the native drawing layer. File panes use the operating-system file image list where available.

Rules:

- no remote icon CDN;
- no external icon-font dependency solely for toolbar decoration;
- one semantic glyph per action;
- destructive actions must not reuse a positive/transfer glyph;
- disabled icons/text must remain readable in both maintained Windows appearances;
- packaging variants must not ship different workspace icon sets.

The product logo comes from the same canonical executable resource used by production Setup/Portable packaging. Documentation renders the repository-local `build/icon.png` asset rather than loading a separate remote logo or icon service, so documentation and runtime branding cannot drift to a second externally maintained source.

## Quick Connect

Quick Connect uses the normal connection path and normal validation. It is not a lightweight or less-secure alternate connector.

The fresh/default protocol is **explicit FTPS on port 21** on both active platforms. Plain FTP remains available for legacy compatibility but must be selected explicitly. SFTP remains available with password or private-key authentication. A failed FTPS negotiation is never silently downgraded to FTP.

On sufficiently wide windows the main connection fields fit on one row. At narrower supported widths the controls reflow so fields do not overlap or become unusable. SFTP private-key/passphrase controls are enabled only for the relevant protocol state.

## Profiles and credential-consent UX

Saving profile identity/path data and saving credentials are distinct decisions. A password/private-key passphrase is persisted only after explicit consent through the maintained privacy prompt. The main Save Profile flow and Windows Site Manager use the same consent semantics and localized catalog text.

If consent is declined, the non-secret profile can still be saved while newly entered credentials are not persisted. Existing credentials that no longer belong to the changed endpoint/account/private-key identity are removed rather than silently transferred to a different identity.

## Local and Remote panes

Both panes use the shared engine and filesystem/remote validation layers.

The local visual columns are Name, Size, Type and Modified. The remote pane adds Permissions when the server provides real permission metadata.

Permissions must never be fabricated. Servers without usable POSIX-style mode information produce an empty Permissions cell rather than a guessed value.

Sorting, double-click navigation, keyboard actions and selection restoration must preserve the item/model identity used by rename, delete, upload and download operations.

## Transfer queue

Pause, resume, cancel, retry and clear-finished actions operate through the canonical transfer manager. Visual progress, speed, ETA or byte counts may be shown only when backed by real transfer state.

The queue must not invent values simply to resemble another FTP client. Transfer refresh is event/state driven; unchanged transfer state should not trigger needless whole-window redraw.

## No duplicated options

The UI may expose one control per canonical behavior. Compatibility fields in persisted JSON are not justification for duplicate controls. Destination conflict handling, for example, is represented by one conflict-policy choice even though legacy mirror fields remain for backward-compatible state migration.

Appearance follows the same rule: one Dark/Classic Light decision on Windows, not separate switches for window background, panels, lists, icon colors and accent colors.

## Localization

English is the default/fallback language. The maintained catalog contains 24 selectable local languages.

Appearance labels/help and credential/privacy prompts are local strings and do not require a translation service. File actions, dialogs, menus, status text and settings must not silently replace the selected locale with hardcoded developer text.

The Linux terminal `language` command advertises its accepted language codes from the same canonical 24-language registry used by runtime validation so help text cannot drift to an older subset.

## Authentic screenshot evidence

Repository screenshots under `docs/images/` are generated from the **real production Windows x64 Portable executable** by `.github/workflows/ui-screenshots.yml`.

The workflow:

1. disables Go telemetry;
2. builds the production Windows package;
3. starts the real Portable executable;
4. captures the native main workspace, Site Manager, Settings and About windows;
5. verifies PNG signature, plausible dimensions, visual non-degeneracy, file size and SHA-256;
6. publishes the authentic captures as workflow artifacts;
7. persists all four maintained repository images only when the capture commit is still the eligible branch head.

The complete maintained repository set was regenerated by authentic UI workflow run **#201** from source commit `a1e9635f5724ea8b53afca9830f28f7fa9159798` and persisted by screenshot commit `29f3a9a069df37107772265987ecfd251b645c3e`:

- `docs/images/ghost-ftp-main-workspace.png` — 976×696, 45,710 bytes, SHA-256 `659caccf3fab3e9add23424e45709f541c902219ba5212f6287ab5ed25c8cb6e`;
- `docs/images/ghost-ftp-site-manager.png` — 920×610, 23,597 bytes, SHA-256 `63e9292530030afab7a95b21788d9ec1da80b6f7bca1ba0ce8a132fd665602a9`;
- `docs/images/ghost-ftp-settings.png` — 776×639, 20,653 bytes, SHA-256 `b46b8c9c0730e96b1a0ed9ba54e84633eb6f2030271407f0944d433046c0c870`;
- `docs/images/ghost-ftp-about.png` — 776×499, 16,693 bytes, SHA-256 `1d1b6487be473e3f59af2620584cf09e2ef3225d30712818a9cb8ca1fa492ca4`.

### Main Workspace

![Ghost FTP Main Workspace](images/ghost-ftp-main-workspace.png)

### Site Manager

![Ghost FTP Site Manager](images/ghost-ftp-site-manager.png)

### Settings

![Ghost FTP Settings](images/ghost-ftp-settings.png)

### About

![Ghost FTP About](images/ghost-ftp-about.png)

The screenshot refresh changed no Ghost FTP runtime source: the branch change preceding the capture was limited to the authentic evidence workflow itself. Future public UI/runtime changes must regenerate this evidence from their own exact gated head; these hashes must not be carried forward as proof for changed UI code.

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
