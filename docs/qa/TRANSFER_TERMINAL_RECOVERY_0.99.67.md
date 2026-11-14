# Ghost FTP 0.99.67 — cancel/retry and durable ledger acceptance matrix

**Execution status: NOT YET VERIFIED on installed Windows/Linux products.** Use disposable servers and credentials; keep raw logs free of passwords, access tokens and signed URLs.

## Terminal state and UI event race matrix

1. Transfer uploads/downloads through FTP, explicit FTPS, SFTP and supported cloud backends. Trigger Cancel immediately before the final chunk, during the final response and just after completion.
2. After Cancel becomes terminal, delayed `transfer://progress`, network success or failure must not change the status or previously committed byte count. `transfer://done` must not be emitted for a canceled row.
3. Repeated Cancel on Done/Skipped must not turn finished results into Canceled. Manual Retry is allowed only on Error/Canceled; double-click Retry or call it concurrently and confirm at most one new transfer task.
4. Simulate an incomplete backend success: `Done` must be refused if measured byte length and the expected source length disagree.
5. Confirm the application panel and persisted SQLite transfer ledger always agree on terminal status and error confidentiality.

## SQLite and crash recovery

- Saturate 1/4/8 parallel transfers with progress, Pause, Resume, Cancel and Retry transitions. Restart the app while concurrent ledger writes are in flight; inspect SQLite row state and manually reattach the correct saved profile.
- Force process termination after a terminal record is committed but before a delayed progress update. Verify no stale `Transferring` row resurfaces as active/erroneously interrupted.
- Repeat on Windows NTFS and Linux ext4 with the application installed. Test storage-full, permission-denied, locked SQLite, failed fsync and unexpected power loss on an expendable test machine/VM; document exact recovery semantics.
- Never include passwords or OS-vault secrets in transfer history, crash recovery or diagnostic files.

## Local upload source identity across retry

- Use a known SHA-256 local source. After a transient network failure but before automatic retry, replace it with another file of the same size, edit content/mtime, remove it or replace it with a symlink. The original attempt must fail without replacing the final remote file.
- Repeat Pause/Resume and manual Retry. A manual Retry explicitly begins a new logical run and may read the current source; document this difference from automatic retries.
- Verify multiple remote staging paths have distinct unpredictable names, SFTP exclusively creates the new path, and final publication only follows successful size and source identity checks.
- Read back both sides and compare SHA-256 for 0/1 B, 64 KiB, 1 GiB and **4 GiB + 1 B** files on each protocol. Size alone cannot verify the contents.

## Required platform evidence

- Windows 10/11: Credential Manager save, app exit, reboot, reauthentication, invalid/revoked/renamed vault records and OS file locks.
- Linux GNOME/KDE: Secret Service locked/unlocked, D-Bus unavailable, secret migration, restart/reboot and session recovery.
- Prove installed Windows NSIS/MSI and Linux DEB/RPM/AppImage startup and test actual transfers. Collect test SHA, OS, server versions and redacted artifacts.

**No release/certification conclusion without these test results and the exact-head CI gates.**
