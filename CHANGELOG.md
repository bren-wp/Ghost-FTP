# Changelog

## 2.1.1-rc.16 — 23 September 2026

- Reworked Ghost FTP around one persistent native application window; Site Manager, Transfers, Preferences, Sync, Cloud Storage, Help, About and Updates now switch inside the main workspace instead of behaving like separate application windows.
- Removed the secondary Tauri terminal pop-out path so terminal work stays docked inside the main Ghost FTP window.
- Replaced the stacked File/Edit/View/Transfer/Server/Bookmarks/Tools/Help navigation with one compact primary sidebar and contextual toolbar actions.
- Removed the language selector from the main header; language selection now lives in Settings only.
- Removed duplicate workspace titlebars and duplicate navigation from Site Manager, Transfer Center, Preferences and About.
- Simplified the main sidebar to Files, Sites, Transfers, Sync & Backup, Cloud Storage, Settings and Help & About, plus one New connection action.
- Kept New Connection, import, consent and other genuinely transient tasks as in-app overlays rather than new native windows.
- Added CI regression guards that reject secondary WebView windows, popup/new-tab navigation and reintroduction of duplicate top-level menus.
- Advanced desktop, Tauri, CLI, agent, runtime, installer, updater and native QA metadata to RC16.

## 2.1.1-rc.15 — 22 September 2026

- Hardened Folder Sync, settings migration, transfer-engine and deep-link startup promises so initialization failures cannot escape silently.
- Added deterministic ghostftp:// listener cleanup and visible registration failure handling.
- Contained saved-site startup failures and surfaced SSH public-key/password clipboard failures.
- Removed lazy-loaded primary/transient application views from the main shell to prevent brief workspace loading/blank flashes during navigation.
- Advanced desktop, Tauri, CLI, agent, runtime, installer, updater and native QA development metadata consistently to RC15.
- Preserved the published RC14 release workflow and immutable RC14 source/tag relationship.
- RC15 remains blocked until exact-head quality, real protocol E2E, Windows/Linux native builds and seven-view visual acceptance all pass.

## 2.1.1-rc.14 — 21 September 2026

- Hardened seven-view Windows native QA so blank/structureless WebView2 frames cannot pass on anti-aliased border colors alone.
- Added quantized-color, luminance, edge-density and duplicate-image checks plus real WebView2 compositor fallback evidence.
- Reordered Preferences General to match the approved reference hierarchy and restored a real Performance card.
- Rebalanced Transfer Center so the real transfer table and professional empty state keep reference-sized workspace area.
- Added keyboard navigation and focus return to the Transfer Center More menu.
- Replaced Site Manager browser-style profile export with a native Save dialog and Rust export command that omits credentials.
- Surfaced native window-control, clipboard, transfer enqueue, preference persistence and shell-integration errors instead of silent no-op failures.
- Preserved real FTP, explicit FTPS and SFTP behavior, keychain credential handling, capability checks and the no-fake-data production policy.
- Bumped Windows/Linux desktop, CLI, agent, updater metadata and release packaging consistently to RC14.

## 2.1.1-rc.13 — 21 September 2026

- Moved Site Manager, Preferences, Transfer Center and About from fixed modal overlays into single-app workspace surfaces.
- Kept New Connection and File Properties as deliberate transient in-app panels with explicit close/cancel behavior.
- Hardened responsive containment so titlebar, action rows, Site Manager columns and dialog buttons cannot overlap.
- Added focused-pane capability reporting so Delete, Rename, Properties, New Folder, Upload and Download disable when the action is not valid.
- Removed the modal loading flash when switching between primary Ghost FTP workspace views.
- Added a UI contract CI gate for single-shell architecture, critical click handlers and popup/new-tab regressions.
- Added a functional in-app Transfer Scheduler with persisted once/daily/weekly schedules and real queue-priority movement.
- Wired Sync & Backup to a real workspace and kept Help/Documentation/Updates inside the main Ghost FTP window.
- Removed obsolete duplicate status-bar code and stray untranslated UI text.
- RC13 remains a development release candidate until quality, protocol E2E, native Windows/Linux build and native visual acceptance are complete.


## 2.1.1-rc.12 — 20 September 2026

- Continued the existing native Ghost FTP React/TypeScript/Tauri/Rust source after RC11 instead of replacing the project.
- Hardened File/Edit/View/Transfer/Server/Bookmarks/Tools/Help popovers with stable overlay stacking and desktop-style menu switching.
- Replaced the misleading Sites toolbar × glyph with an explicit Site Manager affordance.
- Stabilized New Connection at the 752×628 reference envelope with viewport-safe internal scrolling.
- Removed press/entrance transforms from critical production surfaces to reduce WebView2/GTK text/raster flicker under repeated clicks.
- Kept desktop notification permission opt-in and wired the explicit OS permission request to the Preferences enable action.
- Corrected Rust formatting in the native external-link path after the RC12 quality gate exposed it.
- Aligned preview/latest updater templates and release metadata with RC12.
- Preserved real FTP/explicit-FTPS/SFTP behavior, real transfer telemetry and the no-fake-data production policy.
- RC12 remains a pre-release until current quality, protocol, native Windows/Linux build, screenshot and packaging gates are verified.

## 2.1.1-rc.11 — 20 September 2026

- Rebuilt the production UI acceptance geometry from the supplied Ghost FTP references without using screenshots as runtime UI.
- Fixed standalone About, Preferences, Site Manager and Transfer Center surfaces so the File Manager chrome is not rendered beneath them.
- Fixed application menu dropdown layering so File/Edit/View/Transfer/Server/Bookmarks/Tools/Help menus overlay content instead of reflowing Quick Connect.
- Rebalanced titlebar, menu, Quick Connect and toolbar heights for the 1290×852 reference geometry.
- Removed the fixed New Connection height that produced the large dead region in the failed acceptance capture; the dialog now sizes to content with viewport-safe scrolling.
- Tightened responsive compaction from 1120 px through 480 px while keeping key controls available through component-local scrolling.
- Preserved real FTP/explicit-FTPS/SFTP behavior and the production rule that no demo servers or fake connection/transfer state are seeded.
- Bumped desktop, Tauri, CLI, agent, compatibility tooling, updater preview and package naming consistently to RC11.
- RC11 publication is gated on source quality, native Windows/Linux builds and real FTP/FTPS/SFTP E2E acceptance.

## 2.1.1-rc.10 — 20 September 2026

- Closed the final RC9 Rust Clippy blockers in the Ghost FTP CLI.
- Fixed the native Transfer Queue / Server Log desktop layout found by Windows screenshot evidence.
- Added explicit transfer-grid sizing, overflow, truncation, action alignment and log row layout.
- Made Windows native-window screenshot capture a required native-build gate and validated the expected `Ghost FTP` window title.
- Bumped desktop, CLI, agent, compatibility-tool, updater-preview and release packaging versions consistently to RC10.

## 2.1.1-rc.9 — 20 September 2026

- Reorganized repository documentation under `docs/` and removed duplicate root copies while preserving stable runtime source paths.
- Rewrote the main README as a product/marketing landing page with Ghost FTP branding and visual reference screenshots.
- Added the approved reference imagery to `docs/assets/screenshots/` for README and pixel QA use only.
- Removed obsolete RC7/RC8 release workflows and consolidated publication into a single native RC9 workflow.
- Improved release artifact naming with platform, architecture, role and version encoded consistently.
- Added stronger CI checks for frontend, Go and Rust workspace quality, including formatting, tests and Clippy.
- Removed an unused Rust agent-service constant and redundant default dark-theme token block.
- Aligned the default Electric Blue and Ice White tokens more closely with the approved Ghost FTP palette.
- Preserved the canonical 1290×852 1:1 shell geometry and adaptive smaller-window rules.
- Fixed main transfer-queue terminal-state actions: completed rows no longer expose Cancel, skipped/canceled states are explicit, active queue filtering is accurate, and failed transfers can be retried in one action.
- Replaced inline transfer-progress width styling with semantic progress controls and added reduced-motion handling.
- Expanded Preferences with browser layout, hidden-file visibility, remote previews, transfer bandwidth limit, default download folder and default editor controls.
- Added validation for negative transfer throttle values.
- Replaced website glyph/emoji action icons with local Ghost FTP SVG icons and improved website markup, metadata, focus handling and reduced-motion behavior.
- Fixed the website progress animation so its timer stops at completion instead of running indefinitely.
- Made Quick Connect ephemeral by default so a one-off connection is not automatically saved as a Site Manager profile.
- Added an optional SFTP private-key passphrase input to New Connection using the existing native key-auth path.
- Removed the inert FTP Active/Auto selector; FTP/FTPS now truthfully show the native passive data mode instead of presenting a non-functional option.
- Added automated locale-key parity gating and Windows native-window screenshot evidence capture to the RC9 CI path.
- RC9 remains a pre-release pending final native Windows titlebar/pixel, installer lifecycle and real FTP/FTPS/SFTP acceptance evidence.


## 2.1.1-rc.8 — 20 September 2026

- Removed browser-shell compatibility executables from the end-user release path after the visible 127.0.0.1 Chromium/Edge bar regression was reproduced.
- Production GUI releases now come only from the native Tauri/WebView pipeline.
- Fixed encrypted-backup magic-header length compilation error in the Rust backend.
- Fixed Windows RC packaging by using NSIS for prerelease builds instead of invalid MSI prerelease metadata.
- Native build pipeline now emits Windows native portable EXE + NSIS setup and Linux native binary + DEB/RPM/AppImage artifacts.
- Public release publishing is gated on a successful native Windows/Linux build.
- Kept the frameless custom Ghost FTP window, 1290×852 reference geometry and adaptive smaller-window rules.


## 2.1.1-rc.7 — 20 September 2026

- Reworked Site Manager, Preferences, Transfer Center and About into reference-style full application surfaces instead of generic modal cards.
- Added reusable frameless Ghost FTP titlebar/menu chrome and tightened adaptive behavior for smaller windows without removing critical actions.
- Matched New Connection and File Properties closer to the approved reference geometry.
- Added real SHA-256 checksum commands for local and SSH/SFTP paths and recursive chmod where supported.
- Rebuilt File Properties with General/Checksums tabs, chmod matrix, numeric mode, duplicate and open-containing-folder actions.
- Brought compatibility runtime surfaces closer to the same Windows/Linux visual structure used by the native source.
- Rebuilt Windows x64 and Linux x86-64 compatibility artifacts from RC7 source and refreshed package metadata.

## 2.1.1-rc.4 — 20 September 2026

- Hardened native startup against corrupted profile metadata and SQLite state; broken stores are preserved for recovery instead of causing a startup panic.
- Kept the frameless 1290×852 reference window and 480×600 minimum adaptive viewport contract.
- Revalidated compatibility runtime security boundaries, local filesystem operations, state recovery, branding scan and package metadata.
- Regenerated Windows/Linux compatibility artifacts and release checksums from the updated source tree.


## 2.1.1 RC3 — 20 September 2026

- Added true ephemeral Quick Connect sessions in the native Rust/Tauri source.
- Added split FTP/FTPS/SFTP Quick Connect protocol selection in the reference titlebar.
- Persisted Site Manager favorites, bookmarks, tags, folders and last-used metadata.
- Made native transfer retry count configurable and re-applied persisted engine settings at startup.
- Corrected Preferences Cancel/Reset semantics and wired Shell Integration to the real per-user PATH integration.
- Centralized React release metadata in the RC3 line so later release candidates can update version/build data from one source.
- Reached 185-key parity across every advertised non-English native dictionary.
- Removed inline language-switch JavaScript from all 14 localized website pages.
- Rebuilt Windows/Linux compatibility fallback artifacts and the Debian fallback package.
- Re-ran local runtime security/filesystem QA and Go test/vet checks.

## 2.1.0 — 2026-09-19 source/release-candidate hardening

- Continued the existing Ghost FTP v9 source tree rather than replacing it.
- Preserved the native React/Tauri/Rust protocol engine and verified the Tauri main window is frameless with custom decorations disabled.
- Reduced the native minimum window width to 480 px and added compact CSS breakpoints down to the required narrow desktop sizes without intentionally removing toolbar actions.
- Hardened native profile persistence with staged writes, a last-known-good backup and corrupt-file preservation.
- Reworked About/Updates source so it does not claim "up to date" before a real updater check and reports the current platform.
- Rebuilt the Go compatibility Windows/Linux hosts after removing fake connection-success and simulated transfer-progress behavior; fallback networking is now explicitly TCP reachability only.
- Added custom titlebar drag support to the Windows compatibility host and setup source.
- Hardened setup ordering so future steps cannot be clicked, embedded the full commercial EULA, added custom installation-folder selection/validation, improved upgrade rollback and corrected publisher metadata.
- Added/updated release, security, privacy, installation, uninstall, support and QA documentation.

At that point, the package was **not labelled FINAL** because the native Tauri build and several platform acceptance gates had not yet been completed.

### RC7 parity and delivery pass
- Matched the compatibility New Connection surface to the measured 752×628 reference envelope.
- Added the reference-scale Site Manager identity block and refined full-window secondary surfaces.
- Added local Ghost FTP mountain artwork to the About hero without using a reference screenshot as runtime UI.
- Executed automated render QA at all nine required viewport sizes with no whole-window horizontal overflow and no page errors.
- Rebuilt Windows x64 and Linux x86-64 compatibility artifacts from RC7 source.
- Prepared repository bootstrap automation for the official `bren-wp/Ghost-FTP-Premium` GitHub repository so the current source can be reconstructed and committed by GitHub Actions without relying on this sandbox's blocked outbound Git transport.
