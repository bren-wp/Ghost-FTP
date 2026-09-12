# Ghost FTP macOS development app

The `macos/` tree is the dedicated native macOS development surface for Ghost FTP. It is deliberately separate from the Windows and Linux packaging trees, but it is not a separate product or a reduced feature edition.

**Windows desktop is the canonical visual and behavior reference.** The target is 1:1 feature parity in the sense that every supported Windows desktop action has an equivalent, real macOS action backed by the same product state and security rules. Native platform primitives may differ where macOS requires them, but capability, labels, ordering, validation, enabled/disabled state and workflow semantics must remain aligned.

The macOS client uses the same typed `internal/api.Engine` as the maintained Windows/Linux desktop clients through an in-process C ABI bridge. UI work follows one non-negotiable rule: **no decorative or dead controls**. A control is added to the visible Mac UI only when the corresponding engine-backed operation is functional and testable.

## Implemented development surface

The native AppKit application currently includes:

- FTP, explicit FTPS and SFTP Quick Connect and Disconnect through the shared engine;
- native private-key file selection for SFTP;
- strict SFTP pending host-key trust with the bundled AskPass helper and Darwin runtime secret broker;
- Local and Remote file panes backed by `Engine.LocalList` and `Engine.RemoteList`;
- local native folder selection, Local/Remote Up and Refresh, and directory double-click navigation;
- Local and Remote New Folder, Rename and Delete through the shared engine;
- explicit native confirmation before destructive delete operations;
- stale-folder and stale-selection rejection before mutations can reach the engine;
- navigation-generation ownership so an old mutation cannot refresh a newly selected folder;
- remote file metadata including size, modification time and permissions when provided by the protocol;
- real Upload and Download queue entry points; regular files use `Engine.AddTransfer` and directories use the bounded `Engine.AddTreeTransfer` path;
- symbolic-link rejection for transfer actions rather than silent traversal;
- visible-snapshot binding for transfer actions so stale names cannot be submitted after navigation.

The password and private-key passphrase fields are cleared from the visible UI immediately when connecting. For SFTP first-contact trust, the transient in-memory connection input is retained only long enough to perform the explicitly approved host-key retry, then discarded; it is not stored as persistent app state.

## Visual contract

The macOS frontend follows the maintained Windows layout hierarchy: branded header, profile/application actions, Quick Connect, Local and Remote panes, transfer actions, transfer queue, status and version surfaces. The native AppKit window may use macOS window chrome, focus behavior and accessibility APIs, while the Ghost FTP workspace remains visually aligned with Windows.

The local source palettes are identical to the desktop reference:

- Classic Light: workspace `#EEF1F5`, panel `#F6F8FB`, list `#FAFBFD`.
- Dark: workspace `#0B0F17`, panel `#121824`, list `#161D2A`.

The finished Mac client must expose the same **24 languages**, with English as canonical default/fallback, and the same FTP, explicit FTPS and SFTP protocol semantics as the maintained desktop product.

## Privacy and dependency boundary

The Mac app has no telemetry, analytics, advertising, tracking pixels, remote fonts, remote styles or mandatory Ghost FTP account. The build is source-local and must not download runtime UI or application code. The bridge is typed and in-process: no JSON dispatcher, localhost application server or browser IPC is used.

Persistent saved-profile secrets remain fail-closed on macOS until the dedicated Keychain-backed implementation is added behind the same explicit credential-persistence consent and account-identity binding used by the shared desktop model.

## Remaining parity work

The development surface is not yet a complete Mac release. Site Manager, saved-profile Keychain support, Bookmarks, Settings/About/Diagnostics, permissions, filtering/search, Directory Compare, Remote Edit, full transfer queue controls, localization and the remaining Windows behavior inventory are still tracked in `PARITY.md` and remain intentionally absent until their real engine-backed implementations are ready.

The Mac development artifact is **not part of the current 0.0.5 public release allow-list**. The existing verified Windows/Linux 14-platform-artifact / 17-public-file release contract remains unchanged until macOS reaches full functionality, runtime evidence and distribution/signing/notarization gates.

Build on macOS with:

```bash
bash macos/BUILD.sh
```

The development output is `macos/dist/Ghost-FTP-<VERSION>-macOS.app.zip` and contains `Ghost FTP.app`.

See `PARITY.md` for the exact Windows action inventory that must be completed before macOS can be promoted from development source to a public desktop release platform.
