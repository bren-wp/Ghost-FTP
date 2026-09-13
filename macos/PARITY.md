# macOS ↔ Windows parity contract

The Windows desktop is the canonical visual and behavior reference for the macOS client. macOS may use native AppKit windowing, accessibility, file panels and menu conventions, but capability/state/security parity is mandatory. The Mac frontend must use the same typed `internal/api.Engine`; no decorative or dead controls are allowed.

Saved profiles are protected by a native `Security.framework` Keychain-held AES-256 wrapping key with `WhenUnlockedThisDeviceOnly`. The profile envelope and credential payloads are authenticated and type-marked before use. Saved credential values never cross into Swift: Site Manager receives only public profile metadata and `HasPassword` / `HasPassphrase` state. After exact shared profile binding succeeds, macOS converts a durable credential into the existing same-user ephemeral runtime broker; ownership follows the pending SFTP trust flow or the live SFTP/Curl session and is released when that owner closes.

## Appearance and product identity

- Classic Light uses workspace `#EEF1F5`, panel `#F6F8FB`, list `#FAFBFD`.
- Dark uses workspace `#0B0F17`, panel `#121824`, list `#161D2A`.
- English remains the default/fallback and the product exposes the same 24 languages.
- FTP, explicit FTPS and SFTP remain the desktop protocol set.
- Product telemetry remains disabled: no telemetry, analytics, advertising, tracking or hidden Ghost FTP backend.
- Root `VERSION` is the only product version source of truth.

## Windows action inventory

A checkbox moves to `[x]` only when the visible Mac action is wired to real shared-engine/platform behavior and covered by tests. Merely drawing the matching control does not satisfy parity.

### Connection, profiles and application

- [x] Connect
- [x] Disconnect
- [x] Site Manager
- [x] Bookmarks
- [x] Private Key
- [x] Save Profile
- [x] Remove Profile
- [x] Settings
- [x] About
- [x] Diagnostics

### Local file pane

- [x] Local Refresh
- [x] Local Choose Folder
- [x] Local Up
- [x] Local New Folder
- [x] Local Rename
- [x] Local Delete
- [x] Local Filter
- [x] Local Recursive Search

### Remote file pane

- [x] Remote Refresh
- [x] Remote Up
- [x] Remote New Folder
- [x] Remote Rename
- [x] Remote Delete
- [x] Remote Permissions
- [x] Remote Edit
- [x] Remote Filter
- [x] Remote Recursive Search
- [x] Directory Compare

### Transfer actions and queue

- [x] Upload
- [x] Download
- [x] Pause Queue
- [x] Resume Queue
- [x] Cancel Transfer
- [x] Retry Transfer
- [x] Clear Finished
- [x] Move Top
- [x] Move Up
- [x] Move Down
- [x] Move Bottom

## Implemented application boundary

Bookmarks use the shared `Engine.Bookmarks`, `SaveLocalBookmark`, `SaveRemoteBookmark`, `RemoveBookmark` and `NavigateBookmark` APIs. Local and remote save actions take their path only from the authoritative bridge snapshot; Swift cannot submit an arbitrary bookmark path. Remote bookmarks retain the shared account binding and connection-identity checks, and a successful navigation publishes the already-verified shared-engine listing into the visible Mac snapshot before AppKit applies it.

Settings use the shared `model.Settings` and `Engine.Settings` / `Engine.SetSettings` contract. The native window exposes the supported language, Classic Light/Dark appearance, parallelism, independent upload/download limits, connection timeout, automatic retry count and delay, conflict policy, and delete confirmation. Saving starts from the current authoritative settings value so fields unknown to a native frontend are preserved. The language selector is populated from the shared 24-language `i18n.Languages()` registry rather than a Mac-only list. Appearance changes use the documented Classic Light and Dark palette values and are applied through native AppKit appearance without creating a second settings store.

About is the only Mac application surface that exposes the publisher/author identity (`BRENDIGO LTD`, `brendigo.com`). Product version remains bound to root `VERSION` by the native build, while product/support identity remains explicit. Diagnostics is intentionally compact and privacy-safe: it exposes only connected state, protocol and the visible remote path plus the no-telemetry/local-profile statement. It does not export raw logs, protected secrets or generic diagnostic payloads.

## Implemented file-workspace boundary

The native AppKit workspace exposes real Local and Remote file tables backed by `internal/api.Engine.LocalList` and `internal/api.Engine.RemoteList`. Local folder selection uses the native macOS folder panel. Local/remote Up and Refresh actions operate on the active engine paths, and directory double-click navigation follows the same intent as Windows. Upload and Download queue real shared-engine transfers; regular files use `AddTransfer`, directories use bounded `AddTreeTransfer`, and symbolic links are rejected rather than silently followed.

Local and Remote New Folder, Rename and Delete are wired to the same `Engine.LocalMkdir` / `LocalRename` / `LocalDelete` and `Engine.RemoteMkdir` / `RemoteRename` / `RemoteDelete` APIs used by the desktop model. Mutation bridge calls require the directory shown in AppKit to match the engine snapshot, and rename/delete require the selected name to still exist in that snapshot. A stale folder or stale selection therefore fails closed rather than mutating an unseen target. Destructive delete always requires an explicit native confirmation in the current Mac development surface.

Remote Permissions is wired to `Engine.RemoteChmod` for both FTP/FTPS and SFTP. The Mac action mirrors the Windows safety boundary: at most 1000 selected items, symbolic links are skipped, the prompt defaults to `644`, and only 3- or 4-digit octal modes are accepted by the UI before the shared transport layer validates the mode again. Each target must still exist in the active remote snapshot, and mutation completion remains navigation-generation-bound before the pane is refreshed.

Remote Edit is wired directly to the shared `Engine.RemoteEditOpen` / `Engine.RemoteEditSave` contract and uses a native built-in AppKit text editor rather than an external process or a second transfer path. Only one visible regular remote file can be opened; the bridge revalidates the current remote directory snapshot before open and save. The shared engine enforces the UTF-8/size boundary, revision conflict detection, private staging, permission preservation and post-upload read-back verification. A revision conflict is fail-closed: the editor keeps the user's text selectable but requires an explicit reload before another save, so a stale revision cannot silently overwrite newer remote content. The remote listing is refreshed only after a verified save and only while the originating navigation generation and directory are still current.

Directory Compare is wired to the shared `Engine.CompareDirectoryItems` contract using fresh Local/Remote listings from the existing engine. The native comparison window shows conservative status for both sides and enables Open Both only when `Engine.SynchronizedDirectoryName` proves an ordinary directory exists on both sides. Compare cancellation is latched before work enters the serialized engine queue, so Cancel, Close or Disconnect cannot be lost while earlier work is still running. Open Both stages both shared `Engine.LocalList` and `Engine.RemoteList` reads under one cancellable operation and atomically publishes both visible snapshots only after both reads and the final stale-path check succeed; one pane can never navigate while the other fails. Directory Compare performs no independent filesystem or protocol traversal.

Local and Remote Recursive Search are wired to the shared `Engine.SearchLocalRecursive` / `Engine.SearchRemoteRecursive` APIs. The AppKit search window exposes Search, Cancel, Close and Navigate; the bridge requires the visible Local/Remote snapshot to still match the requested root, and the shared engine retains bounded depth, visited-item, result, batch and timeout limits. Remote search holds one generation-bound remote operation for the scan, Disconnect cancels search before entering the serialized engine queue, and stale navigation generations discard results rather than applying them to another folder. Navigate opens the containing Local/Remote folder through the normal listing path; recursive search does not introduce a second filesystem or FTP/SFTP traversal implementation.

Local and Remote Filter now use the same shared `internal/itemlist.Filter` implementation as Windows/Linux. The bridge keeps the authoritative Local/Remote directory snapshots separate from the filtered visible slices, so filtering cannot change mutation or transfer authority. Matching remains Unicode `SimpleFold` aware, whitespace-delimited terms use AND semantics, clearing the query restores an independent full-snapshot view, and applying a filter performs no filesystem or network I/O. Explicit Refresh/List performs one real listing and then reapplies the current filter to that new snapshot. Selection is restored by item name where the filtered result still contains it, and row actions are disabled while a filtered view is being replaced.

Mutation completion is generation-bound: if the user navigates away while an operation is queued or running, the old operation cannot refresh the newly selected directory. Remote mutations also require the active connection and use bounded engine contexts.

The macOS bridge keeps each transfer action bound to the currently visible engine snapshot so stale UI names cannot be used after navigation. Download targets are derived with the shared safe-local-child validation and remote names are validated before transfer. The bridge remains typed C ABI only: no JSON dispatcher, localhost server, browser IPC or credential-bearing generic payload was added.

The native AppKit Transfer Queue now renders the authoritative shared transfer-manager snapshot and paused state. Pause/Resume, multi-select Cancel/Retry, Clear Finished and Top/Up/Down/Bottom priority controls call the existing typed `internal/api.Engine` queue APIs; macOS does not run a second scheduler or protocol stack. The queue refreshes at a bounded one-second cadence, preserves selection by stable transfer ID, allows reordering only for one queued job, and uses the shared `TransferJob` byte/progress/speed/ETA fields without fabricating unsupported metrics. Retry remains connection-bound, and terminal completion refreshes the file panes without changing queue authority.

SFTP host-key confirmation retains the transient connection secret only for the pending trust retry while password/passphrase fields are cleared from the visible UI immediately. The transient value is discarded after the connect/trust decision and is never stored as app state.

## Behavior that must remain identical

The completed Mac surface must preserve the Windows/shared contracts for protocol defaults and explicit secure-protocol selection, FTPS certificate/hostname validation, strict SFTP host-key trust, saved-profile credential consent, account-identity rebinding, connection cancellation/generation ownership, sorting, filtering, bounded recursive search, conservative directory comparison, Remote Edit conflict/read-back semantics, transfer pause/resume/cancel/retry, four-way queued reordering, progress/speed/ETA truthfulness, conflict policy, parallelism, retry behavior, timeout, delete confirmation and independent aggregate upload/download bandwidth ceilings.

## macOS-native boundaries

The following are allowed to be macOS-native without weakening parity:

- application/window lifecycle and menu integration;
- `NSOpenPanel` / `NSSavePanel` style file and folder selection;
- Keychain-backed saved-secret protection after explicit user consent;
- native keyboard focus, VoiceOver/accessibility and Retina scaling;
- Apple signing, hardened runtime and notarization for a future public release.

Native behavior does not permit removing Windows functionality, silently changing protocol/security behavior or exposing controls that are not connected to real state.

## Promotion gate

The native action inventory is complete only when every box above is `[x]` and the exact source passes universal Intel/Apple-Silicon build plus privacy/security CI. Public distribution is a separate release gate: native runtime evidence, Apple Developer signing/notarization and explicit public release-workflow expansion must remain truthful and reviewed rather than being inferred from source parity.
