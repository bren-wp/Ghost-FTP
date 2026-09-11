# Navigation bookmarks and profile start directories

Ghost FTP implements navigation bookmarks and explicit local/server start directories in the **post-0.0.3 source line**, targeted for the next public release. This source capability is **not retroactively part of the already published Ghost FTP 0.0.3 release**. Root `VERSION` remains **0.0.3** until the normal versioned release lifecycle explicitly advances it.

The feature is intentionally narrow: it improves repeated navigation without turning path metadata into credentials, weakening connection identity boundaries, or creating hidden Site Manager profiles.

## Scope

Two related navigation mechanisms are maintained:

1. **Bookmarks** are reusable navigation entries independent of Site Manager profiles. A bookmark is either local or remote.
2. **Profile start directories** are the explicit default local and remote directories saved with one Site Manager profile.

Both Windows and Linux expose bookmark navigation. Profile start directories reuse the existing profile fields and are guarded so a saved server path cannot silently cross to another account identity.

## Security model

### Bookmarks are non-secret metadata

`internal/model/bookmark.go` defines the persisted bookmark model. It contains only:

- stable bookmark ID;
- display name;
- kind (`local` or `remote`);
- path;
- for remote bookmarks only: protocol, host, port and username.

The bookmark model does **not** contain password, passphrase, private-key contents/path, host-key fingerprint, profile ID, access token or other authentication/trust material. Username is treated as account identity, not as a credential.

This distinction is deliberate. A remote server path such as `/home/alice/site` can be safe only when associated with the same login identity that created it. The account metadata exists to reject cross-account reuse, not to reconnect automatically.

### Account identity is stricter than endpoint identity

Remote bookmark and profile-start binding use `profilebinding.AccountMatches` rather than endpoint-only comparison. The identity boundary is:

```text
protocol + canonical host + port + exact username
```

Host/protocol canonicalization follows the shared profile-binding rules, while username remains exact. Changing only the username is therefore enough to invalidate an inherited server path.

Endpoint-only matching remains appropriate for server-scoped data such as host-key identity, but it is not sufficient authority for account-scoped navigation.

## Bookmark persistence

`internal/config/bookmarks.go` owns bookmark persistence in `bookmarks.json` under the normal local Ghost FTP state directory.

The store enforces the following contract:

- maximum 256 bookmarks;
- generated IDs are fixed-length random hexadecimal identities;
- names are trimmed, valid UTF-8, bounded, and reject NUL/newline characters;
- local paths must be absolute and are cleaned with platform-local path semantics;
- a local bookmark is rejected if network/account fields are supplied;
- remote paths pass the shared remote-path validator;
- remote protocol/host/port/username pass the shared connection validator;
- duplicate IDs and malformed persisted entries fail closed;
- an existing corrupt bookmark state must not silently become an empty collection during a later save;
- list results are stable-sorted by name, kind and ID.

Persistence uses the existing durable state-store behavior, including the previous-generation file used by that store. A missing bookmark file means an empty collection; an unreadable/corrupt existing state is treated as an error instead of authorizing destructive replacement.

## Creating bookmarks

### Local bookmark

`Engine.SaveLocalBookmark(id, name, path)` persists only local navigation metadata. The local path is validated by the bookmark store. No profile or connection identity is involved.

### Remote bookmark

`Engine.SaveRemoteBookmark(id, name, path)` requires a real active remote connection. The Engine reads the active `remote.Config()` and binds the bookmark to that actual connection's protocol, host, port and username.

The desktop UI does not get to provide or forge this identity. This prevents stale text fields from creating a bookmark that claims to belong to a session that is not actually active.

Saving a bookmark has no `SaveProfile` side effect. A bookmark created while using Quick Connect therefore remains a bookmark and **does not create a hidden persistent Site Manager profile**.

## Opening bookmarks

All bookmark activation goes through `Engine.NavigateBookmark(ctx, id)`. The desktop panes must not treat stored paths as already-proven navigation authority.

### Local navigation

For a local bookmark, `NavigateBookmark` performs a real bounded local directory list through the local manager. Only a successful list returns a canonical local base and item snapshot for UI commit. A stale, missing or inaccessible local directory therefore returns an actionable error instead of changing the pane to an unverified path.

### Remote navigation

For a remote bookmark, the Engine performs all of the following before returning success:

1. require an active remote connection;
2. compare bookmark identity with the current connection using `RemoteBookmarkMatchesAccount` / `AccountMatches`;
3. capture the remote connection identity;
4. perform a real `RemoteList` of the bookmark path through the active session;
5. capture connection identity again after the list;
6. reject the result if the connection identity changed;
7. re-read active connection configuration and re-check the bookmark/account match.

This double revalidation closes the reconnect race: a request started on one session cannot commit navigation state after the application has reconnected to another server/account.

## Windows desktop behavior

Windows exposes **Bookmarks** as a real desktop navigation entry backed by `internal/desktop/bookmark_manager_windows.go`.

The native manager supports:

- **Open** selected bookmark;
- **Add local** from the currently verified local path;
- **Add remote** from the current remote path only while connected;
- **Delete** with confirmation;
- **Close**.

Opening a local bookmark uses the normal local navigation sequence/cancellation guard. Opening a remote bookmark additionally captures the desktop `connectionGeneration`; the callback is discarded when its sequence is stale, the connection generation changed, or the client disconnected before UI commit.

Remote identity is therefore defended twice: once by the Engine's account/session verification and again by the desktop generation guard that protects visible state from stale asynchronous completion.

## Linux desktop behavior

Linux exposes **Bookmarks** as a native X11 header control and modal overlay through `internal/desktop/bookmark_linux.go`.

The overlay supports the same functional actions as Windows:

- open selected bookmark;
- add current local directory;
- add current remote directory when connected;
- delete selected bookmark;
- close.

The bookmark list uses a bounded visible viewport. Keyboard selection automatically keeps the selected item in view, while mouse-accessible up/down controls move through collections larger than the overlay. Hit testing includes the viewport offset and rejects the list padding, so Open/Delete always refer to a visibly selectable bookmark even when the persisted collection is larger than one page.

Keyboard and mouse handling are routed through the established Linux prompt/modal dispatcher. Bookmark naming prompts feed the same Engine save methods used by Windows, and bookmark activation goes through `Engine.NavigateBookmark` rather than directly trusting the stored path. Cancelling a bookmark-name child prompt returns to the manager instead of abandoning the entire bookmark flow.

The Bookmarks control is rendered through the existing workspace/file-filter extension path and has a matching click handler; it is not a decorative or dead control.

## Profile start directories

Site Manager profiles already contain `LocalPath` and `RemotePath`. The navigation feature treats these values as explicit requested start locations, not as unconditional UI state.

### Identity-changing profile edits

When an existing profile's account identity changes, an inherited remote path is not automatically carried across that boundary. Configuration tests cover:

- host/account identity change resetting an inherited FTPS/FTP remote start to `/`;
- SFTP reset using protocol default `.`;
- an explicitly supplied new remote start for the new identity remaining valid.

This preserves user intent while preventing an old account's directory from silently becoming the start directory of a different login.

### Windows profile start behavior

Windows applies a saved remote start only when the selected profile still matches the connection fields through the same strict account-identity contract, including username. A user who selects a profile and then edits the login identity cannot silently inherit that profile's unrelated remote path.

The local start path is validated through actual local listing behavior before it becomes authoritative pane state.

### Linux profile start behavior

`internal/desktop/profile_start_linux.go` compensates for legacy `cycleProfile()` behavior that copies profile paths into editable fields immediately.

The guard runs on the UI goroutine before the Bookmarks header is painted and treats those copied values as drafts:

- the previous verified local base is restored immediately;
- the requested profile local start is passed to `refreshLocal`;
- only successful local listing may commit the resulting canonical base;
- selecting a profile installs its saved/default remote start once;
- any subsequent change to protocol, canonical host, port or exact username resets the current server start once to the new protocol default (`/` for FTP/FTPS, `.` for SFTP), even if the current value came from navigation on the old account;
- after that account-bound reset has occurred, a newly entered explicit Remote Path survives ordinary repaints;
- profile switching itself is disabled while a Linux connection is active or another Linux UI action is busy, preventing profile selection from mutating path/account drafts in the middle of an active session.

This keeps stale local, inherited, or previously navigated server paths from becoming visible authority merely because a profile row was selected or its account identity was edited.

## Error and stale-state behavior

The feature is fail-closed around navigation authority:

- missing/corrupt bookmark state returns an error;
- invalid bookmark input is rejected before persistence;
- unavailable local paths fail during fresh listing;
- unavailable server paths fail during `RemoteList`;
- remote bookmark account mismatch is rejected;
- disconnect/reconnect during remote navigation invalidates the operation;
- Windows stale async callbacks are rejected by navigation sequence / connection generation;
- profile identity changes discard the previous account's inherited or navigated server start before a new explicit start can be entered.

A failure must leave the previously verified pane state intact whenever the surrounding navigation path supports that behavior; it must not manufacture a successful empty listing.

## Privacy

Bookmarks and start directories are local application state. The feature adds no telemetry, analytics, remote Ghost FTP service, synchronization backend or mandatory account.

Paths and usernames may themselves be sensitive metadata, so UI and diagnostics should avoid exposing them outside the explicit navigation surfaces. Authentication secrets remain governed by the existing protected-profile secret handling and are never copied into `bookmarks.json`.

## Cross-platform parity

The maintained parity target is behavioral rather than pixel-identical:

| Capability | Windows | Linux |
| --- | --- | --- |
| List bookmarks | Native manager | Native X11 overlay with bounded viewport |
| Add local bookmark | Yes | Yes |
| Add remote bookmark | Connected session only | Connected session only |
| Delete bookmark | Confirmed | Confirmed through existing destructive-action policy |
| Open local bookmark via `NavigateBookmark` | Yes | Yes |
| Open remote bookmark via `NavigateBookmark` | Yes | Yes |
| Remote account binding | Yes | Yes |
| Stale-session protection | Engine + connection generation | Engine + serialized Linux action/session behavior |
| Profile local start requires real listing | Yes | Yes |
| Profile remote start rejects account drift | Yes | Yes, one-time reset per identity change |

## Regression coverage

The implementation is protected by Go unit tests and source-level regression contracts, including:

- `internal/config/bookmarks_test.go` — CRUD, validation, non-secret persisted schema, account binding and corrupt-state fail-closed behavior;
- `internal/config/profile_start_directory_binding_test.go` — inherited remote-start reset and explicit-new-path behavior across identity changes;
- `internal/profilebinding/*_test.go` — canonical endpoint/account identity semantics;
- `internal/desktop/profile_start_linux_test.go` — repaint safety, inherited/navigated old-account reset and explicit-new-path behavior after the identity boundary;
- `internal/desktop/profile_cycle_linux_test.go` — connected/busy profile-switch guards and normal idle profile loading;
- `internal/desktop/bookmark_prompt_linux_test.go` — bookmark-name child-prompt classification;
- `internal/desktop/bookmark_viewport_linux_test.go` — viewport clamping, later-row reachability and hit-test padding/offset behavior;
- desktop tests for shared bookmark wording and Site Manager navigation privacy;
- `scripts/test_navigation_bookmarks_contract.py` — cross-layer Engine/config/Windows/Linux/documentation wiring and security invariants.

The regression contract intentionally checks that visible desktop controls have real handlers and that source documentation continues to distinguish this post-0.0.3 work from the already published 0.0.3 binaries.

## Release boundary

This document describes implemented source behavior on the feature-development line. It does not authorize publication by itself.

Before the capability is called part of a public release, the normal Ghost FTP lifecycle still requires exact-head tests/builds, platform packaging/install gates, review/merge, version advancement, release publication, remote read-back and canonical retention verification.

Root `VERSION` remains **0.0.3** during this work. The eventual successor version must be created as a new immutable public release identity rather than rewriting the existing 0.0.3 release.
