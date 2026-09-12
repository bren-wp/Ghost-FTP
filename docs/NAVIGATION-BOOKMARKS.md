# Navigation bookmarks and profile start directories

Ghost FTP **0.0.5** includes navigation bookmarks and explicit local/server profile start directories as maintained Windows/Linux capabilities. The feature is intentionally narrow: it improves repeated navigation without turning path metadata into credentials, weakening connection identity boundaries or creating hidden Site Manager profiles.

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

The bookmark model does **not** contain password, passphrase, private-key data, host-key fingerprint, profile ID, access token or other authentication/trust material. Username is treated as account identity, not as a credential.

### Account identity is stricter than endpoint identity

Remote bookmark and profile-start binding use `profilebinding.AccountMatches`. The identity boundary is:

```text
protocol + canonical host + port + exact username
```

Changing only the username is enough to invalidate an inherited server path. Endpoint-only matching remains appropriate for server-scoped data such as host-key identity, but it is not sufficient authority for account-scoped navigation.

## Bookmark persistence

`internal/config/bookmarks.go` owns `bookmarks.json` under the normal local Ghost FTP state directory. The store enforces:

- maximum 256 bookmarks;
- generated fixed-length random hexadecimal IDs;
- bounded valid UTF-8 names with NUL/newline rejection;
- absolute cleaned local paths;
- shared remote-path validation;
- shared protocol/host/port/username validation;
- exact account binding for remote entries;
- duplicate-ID and malformed-state rejection;
- corrupt existing state failing closed rather than silently becoming an empty collection;
- stable sorting by name, kind and ID.

Persistence uses the existing durable state-store/recovery behavior. A missing bookmark file means an empty collection; an unreadable existing file is an error.

## Creating bookmarks

### Local bookmark

`Engine.SaveLocalBookmark(id, name, path)` persists only validated local navigation metadata. No profile or remote identity is created as a side effect.

### Remote bookmark

`Engine.SaveRemoteBookmark(id, name, path)` requires a real active remote connection. The Engine reads the active `remote.Config()` and binds the bookmark to that session's protocol, host, port and username.

The UI cannot forge this identity from stale text fields. Saving a Quick Connect bookmark has no `SaveProfile` side effect and does **not** create a hidden persistent Site Manager profile.

## Opening bookmarks

All activation goes through `Engine.NavigateBookmark(ctx, id)`. Stored paths are never treated as already-proven navigation authority.

### Local navigation

A local bookmark performs a fresh bounded local listing. Only a successful listing returns a canonical local base and authoritative item snapshot for UI commit.

### Remote navigation

A remote bookmark requires all of the following before visible state can commit:

1. an active remote connection;
2. `RemoteBookmarkMatchesAccount` / `AccountMatches` against current connection configuration;
3. capture of the current connection identity;
4. a real `RemoteList` of the stored path;
5. a second connection-identity read after the listing;
6. rejection if identity changed during the operation;
7. a fresh active-config/account match after the listing.

This closes the reconnect race: work started on one session cannot commit navigation after the application reconnects to another account/server identity.

## Windows desktop behavior

Windows exposes a native Bookmarks manager backed by `internal/desktop/bookmark_manager_windows.go` with:

- **Open**;
- **Add local**;
- **Add remote** only while connected;
- **Delete** with confirmation;
- **Close**.

Remote opening is protected both by the Engine account/session checks and by the desktop `connectionGeneration` guard so stale asynchronous completion cannot mutate visible state.

## Linux desktop behavior

Linux exposes a native X11 Bookmarks control and modal overlay through `internal/desktop/bookmark_linux.go`. It provides the same functional Open/Add local/Add remote/Delete/Close contract as Windows.

The Linux list has a bounded viewport, keyboard-visible selection, mouse-accessible scrolling and offset-aware hit testing. Child naming prompts return to the bookmark manager on cancellation instead of abandoning the whole flow. All save/open/delete operations route through the shared Engine methods rather than directly trusting persisted path state.

## Profile start directories

Site Manager profiles contain `LocalPath` and `RemotePath`, but those values are requested starts rather than unconditional UI authority.

When profile account identity changes, an inherited server start is not carried across that boundary. Tests cover FTP/FTPS reset to `/`, SFTP reset to `.`, and preservation of an explicitly supplied new remote start for the new account identity.

### Windows

Windows applies a saved remote start only while the selected profile still matches the connection fields under the strict account-identity contract. Local starts are validated through actual local listing before they become authoritative pane state.

### Linux

`internal/desktop/profile_start_linux.go` treats copied profile paths as drafts until validation succeeds:

- restores the previous verified local base before a fresh local listing;
- commits the requested local start only after successful listing;
- installs a selected profile's saved/default remote start once;
- resets inherited or previously navigated server state once when protocol/host/port/username identity changes;
- preserves a newly entered explicit Remote Path after that account-bound reset;
- blocks profile switching while connected or while another Linux UI action is busy.

## Failure behavior

The feature fails closed around navigation authority:

- corrupt bookmark state returns an error;
- invalid bookmark input is rejected before persistence;
- missing local paths fail during fresh listing;
- unavailable server paths fail during `RemoteList`;
- remote account mismatch is rejected;
- disconnect/reconnect during remote navigation invalidates the operation;
- stale Windows callbacks fail the sequence/generation gate;
- profile identity changes discard old-account inherited/navigation state before a new explicit start is accepted.

Failures must preserve the previously verified pane state wherever the surrounding navigation path supports that behavior; they must not manufacture a successful empty listing.

## Privacy

Bookmarks and start directories are local application state. The feature adds no telemetry, analytics, Ghost FTP synchronization service or credential store. Paths/usernames may be sensitive metadata, so they remain limited to explicit local navigation surfaces and privacy-safe diagnostics.

## Cross-platform parity

| Capability | Windows | Linux |
| --- | --- | --- |
| List bookmarks | Native manager | Native X11 overlay with bounded viewport |
| Add local bookmark | Yes | Yes |
| Add remote bookmark | Connected session only | Connected session only |
| Delete bookmark | Confirmed | Confirmed through destructive-action policy |
| Open via `NavigateBookmark` | Yes | Yes |
| Remote account binding | Yes | Yes |
| Stale-session protection | Engine + connection generation | Engine + serialized action/session behavior |
| Profile local start requires fresh listing | Yes | Yes |
| Profile remote start rejects account drift | Yes | Yes |

## Regression coverage

The 0.0.5 contract is protected by:

- `internal/config/bookmarks_test.go` for CRUD, validation, non-secret schema, account binding and corrupt-state fail-closed behavior;
- `internal/config/profile_start_directory_binding_test.go` for inherited start reset and explicit-new-path behavior;
- `internal/profilebinding/*_test.go` for endpoint/account identity semantics;
- `internal/desktop/profile_start_linux_test.go` and `profile_cycle_linux_test.go` for repaint/account/switching guards;
- `internal/desktop/bookmark_prompt_linux_test.go` and `bookmark_viewport_linux_test.go` for Linux modal/viewport behavior;
- desktop tests for shared bookmark wording and Site Manager navigation privacy;
- `scripts/test_navigation_bookmarks_contract.py` for cross-layer Engine/config/Windows/Linux/documentation invariants.

## 0.0.5 release boundary

Root `VERSION` is **0.0.5**. Navigation bookmarks and profile start directories are part of the 0.0.5 source/release contract, but this document never authorizes publication by itself. Publication still requires exact-head CI/native-build/authentic-runtime evidence, review/merge, exact post-merge verification, canonical `ghostftp-v0.0.5` publication/read-back and latest-only retention.

The feature does not change the public platform allow-list: Windows/Linux remain the published release surfaces and Android remains a separately validated development APK.
