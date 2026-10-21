> Active Ghost FTP source: **0.99.1** Preview; last published edition: **0.99.0**.

> Active development: Ghost FTP **0.99.0** Preview, preceding published **0.95.53**.

> Security development preview: Ghost FTP 0.95.53. Published predecessor: v0.99.0. This source is unreleased; v0.95.52 artifact recovery is pending.

> Ghost FTP **0.95.45** (development preview), following published **v0.95.43**. Distribution status is verified from GitHub Releases; unsigned/mobile and notarization limitations remain explicit.

# Ghost FTP — Implemented Features

## v0.95.45 Transfer Center selector modularization — development preview

- Extract status counts, all/active/completed/failed filters, search, direction and hour/day windows and bounded transfer-log projection from the desktop Transfer Center into a pure TypeScript module.
- Preserve real backend transfer IDs, status and queue order so pause, resume, retry, cancel and history actions still target their existing IPC implementation; no new completion inference or vault behavior.
- Node regression checks cover positive/negative filters, timestamp boundaries, original transfer identity, history immutability and a synthetic 10,000-entry inventory.
- **Not accepted yet:** actual user-study results, full site-based grouping, 10MB editor test, real server proxy/IPv6 E2E, production signing and the 75-screen installed UI matrix remain separate work.


## v0.95.43 Honest Preview release classification — development candidate

- Refuse stable release publication while Authenticode, Play upload acceptance, notarization and upgrade/rollback proofs are absent. The source-to-GitHub-release mapping now marks future Preview builds as GitHub prereleases, never automatic stable latest.
- Preserve the existing six exact-SHA CI gates, immutable tags and all-or-nothing signed desktop updater manifest enforcement. Negative regression tests check missing, unknown and falsely stable release channels.
- **Not a production signing certification:** the 75 UI-reference real-screen audit and actual Windows/Linux/Android/macOS installation lifecycle remain unverified.


## v0.95.42 Android upload signing metadata alignment — development preview

- Consolidate desktop OAuth per-login PKCE state binding and token diagnostic redaction; an altered/missing or duplicate callback state must not authorize a login.

- Retain the existing protected Play upload signing workflow merged in #189; do not duplicate it.
- Advance canonical source and Android versionCode 954200, aligning Rust/Tauri/Go/frontend/installers after published v0.95.41.
- Existing test/build CI remains unsigned for production Android while the isolated preview package stays independently signed. Play Console acceptance, actual keystore use and secure macOS FTPS/SFTP are not established.


## v0.95.41 Bounded HTTP directory inventory and duplicate rejection — security preview

- Limit HTTP autoindex response bytes before parsing, reject incomplete/unrecognized HTML index pages and malformed links.
- Reject duplicate, path-escaping, malformed Unicode and oversized HTML/JSON listing entries rather than reporting a partial successful source tree.
- Add regression tests including valid Apache/nginx directory listings, Unicode filenames, malicious JSON rows, oversize bodies and duplicate names.
- Live Apache/nginx/IIS compatibility and all six final-SHA CI families remain acceptance gates; no complete FILE-03 certification is claimed.


## v0.95.40 Fail-closed encrypted backup and OS vault restore — security preview

- Export refuses unreadable OS credential vault entries and a missing credential referenced by the keychain manifest; optional OAuth entries remain optional when absent.
- Encrypted import validates profile metadata and restores credentials via OS vault before staging local state. OS vault write failures return errors, with best-effort reversal of any modified credentials.
- Add fault-injection tests for locked vault, partial secret writes and existing profile preservation. No production signing or manual OS-vault acceptance has yet been established.


## v0.95.39 WebDAV and HTTP listing fail-closed hardening — preview candidate

- Reject malformed or truncated WebDAV multistatus XML, unexpected HTTP status, missing response hrefs and invalid listing bodies rather than treating partial inventories as complete.
- Reject malformed nginx autoindex JSON instead of returning a falsely empty folder; valid empty arrays remain supported.
- Targeted Rust regression tests cover failure handling. Full live WebDAV failure-injection and exact-head cross-platform release acceptance are still required before publication.


## v0.95.38 Transaction-safe SQLite folder sync and strict source inventories — development candidate

- Mirror/additive sync refuses unreadable source or destination directory inventories instead of silently treating missing folders as empty.
- SQLite sync index reads fail closed; source snapshots replace all rows for one pair in a single transaction, preserving other pairs on write failures.
- Mirror deletion safety warnings cannot advance the last fully synchronized timestamp or replace the saved source history.
- Rust regressions cover transaction rollback, size validation, cross-pair isolation and unreadable directory scans. Existing OS-backed password storage, live watcher, safe CMD/.NET scripts and PHP/JS/HTML/CSS editor remain unchanged.
- Bulk-transfer benchmarks, 59-screen pixel parity and independent cloud provider production verification remain outstanding.


## v0.95.37 SQLite folder-sync migration and built-in source editor — preview

- Watched sync settings use the local SQLite database transactionally with guarded one-time import of previous JSON settings and explicit corrupt-data failure.
- Remote file context menu opens a built-in text editor. PHP, JS, HTML and CSS syntax highlighting, Ctrl+F search, Ctrl+S save, indentation and closing brackets are available for UTF-8 files up to 2 MiB.
- Editor IPC is edit-session scoped, and the background remote upload must confirm success separately from the local save.
- Windows native keyring/DPAPI and Linux native secret storage remain authoritative for passwords: SQLite is not encrypted merely by being SQLite.


## v0.95.36 Security, live folder sync and script export — preview

- Enforce OS keyring credential storage at the profile JSON write boundary; never persist new plaintext passwords/passphrases.
- Live watcher plus poll backup: source index is updated only when every transfer succeeds; internal partial files are never mirrored.
- External-editor saved-file watcher is bounded and retries failed uploads, with private temporary workspaces.
- Sync & Backup can copy credential-free one-shot CMD and PowerShell sync scripts or a PowerShell/.NET live FileSystemWatcher script for safe additive push pairs. Mirror, exclusion mismatch and unsupported modes are explicitly refused.
- S3-compatible (including B2), GCS and WebDAV adapters already exist; independently authenticated vendor E2E validation remains outstanding.
- Built-in syntax highlighting, auto-bracket completion and in-editor find are not yet fully implemented.


## v0.95.35 Windows/Linux guarded transfers and drop safety — development candidate

- Preserve existing local backup files; use no-replace file moves, refuse symlinks and special filesystem nodes when promoting downloads.
- Inventory folder uploads first; reject unsupported entries, unsafe/non-UTF-8 names and oversized trees rather than silently omitting files.
- Bind cross-pane drag/drop to the originating FTP/FTPS/SFTP session; reject stale or malformed drag payloads.
- Add regression tests and audit traceability. Pixel-perfect 75-screen fidelity and comprehensive FileZilla superiority remain unverified.


## v0.95.35 FTP LIST and SSH host verification safety — development candidate

- Bound FTP/FTPS directory inventories and reject server responses that exceed entry, raw-byte or filename-byte limits.
- Inspect entire FTP LIST inventories before overwrite/existence decisions so malformed or duplicate rows fail closed.
- Serialize SSH host-key decisions and require deliberate acknowledgement before trusting changed fingerprints.
- Native 1:1 screenshot comparison and full FileZilla feature parity are not yet proven.


## v0.95.33 Windows/Linux SFTP safety, privacy and transfer usability — development candidate

- Fail closed on invalid, duplicate or excessively large SFTP directory listings.
- Reject ambiguous symbolic-link and special-file transfers rather than converting them into ordinary files.
- Protect same-batch duplicate basenames; redact backend diagnostics in file-manager and transfer displays.
- Improve semantic transfer progress and consistent time filtering. Actual native UI pixel parity remains unverified.

## v0.95.32 Windows/Linux SFTP recursive-delete hardening — development candidate

- SFTP recursive deletion inventories targets before any destructive call and refuses unsafe/duplicate nodes or exceeded entry/depth/path limits.
- Symlink targets are protected using LSTAT, and node kinds are rechecked before individual deletions.
- Unit regressions and a real OpenSSH preservation test cover the failure mode. Remote SFTP cannot guarantee atomic multi-file deletion.

## v0.95.31 Windows/Linux local directory resource limits — development candidate

The local file browser and sync inventory now fail closed if a single directory exceeds 100,000 entries or 16 MiB cumulative UTF-8 filename bytes. Returning incomplete inventories could otherwise cause incorrect sync comparisons. This is source-level hardening; exact-commit native build and large-share acceptance remain pending.

## v0.95.30 development candidate — Windows/Linux folder sync safety

Filesystem watcher bursts coalesce into one pending reconcile rather than exhausting memory; a five-second debounce cap prevents endless deferral; Mirror roots refuse parent/dot traversal. Ignore/exclude wildcard matching runs without recursive exponential backtracking. Best-effort aged delta-transfer temp cleanup skips symlinked files/directories and special nodes and caps scanning at 100,000 entries/64 levels. This does not establish descriptor-relative, race-free containment when concurrent filesystem changes occur, or manual installed-build acceptance.


## 0.95.29 Windows/Linux safety, connection ordering and responsive transfer UI — development candidate

Local rename now uses OS-enforced no-replace operations on Windows and Linux, preventing a collision from silently overwriting an existing file, directory or symbolic link. Linux chmod now uses a no-follow file descriptor and changes permissions on that same inode, avoiding final-component symlink-swap races. React connection management honors the latest intended focused session even if an older handshake completes later, ignores stale health checks/profile lists, and coalesces Agent Bridge focus IPC requests. File browsers subscribe to a stable transfer activity count rather than recalculating the full transfer list at every progress event. Their shared pane now invalidates outstanding directory reads on navigation and clears previous-directory rows/selections before loading, preventing late responses or failed listings from exposing stale file actions. Exact-source Rust/Windows/Linux/FTP E2E CI and manual device UI acceptance are required before asserting release readiness.


## 0.95.28 candidate — macOS FTP control-character safety

The native macOS FTP codec rejects every ASCII C0 control code (including TAB, ESC, BS, CR, LF and NUL) and DEL in command arguments, remote paths and printable USER/PASS operands before sending a control-channel command. Previously only CR, LF and NUL were rejected. Real Unicode filenames and ordinary spaces remain accepted. Swift regression tests cover rejected control sequences and allowed international filenames. This is scoped FTP-only hardening: macOS FTPS/SFTP engines, Developer ID/notarization and full desktop parity remain outstanding (issue #157).


## 0.95.27 development candidate — bounded FTP recursive deletion

Windows/Linux FTP and FTPS directory deletion performs a complete preflight inventory before any DELE or RMD. The preflight now rejects duplicate names/paths, an incomplete hidden-file listing, unsupported LIST formats, nesting deeper than 64 directory levels, more than 25,000 entries, or cumulative listed target-path sizes beyond 8 MiB. Successful small-tree deletion retains the previous bottom-up order. These guards reduce server-induced runaway traversal and excessive allocation; they do not make remote recursive delete reversible or guarantee FileZilla parity.


## 0.95.26 development candidate — FTP command-path framing safety

Windows/Linux FTP and explicit FTPS now reject empty or ASCII-control-bearing pathname operands (CR, LF, NUL, TAB, DEL and other ASCII controls) before LIST, existence checks, RENAME, DELETE, MKD, SITE CHMOD and low-level SIZE/RETR/STOR/APPE transfers (including resume paths). Unicode filenames and ordinary spaces remain supported. This guards control-channel command framing; it is not a remote sandbox authorization guarantee.

## 0.95.24 baseline — cross-platform reliability (historical published source)

macOS native FTP now negotiates an authenticated passive directory listing and supports fail-closed Unix LIST fallback for servers that explicitly reject MLSD. The Files editor supports navigable folders, creating directories (MKD), confirmed deletion of files (DELE), and confirmed removal of **empty** folders (RMD). Untrusted remote file sizes are formatted without converting UInt64 values beyond Int64 capacity. These macOS additions are for unencrypted FTP only: macOS FTPS, SFTP, signed/notarized distribution and full FileZilla parity are **not implemented** by this patch.

Windows/Linux FTP and explicit FTPS now bound the sum of TCP attempts across DNS addresses. Android uses existing premium tokens but limits contextual FTP buttons and status to relevant workspaces. The 16 Android 1:1 screen acceptance tests have not passed; do not claim pixel-perfect or full visual parity.



## Post-v0.95.18 development: safer FileZilla saved-site import (PR candidate)

The existing FileZilla XML importer now brings saved folder breadcrumbs into Ghost FTP's Sites grouping, accepts a trailing text site label as exported by FileZilla, and retains mandatory explicit FTPS mapping without importing credentials. The selected-profile preview and save UI remain the same. Bounded input, nesting, site count and DTD/malformed-entity rejection prevent incomplete or oversized imports from being treated as success. This work is **unreleased** until a reviewed PR passes exact-SHA CI; refer to `docs/qa/FILEZILLA_PARITY.md` for evidence and limitations.



> Current development cycle: **0.95.30**; latest verified public release: **0.95.29**. Historical sections remain version-specific.

This document describes what is implemented in the current Ghost FTP 0.95.53 development source. Previous canonical release: **0.99.0**. It is intentionally separated from future ideas so the repository does not present planned work as finished functionality.

## Cloud storage — 0.93.0 candidate

Windows/Linux already implement authenticated S3-compatible object storage, Google Cloud Storage, Azure Blob, WebDAV, Google Drive, OneDrive, Dropbox and Box adapters. Custom S3/Azure endpoints now require HTTPS without URL-embedded credentials. Android and macOS Cloud storage workspaces transfer individual objects through temporary provider-signed HTTPS PUT/GET URLs (S3/R2/B2, GCS, Azure SAS), without persisting bearer links. Android stages downloads before writing a user-selected document; macOS stages before replacing a selected file. Both limit objects to 1 GiB and block HTTP redirects. Neither platform has implemented OAuth Drive account browsing or general S3 bucket navigation.

## macOS Keychain write failure safety (0.92.2)

Existing saved-site passwords are updated in place instead of removed before inserting replacements. A concurrent insert collision retries the update without destructive deletion. If writing or deleting a password fails, the Sites editor preserves the prior profile state and presents an error rather than reporting a successful save/deletion. Six isolated Keychain status-path unit tests cover these transitions. The macOS application remains a Preview with intentionally unsupported FTPS/SFTP transfers.

## macOS backup restore — credential isolation (0.92.1)

Saved-site JSON restores reject existing profile UUIDs when the imported protocol,
server hostname, port or username differs from the saved site. This prevents an
import from redirecting an existing UUID-bound macOS Keychain password to another
connection. The whole backup is rejected without partial profile updates;
same-identity display-name and reliability-setting updates remain supported.

## macOS Sites — advisory duplicate discovery (0.92.0)

The SwiftUI Sites list can filter possible duplicate saved connection endpoints and account identities. The Files connection editor warns about other saved entries matching protocol, DNS-case-insensitive host (ignoring a final dot), port and case-sensitive username. Matching does not touch Keychain or modify stored profiles; empty/incomplete records are skipped. Swift XCTest covers these rules. This is macOS saved-sites parity, not macOS encrypted FTPS/SFTP protocol parity.

## Windows/Linux Site Manager (0.91.0)

Advisory duplicate discovery highlights saved profiles that share protocol, normalized host, port and case-sensitive username, while retaining separate credentials and metadata. The Duplicates filter and count are implemented in Windows/Linux, while a corresponding advisory filter and editor warning now exist on macOS 0.92.0. Android does not currently maintain an equivalent persistent saved-profile list.

## 0.91.1 cross-platform maintenance

Windows/Linux Site Manager prevents switching away from unsaved edits and offers Cancel; Android masks URL credentials/account names in connection errors; macOS keeps a recovery copy of unreadable profile data and rejects imported backups with duplicate UUIDs. No platform acquires previously unsupported transfer protocols from these fixes.

## Cross-app shared actions

Implemented on Windows, Linux and Android; macOS now follows the same primary workspace model while protocol parity continues incrementally:

- FTP, explicit FTPS and SFTP connect/disconnect.
- Remote folder listing and refresh.
- Upload and download.
- Remote folder creation.
- Remote Rename.
- Remote Delete for files; Android also safely falls back to empty-folder removal without recursive deletion.
- Guarded destructive actions and remote-root protection.
- Release-gated interaction checks so exposed Android actions are backed by real handlers rather than static UI.

Desktop retains additional productivity, analysis, terminal and agent workflows that are not falsely presented as completed Android features.

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

## 0.20.1 patch hardening

- Transfer worker registration is race-safe when a very fast worker completes before its `JoinHandle` reaches the task map.
- The task-map lock is acquired before `tokio::spawn`, so an immediately scheduled worker cannot reach cleanup before its `JoinHandle` is registered.
- Stable-channel Android publication remains gated on a persistently signed `com.ghostftp.android` APK; the preview package is CI evidence only.

## 0.20.0 hardening retained

- Desktop transfer queue/history state is persisted in SQLite without passwords, tokens or private-key material.
- Transfer Center exports the real persisted transfer ledger to CSV through Rust IPC; exported path cells are spreadsheet-formula neutralized and raw backend error strings are excluded.
- Interrupted transfers restored after restart/update are shown as recovery/error rows rather than falsely remaining active.
- Recovered retries reconnect through saved profiles and the OS credential store, while the first cross-process retry restarts from byte zero when file identity cannot be proven.
- Runtime pruning bounds terminal transfer history while preserving all active transfer rows.
- FTP and explicit FTPS pause/resume use cooperative bounded chunks, ABOR cleanup and final-size verification with safe full-restart fallback when a server persists the wrong byte count.
- Real FTP and explicit FTPS E2E coverage verifies pause/resume, cancellation, control-channel reuse and I/O-failure cleanup; real OpenSSH SFTP E2E separately verifies the non-zero-offset seek/read/write primitives used by production byte-range resume.
- Android uploads/downloads use staged replacement so incomplete transfers do not replace known-good targets.
- Android lifecycle/disconnect cancellation propagates through upload, download, delete and folder creation.
- Android upload streams are closed at the controller ownership boundary on all early-cancellation paths.
- Android SAF persistence uses valid READ grant modes while the picker still requests persistable document access.
- Stable Android signing continuity is fail-closed once a canonical stable APK exists.

## 0.19.0 hardening retained in 0.20.0

- Recurring transfer schedules honor their configured start date and wait for the initial backend snapshot.
- Concurrent native connection attempts for the same profile are deduplicated.
- Native OS notification and updater diagnostic text is credential-redacted.

0.19.0 retains the single-window architecture and further hardens:

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
- Terminal and transfer listener startup is disposal/race-aware so late async initialization cannot leak listeners or overwrite newer live transfer state.
- Reduced-motion support is applied to desktop UI transitions.
