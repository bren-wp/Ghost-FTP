# Ghost FTP — Implemented Features

This document describes what is implemented in the current Ghost FTP 2.1.1 RC18 source. It is intentionally separated from future ideas so the repository does not present planned work as finished functionality.

## Connection and profile management

Implemented:

- FTP, FTPS and SFTP connection profiles.
- Quick Connect for temporary sessions without creating a permanent Site Manager entry.
- Saved profiles with host, port, username, remote path and connection metadata.
- Site Manager favorites, folders, tags, bookmarks and recent-server metadata.
- Private-key authentication paths for SFTP.
- OS credential/keychain storage for profile passwords and private-key passphrases where supported.
- Host-key verification flow for SSH/SFTP.
- TLS/FTPS verification through the native protocol engine.
- Test Connection flow.
- Protocol-default port behavior.
- Profile import paths for common server/client formats where supported by the native importers.

## File management

Implemented:

- Dual-pane local/remote browser.
- Local-only and server-focused layouts.
- List/details/grid presentation modes.
- Compact and comfortable density.
- Hidden-file visibility.
- Name/size/modified sorting.
- Upload and download.
- Folder upload/download.
- New folder.
- Rename.
- Delete.
- Duplicate.
- Open containing folder where supported.
- File Properties.
- SHA-256 checksum support for local and SSH/SFTP paths.
- Numeric chmod and owner/group permission display where the backend can provide it.
- Recursive chmod for supported local/SFTP paths.
- Remote image previews as an opt-in setting.
- Context-menu and keyboard interaction paths.

## Transfers

Implemented:

- Concurrent transfer queue.
- Pause/resume for supported active transfers.
- Pause all / resume all.
- Cancel active/queued transfers.
- Retry failed transfers.
- Retry-all action for failed transfers.
- Queue, Failed and Completed views.
- Correct Completed, Skipped, Canceled, Failed, Paused, Queued and Transferring labels.
- Semantic progress controls and percent display.
- Configurable concurrency.
- Configurable retry count.
- Global bandwidth throttle.
- Overwrite / skip / rename conflict policies.
- Per-file conflict prompts with optional remembered behavior.
- Preferences UI for browser layout, hidden files, remote previews, transfer speed limit, default download folder and default editor.

- Desktop completion/failure notifications.
- Delta synchronization support on compatible Ghost FTP Agent paths.
- Transfer engine settings restored at startup.

## Synchronization and analysis

Implemented:

- Folder synchronization workflows.
- Directory comparison/diff tooling.
- Duplicate-file discovery.
- Disk usage / treemap tooling.
- Background Ghost FTP Agent architecture for controlled machines.
- Agent pairing and secure protocol components.
- Fleet/search surfaces present in the desktop source.

## Terminal and productivity

Implemented:

- Integrated terminal surfaces.
- Configurable terminal font, theme and scrollback.
- Copy-on-select.
- Inline command/history suggestions with process-memory-only history and sensitive-command filtering.
- Command palette.
- Keyboard shortcut settings and shortcut reference.
- Snippets.
- Variables and reusable command/productivity tools.
- Shell integration path management where supported.
- File-association control is intentionally locked unless installed by a production package.

## Preferences

Implemented:

- Dark/light and named themes.
- Ghost-branded palettes.
- Accent handling.
- Interface density.
- Thumbnail/preview control.
- Transfer concurrency.
- Retry count.
- Delta synchronization.
- Overwrite policy.
- Prompt-before-overwrite.
- Default SFTP port.
- Notification controls.
- OS keychain requirement display.
- Privacy/no-tracking controls.
- Update checking and installation flow.
- Language selector.
- Preferences Cancel restores the captured settings snapshot instead of silently keeping preview changes.
- Reset restores the Ghost FTP defaults and reapplies live transfer-engine values.

## Updates and lifecycle

Implemented:

- Signed-update configuration through the desktop updater.
- Update-check status.
- Download/install flow.
- Restart-to-complete flow.
- Update policy documentation.
- Windows NSIS release packaging.
- Native Windows portable executable.
- Linux native executable, AppImage, DEB and RPM packaging.
- Uninstall documentation and platform removal paths.

## Languages

Advertised and implemented selector coverage:

- English
- Hrvatski
- Deutsch
- Français
- Español
- Italiano
- Português
- Nederlands
- Polski
- Slovenščina
- Srpski
- Bosanski
- Македонски
- Shqip

The native non-English dictionaries are tracked against the same canonical key set. Linguistic/visual review remains part of release QA.

## Privacy and security

Implemented architectural controls include:

- No required analytics/telemetry.
- Sensitive profile secrets separated from ordinary profile JSON.
- OS-protected credential storage where supported.
- CSP on the desktop webview.
- Signed update verification path.
- SSH/TLS identity verification.
- Corrupt profile/database recovery instead of hard startup failure.
- Restricted developer compatibility API and mutation authorization.
- No production dependency on a browser-host `127.0.0.1` GUI wrapper.

## Current RC18 hardening

RC18 retains the single-window RC16/RC17 architecture and further hardens:

- Terminal transfer rows no longer show a meaningless Cancel action after completion.
- Skipped and canceled transfers have explicit labels.
- The main Transfer Queue tab now represents active queue items rather than duplicating completed/failed history.
- Retry All is available for failed transfers.
- Clear controls are disabled when there is nothing to clear.
- Transfer progress uses a semantic `<progress>` element instead of inline width styling.
- Live transfer speed and ETA are derived from real backend progress byte deltas with stale-sample handling.
- Transfer scheduling remains available inside Transfers instead of a duplicate top-level workspace.
- Terminal command suggestions never persist shell history to WebView storage.
- Notification history is session-only and all toast/error text passes through credential redaction.
- Startup removes legacy persisted terminal/notification history left by older release candidates.
- Reduced-motion support is applied to desktop UI transitions.
- Website loading animation stops when complete instead of running an interval forever.
- Website loading animation respects `prefers-reduced-motion`.
