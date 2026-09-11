# Ghost FTP Android

Native Android client source lives entirely under this `android/` directory.

## Current source capability

- Native Android Java UI with no AndroidX/runtime SDK dependency.
- FTP and explicit FTPS Quick Connect.
- FTPS uses the platform trust store and strict hostname verification on both control and protected passive data channels. There is no trust-all fallback.
- FTP remains available for compatibility but is explicitly unencrypted.
- Local navigation uses Android Storage Access Framework (`ACTION_OPEN_DOCUMENT_TREE`); the app does not request all-files storage access.
- Remote directory listing uses MLSD over EPSV/PASV.
- Binary upload and download are implemented.
- Uploads are staged under a random same-directory `.ghostftp-upload-<uuid>.part` name and are committed to the requested remote name only after the FTP server confirms transfer completion and accepts `RNFR`/`RNTO`.
- Downloads are written to a temporary SAF `.ghostftp-download-<uuid>.part` document and receive the requested final local name only after the FTP transfer is confirmed complete and the storage provider accepts an exact-name commit.
- Active upload/download transfers can be cancelled from the connection button. Cancellation closes the active data and control transports, invalidates the session, and requires reconnecting before further server work.
- Host, username, protocol, port and the user-granted folder URI may be remembered. Passwords are memory-only and are cleared from the UI after connection.
- Explicit saved sites store only non-secret connection identity and navigation metadata; Quick Connect never creates a hidden site.
- A saved site can own a local SAF start folder, a remote start directory, local SAF bookmarks and remote path bookmarks.
- Changing a saved site's protocol/host/port/username identity clears its remote start directory and remote bookmarks instead of silently carrying server paths to a different endpoint.
- A saved remote start directory is freshly listed before the connection becomes visible as connected; a stale/unavailable start path fails with an actionable error instead of falling back silently.
- Opening a remote bookmark performs a fresh server listing before the visible remote path is committed.
- Local starts/bookmarks are usable only while their persisted SAF read permission still exists and the provider can return a fresh directory listing.
- Site/bookmark persistence is bounded to 50 sites and 50 bookmarks of each type per site.
- No telemetry, analytics, ads, crash-reporting service or Ghost FTP backend is used.

## Upload commit safety

Android upload never writes the incoming stream directly to the requested final remote path. Ghost FTP generates a random `.ghostftp-upload-<uuid>.part` object in the same remote directory and sends `STOR` only to that staging path. The requested final path is used only after the data transfer has finished and the server has returned an accepted `226` or `250` completion reply.

After confirmed transfer completion, Ghost FTP requires `RNFR` for the staging object and `RNTO` for the requested final path. There is no silent fallback to direct, non-atomic `STOR` when a server does not support this safe commit sequence. A rejected final rename is reported as an upload failure rather than as success.

If the data stream or the completion reply fails in a way that can leave the FTP control channel state uncertain, Ghost FTP hard-closes that session instead of issuing further commands against a potentially desynchronized connection. A staging `.part` object can therefore remain on the server after a transport interruption, but the requested final remote name is not reported as successfully committed. When a rename is rejected after the transfer has been cleanly confirmed, Ghost FTP makes a best-effort attempt to delete the staging object.

The staging flow uses the same existing passive-data transport. For FTPS, the staging upload therefore retains platform-trusted certificate validation and strict hostname verification on the protected data channel; it does not introduce a trust downgrade.

## Download commit safety

Android download does not create the requested final local filename before the server has confirmed transfer completion. Before transfer starts, Ghost FTP performs a fresh SAF listing of the selected destination directory and rejects an exact-name conflict rather than asking the storage provider to overwrite or auto-rename an existing object.

The incoming data is written only to a temporary `.ghostftp-download-<uuid>.part` SAF document. `FtpSession.download()` returns only after the remote data stream has finished and the FTP server has returned an accepted `226` or `250` completion reply, so the final local-name commit is not attempted before that protocol confirmation.

Immediately before commit, Ghost FTP performs a second fresh SAF listing and again rejects an exact-name conflict. It then asks the provider to rename the staging document to the requested filename and reads back `COLUMN_DISPLAY_NAME`. A provider result such as `file (1)` is not treated as a successful download when `file` was requested. If final-name verification fails, Ghost FTP best-effort deletes the unverified result and reports failure.

Any failure before final commit best-effort deletes the staging document. The download is reported as completed only after exact-name read-back succeeds. All local download work remains inside the user-granted Storage Access Framework tree. No broad or all-files storage permission is introduced.

## Active transfer cancellation

During an active upload or download, the existing **Disconnect** control becomes **Cancel transfer**. Cancellation is deliberately fail-closed: Ghost FTP invalidates the active transfer generation, detaches the current session from visible UI state, and closes both the passive data socket and the FTP control socket. The cancelled connection is not reused; the user reconnects before performing further server operations.

The transport abort path is intentionally not synchronized on the long-running upload/download monitor. This allows the UI thread to close sockets while a worker is blocked in network I/O. The worker can unwind afterward, but generation/session guards prevent its stale completion or error callback from converting the cancelled operation into a visible success or replacing the cancellation status.

For a staged download, the worker's failure path best-effort deletes the local `.ghostftp-download-*.part` document after the socket abort. For a staged upload, cancellation can leave the remote `.ghostftp-upload-*.part` object because Ghost FTP does not issue cleanup commands over a control connection that it has intentionally invalidated. The requested final remote name is still not committed by the cancellation path.

Cancellation is not resume. Ghost FTP does not currently continue a partially transferred Android upload/download after reconnect, and it does not claim FTP `ABOR` interoperability across servers. Closing the transport and requiring a fresh session is the authoritative cancellation boundary.

## Saved-site and bookmark security boundary

Saved sites are deliberately non-secret. The persisted profile schema contains site ID/name, FTP/FTPS identity, optional SAF tree URI, optional remote start path, and bookmark lists. It contains no password, passphrase, private-key material or credential surrogate.

Quick Connect remains transient application state. A Quick Connect endpoint becomes a saved site only after the user explicitly presses **Save / update**. Bookmarks likewise require an explicitly loaded/saved site; they are never created implicitly from Quick Connect.

Remote navigation state is bound to `(protocol, host, port, username)`. If that identity changes, old remote navigation state is discarded fail-closed. Local navigation state is capability-based: the application revalidates Android's persisted SAF permission and performs a fresh directory query before committing a saved local start/bookmark.

## SFTP security boundary

SFTP is intentionally not exposed in the Android source line yet. Ghost FTP desktop requires strict host-key verification/pinning; Android will not present an SFTP option until equivalent host-key identity verification is implemented and tested. There is no silent fallback from SFTP to FTP/FTPS.

## Build

The canonical CI build uses Gradle 8.9 and Android SDK 35:

```text
gradle :app:lintDebug :app:packageGhostFtpApk --no-daemon
```

A successful build must create the installable debug APK at:

```text
android/dist/Ghost-FTP-Android.apk
```

The GitHub Actions workflow uploads that exact file as the `ghostftp-android-apk` artifact. The APK is generated by the Android toolchain; a placeholder or renamed non-APK file is not accepted.

The current Android version name is derived from the repository `VERSION` plus the development suffix because Android development is not retroactively added to an already published desktop release.

## Release signing

The CI artifact is a development/debug-signed APK suitable for installation and functional validation. A future public Android release must use a protected production signing key and must verify the resulting APK signature before publication. The repository must never contain that private key or signing password.
