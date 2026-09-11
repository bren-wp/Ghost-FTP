# Ghost FTP Android

Native Android client source lives entirely under this `android/` directory.

## Current source capability

- Native Android Java UI with no AndroidX/runtime SDK dependency.
- FTP and explicit FTPS Quick Connect.
- FTPS uses the platform trust store and strict hostname verification on both control and protected passive data channels. There is no trust-all fallback.
- FTP remains available for compatibility but is explicitly unencrypted.
- Local navigation uses Android Storage Access Framework (`ACTION_OPEN_DOCUMENT_TREE`); the app does not request all-files storage access.
- Remote directory listing uses MLSD over EPSV/PASV.
- Passive data-channel setup is fail-closed: malformed EPSV/PASV replies, invalid passive ports, TCP data-connect failures and FTPS data-channel TLS failures close the FTP session and require reconnect.
- Binary upload and download are implemented.
- Uploads are staged under a random same-directory `.ghostftp-upload-<uuid>.part` name and are committed to the requested remote name only after the FTP server confirms transfer completion and accepts `RNFR`/`RNTO`.
- Downloads are written to a temporary SAF `.ghostftp-download-<uuid>.part` document and receive the requested final local name only after the FTP transfer is confirmed complete and the storage provider accepts an exact-name commit.
- An active upload/download can be cancelled from the existing connection action. While a transfer is cancellable, **Disconnect** becomes **Cancel transfer**.
- Cancellation and final-name commit are serialized by an explicit transfer commit gate. A cancel that wins before finalization prevents the final remote/local name from being committed; once irreversible finalization has atomically started, the UI changes to **Finalizing…** and no longer claims that cancellation is possible.
- Cancelling during active data I/O hard-closes both the active data socket and FTP control socket and requires a fresh reconnect before any further server operation.
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

After confirmed transfer completion, Ghost FTP moves the transfer lifecycle into a staged-ready phase. Cancellation and final-name commit then race through one synchronized `TransferCommitGate`: if cancellation wins, `RNFR`/`RNTO` are never started; if finalization wins, later cancel requests are rejected as too late rather than closing the control channel mid-rename and pretending an irreversible operation was rolled back.

Only after the commit gate grants finalization does Ghost FTP require `RNFR` for the staging object and `RNTO` for the requested final path. There is no silent fallback to direct, non-atomic `STOR` when a server does not support this safe commit sequence. A rejected final rename is reported as an upload failure rather than as success.

If cancellation arrives after the server has cleanly confirmed the staged upload but before finalization starts, the worker keeps the synchronized control channel only long enough to make a best-effort `DELE` of the staging object and then hard-closes the session. If data I/O or a completion reply was interrupted, the control channel is treated as unknown and is hard-closed immediately; in that case a `.part` object can remain on the server and must be treated as an orphan staging file rather than a completed upload.

The staging flow uses the same existing passive-data transport. For FTPS, the staging upload therefore retains platform-trusted certificate validation and strict hostname verification on the protected data channel; it does not introduce a trust downgrade.

## Download commit safety

Android download does not create the requested final local filename before the server has confirmed transfer completion. Before transfer starts, Ghost FTP performs a fresh SAF listing of the selected destination directory and rejects an exact-name conflict rather than asking the storage provider to overwrite or auto-rename an existing object.

The incoming data is written only to a temporary `.ghostftp-download-<uuid>.part` SAF document. `FtpSession.download()` returns only after the remote data stream has finished and the FTP server has returned an accepted `226` or `250` completion reply, so the final local-name commit is not attempted before that protocol confirmation.

Immediately before commit, Ghost FTP performs a second fresh SAF listing and again rejects an exact-name conflict. It then atomically claims the transfer commit gate before calling `DocumentsContract.renameDocument()`. A cancellation that already won causes the staging document to be best-effort deleted and prevents the rename call. Once the commit gate is owned by finalization, the UI disables cancellation and treats the rename/read-back sequence as an irreversible finalization phase.

After rename Ghost FTP reads back `COLUMN_DISPLAY_NAME`. A provider result such as `file (1)` is not treated as a successful download when `file` was requested. If final-name verification fails, Ghost FTP best-effort deletes the unverified result and reports failure.

Any failure before final commit best-effort deletes the staging document. The download is reported as completed only after exact-name read-back succeeds. Generation ownership is also checked before a worker can publish completion, so a stale worker from a cancelled/destroyed transfer cannot clear or overwrite the state of a newer transfer.

All local download work remains inside the user-granted Storage Access Framework tree. No broad or all-files storage permission is introduced.

## Active transfer cancellation

Cancellation is deliberately fail-closed and phase-aware. The FTP data-transfer methods hold the session's protocol lock while transfer commands are active, so the UI does not wait on that monitor to send `ABOR` or graceful `QUIT`. During active data I/O, `cancelActiveTransfer()` remains non-synchronized and closes the passive-data and control sockets directly, interrupting blocked reads, writes, TLS data-channel setup, or completion-reply waits without freezing the UI.

The final-name boundary is stricter. `TransferCommitGate` tracks `TRANSFERRING`, `READY_TO_COMMIT`, `COMMITTING`, `CANCELLED`, and `FINISHED` phases. A cancellation in `TRANSFERRING` wins immediately and interrupts I/O. A cancellation in `READY_TO_COMMIT` wins before any irreversible rename: the worker cleans the staged object/document where that cleanup can still be performed safely and then closes the session. A cancellation in `COMMITTING` is rejected as too late; Ghost FTP does not close the control connection in the middle of an already-started `RNFR`/`RNTO` confirmation or describe a potentially committed final name as cancelled.

The same control connection is never reused after an accepted cancellation. A normal idle Disconnect still uses the regular graceful close path. Disconnect during a cancellable transfer routes through the same phase-aware cancellation lifecycle instead of a separate weaker path.

The UI also uses a monotonically increasing transfer-generation token plus the exact gate instance that owns the transfer. Workers must still own both before changing transfer UI state. Cancel and Activity destruction invalidate that ownership. Failure/success callbacks from stale workers therefore return without clearing a newer transfer's `busy`, progress, connection, or completion state.

Activity destruction requests the same cancellation lifecycle before shutting down the executor. If destruction wins before the final-name commit gate, staging is never committed. If the irreversible commit phase already won the gate first, it is not relabeled as a cancellation; the worker is allowed to finish that already-started finalization and then closes its detached session without publishing stale Activity state.

Cancellation does not claim impossible rollback semantics. If transport interruption makes the FTP control state ambiguous, a remote staging `.part` may remain and is documented as an orphan staging possibility. The final requested name is not committed by a transfer whose cancellation won the gate. Resume/restart-from-offset remains a separate feature and must not reuse a cancelled FTP session.

## Passive data-channel failure boundary

Every listing, upload and download depends on a new passive data channel negotiated through EPSV with PASV fallback. Ghost FTP treats failure during that channel setup as a session-boundary failure rather than assuming the existing control stream remains safe for another operation.

If EPSV/PASV negotiation cannot be completed, the passive reply is malformed, the passive port cannot be parsed or validated, the TCP data connection fails, or an FTPS data-channel TLS handshake fails, `openPassiveDataSocket()` closes the active data socket and hard-closes the FTP control session before returning the error. The Android UI then discards the dead session through the existing reconnect lifecycle.

This policy is deliberately conservative. It can require reconnect even when a particular server might have kept its control channel usable, but it avoids reusing a session after an ambiguous transport setup failure. There is no fallback to an unprotected FTPS data channel, no disabled hostname verification, and no acceptance of an invalid passive port.

The EPSV parser converts malformed/non-numeric ports into checked I/O failures rather than allowing a runtime parsing exception to escape outside the session cleanup path. The dedicated passive-data regression contract is executed by the Android APK workflow alongside the broader Android source contract.

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
