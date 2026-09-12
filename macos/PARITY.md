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

- [ ] Connect
- [ ] Disconnect
- [ ] Site Manager
- [ ] Bookmarks
- [ ] Private Key
- [ ] Save Profile
- [ ] Remove Profile
- [ ] Settings
- [ ] About
- [ ] Diagnostics

### Local file pane

- [ ] Local Refresh
- [ ] Local Choose Folder
- [ ] Local Up
- [ ] Local New Folder
- [ ] Local Rename
- [ ] Local Delete
- [ ] Local Filter
- [ ] Local Recursive Search

### Remote file pane

- [ ] Remote Refresh
- [ ] Remote Up
- [ ] Remote New Folder
- [ ] Remote Rename
- [ ] Remote Delete
- [ ] Remote Permissions
- [ ] Remote Edit
- [ ] Remote Filter
- [ ] Remote Recursive Search
- [ ] Directory Compare

### Transfer actions and queue

- [ ] Upload
- [ ] Download
- [ ] Pause Queue
- [ ] Resume Queue
- [ ] Cancel Transfer
- [ ] Retry Transfer
- [ ] Clear Finished
- [ ] Move Top
- [ ] Move Up
- [ ] Move Down
- [ ] Move Bottom

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
