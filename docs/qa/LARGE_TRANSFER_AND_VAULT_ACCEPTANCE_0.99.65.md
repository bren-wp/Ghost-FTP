# Ghost FTP 0.99.65 — Windows/Linux acceptance matrix

**Status: NOT YET VERIFIED.** Use disposable test hosts, profiles and test credentials; do not overwrite production data. Record CI head SHA, OS, server version, transfer IDs, file lengths, SHA-256, error logs (redacted) and before/after screenshots.

## FTP / FTPS / SFTP overwrite fault injection

1. On each disposable endpoint seed `existing.bin` with a known SHA-256. Choose a different local file with another SHA-256.
2. Run **Overwrite** and interrupt during the upload by network drop, socket reset, Pause, Cancel, clicking Disconnect and killing the desktop process. Check that the *existing final remote file* retains the original checksum/length. Check and record any leftover `.ghostftp-upload-*.part` files, without deleting unrelated files.
3. Retry on a healthy network. Verify final file SHA-256 equals local source and uploaded size matches exactly. If remote rename-over-existing is unsupported, operation must fail while preserving original; never silently remove it.
4. Test Skip/Rename target collisions, symlink or path traversal names, hostile CRLF input and a pre-existing staging path. Run a competing writer against the final name and document FTP race limits.

## Very large and parallel transfers

- Upload/download 0 B, 1 B, 64 KiB, 128 KiB, 1 GiB, and **4 GiB + 1 byte** samples.
- Run **1 / 4 / 8 concurrent** transfers of independent files, followed by two parallel uploads to the same remote final path. Confirm that user-selected policy is respected, one transfer's Cancel does not cancel unrelated work, and progress never claims a false 100%.
- Test server restart, network loss, timeout, exhausted disk quota and repeated pause/resume/retry/disconnect while many transfers are active.
- Compare local and remote SHA-256 and exact byte counts after every successful run (size alone is insufficient), check temp-file leak count and error confidentiality.

## Keep connected until Disconnect

- Open several saved and temporary sites. Let server idle beyond control socket timeout while app remains open. Verify FTP/FTPS auto-reconnect performs a new authentication and health probe, tab remains open, SFTP transport reconnects if dead, and failed reconnect shows Unhealthy rather than false Healthy.
- While a probe or reconnect is in flight, click **Disconnect**. Verify this session is not resurrected after the request, and other sessions/tabs continue working. Repeat suspend/resume, host sleep, network changes and rapidly switching tabs.
- On desktop application exit, OS sockets close. On next launch, **saved profiles** remain but TCP connections need to be re-established; never claim 24/7 background connectivity.

## Real OS credential vaults

- **Windows 10/11:** Save a disposable FTP/SFTP password through Ghost FTP and confirm OS Credential Manager entry. Close/relaunch app, reboot system and reconnect from saved profile. Test renamed profile, missing/deleted vault entry, rejected/expired credentials and multiple profiles, with no secret in exported JSON or logs.
- **Linux GNOME and KDE:** Test user D-Bus session with unlocked Secret Service provider, close/relaunch/reboot and reconnect. Test locked vault, missing D-Bus, migration from previously present keyutils credential and no silent volatile fallback. Verify that failed vault access does not erase profiles/manifests.
- Run native credential tests **only with disposable test credentials**, explicit cleanup, and a real logged-in desktop user; headless CI is not evidence of complete OS-vault compatibility.

## Required release gates

Exact-commit GitHub Actions must pass Rust fmt/tests/Clippy, Windows hardening and native build, Linux native build, TypeScript/React compile and source reachability scanner, protocol E2E and dependency/security checks. Native installed-product manual matrix above must be evidenced in release notes. Do not sign off on every option or promise 100% portability from code review alone.
