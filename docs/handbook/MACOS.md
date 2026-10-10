# Ghost FTP for macOS — source-accurate application handbook and development plan

**Candidate:** 0.93.0 · **Platform:** macOS 13+ · **Implementation:** Swift 5.10, SwiftUI, AppKit file panels, Foundation Networking and macOS Security/Keychain. **Stage:** limited Preview application; not feature-identical to Windows/Linux. A compiled application, successful unit tests and a real device acceptance test are distinct achievements. Read the status column before calling a feature available.

## 1. Interface and navigation

The SwiftUI ContentView in WorkspaceShell.swift contains the visible single-app workspace navigation. The following pages are implemented:

| Workspace | Controls and behavior in source | Status / important caveats |
| --- | --- | --- |
| Files | Select a saved site, edit connection, validate host/port, test endpoint, open FTP session, navigate/refresh remote directory, upload and download, save/delete site | **Implemented FTP path**; not a functional FTPS/SFTP transfer client |
| Sites | List saved connections, select/open, possible-duplicate filter, create via sidebar, delete with Keychain removal safety | Implemented with local JSON persistence and Keychain integration |
| Transfers | FTP upload/download history, direction/status/time and clear history | Source implemented; history is not a multi-protocol persistent resumable queue |
| Cloud storage | Masked signed HTTPS PUT/GET link inputs, file open/save panels, progress status, one-object provider upload/download | **0.93.0 candidate**, 1 GiB limit; no cloud account browser |
| Sync & Backup | Export and restore **non-secret** saved-site metadata; bounded backup size and record count | Implemented; **not** general bidirectional folder sync |
| Settings | Dark appearance; remember last workspace; clear history and remove saved Keychain passwords | Implemented; verify destructive prompts and error paths |
| Help & About | Product identity, version, privacy/implementation disclosure and support links | Implemented; avoid claiming protocol parity in About |

The window uses a navigation split view and brand palette. The declared minimum usable size differs from the Windows reference; verify on actual 13-inch MacBooks, high-resolution displays, accessibility zoom and VoiceOver. The app remains a SwiftUI program rather than a port of the Tauri desktop architecture.

## 2. Saved-site editor — every relevant field

The ConnectionProfile model tracks UUID, name, protocol kind, host, port, username, keep-alive seconds and reconnect-attempt count. The editor offers password entry and optional Remember password in macOS Keychain. FTP defaults to port 21; explicit FTPS also defaults to port 21; SFTP defaults to port 22. These protocol choices represent configuration and security explanation; **the actual file-transfer engine currently supports FTP only**.

The endpoint check verifies basic TCP reachability. It **must never be labelled an authenticated session**. FTPS certificate authentication and SFTP host-key validation are not complete in the macOS engine; their file-operation controls remain disabled. An incorrect hostname, dangerous characters in FTP command arguments, bad port, duplicate profile identity or inaccessible Keychain entry should have a clear non-secret error.

Saved site IDs are security identifiers. Backup restore rejects an imported record that tries to reuse a saved UUID for a different protocol, server, port or username, preventing reuse of the wrong associated Keychain credential. Backups contain names, endpoints and options but **not passwords**. Keychain writes update existing items rather than deleting the old secret first; any failure must not falsely report saved state. Site deletion is canceled if Keychain cleanup fails instead of leaving an orphaned credential.

## 3. FTP functionality and its protocol limits

| Operation | Source status | Verification scenario |
| --- | --- | --- |
| FTP connect/disconnect | Implemented FTP control connection | Correct and incorrect password; closing window mid-session |
| Remote PWD/CWD | Implemented for FTP | Root, nested/Unicode path, traversal and permission denied |
| Directory listing | FTP MLSD/EPSV support | Malformed server entries, control-reply bound, hostile filenames |
| Upload | FTP passive data channel, file selection | Zero bytes, Unicode, partial upload and server disconnect |
| Download | FTP passive data, destination picker, temporary stage | Interrupted retrieval leaves previous destination intact |
| FTP keep-alive/reconnect fields | Model/UI options, scope depends on session engine | Verify NOOP scheduling instead of assuming it |
| FTPS | UI/configuration/security warning only | **Not implemented for file operations** until TLS engine verifies chain+hostname |
| SFTP | UI/configuration/security warning only | **Not implemented for file operations** until trusted SSH host-key engine is complete |
| Terminal/SSH shell | Not present as equivalent of desktop | Planned; no misleading shell button |
| Full local/remote dual pane | Not equivalent to Windows/Linux | Planned; current Files focuses on selected connection and remote list |
| Folder synchronization | No general remote sync engine | Backup/export page must not be described as cloud mirror/sync |

## 4. Cloud storage — signed object transfer workflow

The 0.93.0 Cloud storage page has two independent forms:

**Upload:** obtain a short-lived HTTPS object URL signed for PUT from a trusted provider console or integration; paste it into the masked input; select a local file with NSOpenPanel; transfer directly over ephemeral URLSession. The URL contains authorization in query parameters, is cleared after use and is not persisted.

**Download:** obtain the provider's signed GET URL; choose a destination with NSSavePanel; the app downloads to a temporary file, checks status and staged size, then promotes it into the selected location on success. Failed HTTP operations do not intentionally overwrite a previous valid destination.

S3-compatible services include AWS S3, Cloudflare R2, Backblaze B2 and other compatible providers **only insofar as they issue supported signed HTTP requests**. Google Cloud Storage signed URLs and Azure SAS URLs have their own semantics. A Google Drive sharing page is not a GCS signed URL and does not work as a Drive account. The macOS Preview currently has no AWS SigV4 credential-backed bucket browser, listing or pagination, Google Drive OAuth, OneDrive, Dropbox, Box sign-in, large multi-part upload, or background resumable sync.

**Cloud security:** the link is a bearer secret. A valid signature string can still be user-supplied and unsafe: check intended object and provider hostname out of band; do not accept unsolicited links for confidential files. Current source rejects plain HTTP, userinfo, fragments, literal IPs and private-looking hostnames, disallows HTTP redirects, preserves system TLS checks and caps file sizes at 1 GiB. macOS's hostname pattern check is not a full SSRF/DNS-rebinding defense. Additional resolved-address verification, provider allow-list policy and test coverage are pending.

## 5. Storage, logging and privacy

- ProfileStore persists non-secret profile metadata locally. Validate file permission restrictions, corrupt-file recovery and atomic writes. The backup exporter must not serialize credentials or raw cloud signed URLs.
- KeychainStore uses macOS Keychain restricted accessibility and non-synchronizable entries. No plaintext password fallback on failure. A deliberate "remove all passwords" action requires an explicit result and should not erase profiles unexpectedly.
- TransferHistoryStore stores bounded FTP action metadata; do not store password, private key, signed link, authorization header, complete URL query or file content. Provide clear history as a UI operation.
- URLSession for signed cloud links must remain ephemeral, without HTTP cookies/cache and without automatic redirect following. System certificate and hostname checks are required.
- Application screenshots and crash diagnostics can expose hostnames and filenames; define sensitive UI masking and a redaction policy before production telemetry (which is not required).
- If macOS Sandbox is enabled for distribution, preserve user-selected file security-scoped URL access only for the duration required; never broaden filesystem access silently.

## 6. Security, privacy and reliability work still required

| Priority | Task | Exit gate |
| --- | --- | --- |
| P0 | Complete native FTPS engine | AUTH TLS, certificate chain/hostname validation, fail-closed expiry/mismatch test and FTP operations over encrypted channel |
| P0 | Complete native SFTP engine | SSH host-key pinning/mismatch handling, key/password auth, actual file operations and interrupt tests |
| P0 | Harden signed cloud host resolution | SSRF, DNS rebinding, cloud metadata addresses, redirects, IPv6 and normalization regression suite |
| P0 | Signed upload/download failure atomicity | Preserve preexisting file, verify expected length/hash, test cancel/restart, staging cleanup |
| P0 | Keychain lifecycle | Lock/deny/rotate/delete/restore failure tests without orphan profiles or secret disclosure |
| P0 | macOS app distribution | Install/open/quit/uninstall on signed and unsigned development builds; Gatekeeper and notarization evaluation |
| P1 | OAuth Drive | System browser + PKCE, scopes, secure refresh storage, revoke/disconnect, token redaction |
| P1 | S3 account browser | Region/endpoint, SigV4, least-privilege bucket list/get/put, pagination and expiration handling |
| P1 | Cloud progress/cancel | Bounded streaming, cancel button, background/resume policy and UI state reset |
| P1 | Real backup/sync | Explicit distinction between connection backup and remote directory sync; dry-run and conflict behavior |
| P1 | Accessibility and languages | VoiceOver and keyboard-only audit; complete copy localization rather than an unverified language selector |
| P2 | Parity roadmap | Additional authenticated cloud providers, dual-pane browser, queue/scheduler, diff and SSH terminal |

## 7. Manual and CI acceptance

1. Swift Package build and XCTest must compile, execute and report tests at the exact PR commit. Tests cover saved-site identity, credentials, FTP codec and signed-link input validation.
2. FTP E2E on an isolated server must validate MLSD, upload/download, zero-byte files, Unicode filenames, interrupted transfers and recovery.
3. Keychain tests must exercise failed updates, concurrent add/update, denied deletion, recovery and no incorrect saved state.
4. Signed cloud provider tests need short-expiry PUT/GET, mismatched method/headers, 3xx rejection, 403/404/429, corrupt/incomplete payload and canceled staging.
5. AppKit open/save tests must cover scoped permissions, read-only path, disk full, existing target and app quit during operation.
6. Run on real supported macOS version(s) with VoiceOver, high contrast, narrow layouts and an actual release bundle. Source compile alone does not prove a production-safe macOS app.

## 8. Source map

- macos/Sources/GhostFTPMacApp/Models/ConnectionProfile.swift — profile fields and protocol security summaries.
- Services/ProfileStore.swift and Security/KeychainStore.swift — persistence, backup and credential lifecycle.
- Services/FTPControlSession.swift, FTPConnectionController.swift and EndpointProbe.swift — FTP operations and reachability.
- Views/WorkspaceShell.swift, ContentView.swift, CloudWorkspace.swift — all current workspaces, forms and transfer entry points.
- Services/SignedCloudTransfer.swift — signed one-object direct HTTPS transfer.
- macos/Tests/GhostFTPMacAppTests — regression coverage; macOS/Package.swift sets build targets.
