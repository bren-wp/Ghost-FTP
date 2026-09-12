# macOS ↔ Windows parity contract

The Windows desktop is the canonical visual and behavior reference for the macOS client. macOS may use native AppKit windowing, accessibility, file panels and menu conventions, but capability/state/security parity is mandatory. The Mac frontend must use the same typed `internal/api.Engine`; no decorative or dead controls are allowed.

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
- [ ] Site Manager
- [ ] Bookmarks
- [x] Private Key
- [ ] Save Profile
- [ ] Remove Profile
- [ ] Settings
- [ ] About
- [ ] Diagnostics

### Local file pane

- [x] Local Refresh
- [x] Local Choose Folder
- [x] Local Up
- [x] Local New Folder
- [x] Local Rename
- [x] Local Delete
- [x] Local Filter
- [ ] Local Recursive Search

### Remote file pane

- [x] Remote Refresh
- [x] Remote Up
- [x] Remote New Folder
- [x] Remote Rename
- [x] Remote Delete
- [x] Remote Permissions
- [ ] Remote Edit
- [x] Remote Filter
- [ ] Remote Recursive Search
- [ ] Directory Compare

### Transfer actions and queue

- [x] Upload
- [x] Download
- [ ] Pause Queue
- [ ] Resume Queue
- [ ] Cancel Transfer
- [ ] Retry Transfer
- [ ] Clear Finished
- [ ] Move Top
- [ ] Move Up
- [ ] Move Down
- [ ] Move Bottom

## Implemented file-workspace boundary

The native AppKit workspace exposes real Local and Remote file tables backed by `internal/api.Engine.LocalList` and `internal/api.Engine.RemoteList`. Local folder selection uses the native macOS folder panel. Local/remote Up and Refresh actions operate on the active engine paths, and directory double-click navigation follows the same intent as Windows. Upload and Download queue real shared-engine transfers; regular files use `AddTransfer`, directories use bounded `AddTreeTransfer`, and symbolic links are rejected rather than silently followed.

Local and Remote New Folder, Rename and Delete are wired to the same `Engine.LocalMkdir` / `LocalRename` / `LocalDelete` and `Engine.RemoteMkdir` / `RemoteRename` / `RemoteDelete` APIs used by the desktop model. Mutation bridge calls require the directory shown in AppKit to match the engine snapshot, and rename/delete require the selected name to still exist in that snapshot. A stale folder or stale selection therefore fails closed rather than mutating an unseen target. Destructive delete always requires an explicit native confirmation in the current Mac development surface.

Remote Permissions is wired to `Engine.RemoteChmod` for both FTP/FTPS and SFTP. The Mac action mirrors the Windows safety boundary: at most 1000 selected items, symbolic links are skipped, the prompt defaults to `644`, and only 3- or 4-digit octal modes are accepted by the UI before the shared transport layer validates the mode again. Each target must still exist in the active remote snapshot, and mutation completion remains navigation-generation-bound before the pane is refreshed.

Local and Remote Filter now use the same shared `internal/itemlist.Filter` implementation as Windows/Linux. The bridge keeps the authoritative Local/Remote directory snapshots separate from the filtered visible slices, so filtering cannot change mutation or transfer authority. Matching remains Unicode `SimpleFold` aware, whitespace-delimited terms use AND semantics, clearing the query restores an independent full-snapshot view, and applying a filter performs no filesystem or network I/O. Explicit Refresh/List performs one real listing and then reapplies the current filter to that new snapshot. Selection is restored by item name where the filtered result still contains it, and row actions are disabled while a filtered view is being replaced.

Mutation completion is generation-bound: if the user navigates away while an operation is queued or running, the old operation cannot refresh the newly selected directory. Remote mutations also require the active connection and use bounded engine contexts.

The macOS bridge keeps each transfer action bound to the currently visible engine snapshot so stale UI names cannot be used after navigation. Download targets are derived with the shared safe-local-child validation and remote names are validated before transfer. The bridge remains typed C ABI only: no JSON dispatcher, localhost server, browser IPC or credential-bearing generic payload was added.

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

macOS remains a development source platform until all action boxes above are complete, universal Intel/Apple-Silicon builds pass from exact source, native runtime screenshots are captured from the tested SHA, privacy/security audits pass, signing/notarization policy is truthful and the public release workflow is explicitly expanded in a separate reviewed change.
