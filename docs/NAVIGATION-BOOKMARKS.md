# Navigation bookmarks and profile start directories

Ghost FTP **0.0.8** includes navigation bookmarks and explicit local/server profile start directories as maintained Windows/Linux desktop capabilities. The native macOS development frontend also wires bookmark navigation through the shared Engine. Android has its own saved-site/bookmark model and is now a production-signed public application, but it does not replace or weaken the desktop bookmark/account-binding contract documented here.

## Scope

Two related desktop navigation mechanisms are maintained:

1. **Bookmarks** are reusable local or remote navigation entries independent of Connections/saved profiles.
2. **Profile start directories** are explicit default local and remote directories saved with a saved connection profile.

Windows and Linux expose the complete desktop bookmark workflow. macOS uses the same shared bookmark Engine APIs under its native development frontend. This is **source/development parity** for macOS, not a public macOS distribution claim.

## Security model

Bookmarks are non-secret metadata. `internal/model/bookmark.go` stores stable ID, display name, kind, path and, for remote entries, protocol/host/port/username. Passwords, passphrases, private-key data and host-key trust material are never bookmark fields.

Remote bookmark and profile-start identity uses `profilebinding.AccountMatches`:

```text
protocol + canonical host + port + exact username
```

Changing account identity invalidates inherited remote navigation authority.

## Bookmark persistence

`internal/config/bookmarks.go` owns bounded local `bookmarks.json` state and validates UTF-8 names, local/remote paths, remote identity, generated IDs, duplicates and corrupt-state behavior. A missing file means no bookmarks; a malformed existing store fails closed rather than silently becoming empty state.

## Creating and opening bookmarks

`Engine.SaveLocalBookmark` stores validated local navigation metadata. `Engine.SaveRemoteBookmark` requires a real active connection and captures identity from the authoritative session; Saving a bookmark does **not** create a hidden persistent saved connection profile.

`Engine.NavigateBookmark` never treats a stored path as pre-verified authority. Local navigation performs a fresh listing. Remote navigation requires active-account match, captures connection identity, performs a real fresh remote listing, rechecks connection identity/account after the listing and rejects stale reconnect races before visible state commits.

## Windows desktop behavior

Windows exposes a native Bookmarks manager with Open, Add local, Add remote, Delete and Close actions. Remote opening is guarded by Engine account/session validation and `connectionGeneration`, so stale asynchronous completion cannot update replacement-session state.

## Linux desktop behavior

Linux exposes an X11 Bookmarks overlay with the same functional actions, bounded viewport, keyboard/mouse navigation and shared Engine ownership. Save/open/delete operations do not bypass the authoritative Engine path.

## macOS development behavior

The native AppKit development frontend exposes Bookmarks through the **same shared bookmark Engine APIs**: `Engine.Bookmarks`, `SaveLocalBookmark`, `SaveRemoteBookmark`, `RemoveBookmark` and `NavigateBookmark`. Local/remote save actions use authoritative bridge state and remote activation retains shared account/session revalidation before visible commit.

This is **source/development parity**. macOS remains a separately validated native development/source frontend; a successful development build is not Developer ID signing/notarization evidence and does not add a macOS public artifact to Ghost FTP 0.0.8.

## Profile start directories

Saved `LocalPath`/`RemotePath` values are requested starts, not unconditional UI authority. When protocol/host/port/username identity changes, inherited remote paths are reset rather than carried into a different account boundary.

Windows and Linux both require real listing/validation before requested starts become authoritative visible pane state. Linux additionally restores the previous verified local base when a requested local start cannot be validated and blocks conflicting profile switching while connected/busy.

## Failure behavior

The feature fails closed for corrupt state, invalid input, missing local paths, unavailable remote paths, account mismatch, disconnect/reconnect races and stale callbacks. Failure preserves previously verified pane state where the surrounding navigation lifecycle allows it; it never manufactures a successful empty listing.

## Privacy

Bookmarks/start directories remain local application state and add no telemetry, Ghost FTP synchronization service or credential store. Paths/usernames may be sensitive metadata and are treated accordingly in diagnostics.

## Cross-platform bookmark parity

| Capability | Windows | Linux | macOS development |
| --- | --- | --- | --- |
| List bookmarks | Native manager | Native X11 overlay | Native AppKit surface |
| Add local bookmark | Yes | Yes | Shared Engine API |
| Add remote bookmark | Connected session only | Connected session only | Connected authoritative snapshot only |
| Delete bookmark | Confirmed | Confirmed | Shared Engine removal path |
| Open via `NavigateBookmark` | Yes | Yes | Yes |
| Remote account binding | Yes | Yes | Yes |
| Stale-session protection | Engine + generation | Engine + serialized session behavior | Shared Engine + bridge generation rules |

Windows/Linux remain the maintained desktop implementation surfaces for this feature. Public release scope for Ghost FTP 0.0.8 is broader: Windows/Linux desktop, a production-signed Android APK and Chrome/Edge/Firefox/Opera helper packages. Browser helpers do not own bookmark state; on supported Windows installs their explicit sanitized `ghostftp://connect` handoff carries no credentials and never auto-connects.

## Regression coverage

The 0.0.8 contract is protected by bookmark/config/profile-binding Go tests, Linux desktop modal/viewport tests, `scripts/test_navigation_bookmarks_contract.py`, and the macOS development parity contract. Publication additionally requires exact-head CI/native-build/authentic-runtime evidence.

## 0.0.8 release boundary

Root `VERSION` is **0.0.8**. Navigation bookmarks/profile starts remain part of the maintained desktop/source contract, but this document does not authorize publication by itself. Publication requires exact-head and post-merge verification plus canonical `ghostftp-v0.0.8` publication/readback/retention.

The public release is **13 platform artifacts / 16 public files**. Android is public through the protected production-signing path, Android SFTP remains hidden until strict maintained host-key verification exists, browser packages remain local parser/copy helpers with only the sanitized Windows `ghostftp:` handoff and no secrets or automatic connection, and macOS remains a separately validated native development/source frontend.
