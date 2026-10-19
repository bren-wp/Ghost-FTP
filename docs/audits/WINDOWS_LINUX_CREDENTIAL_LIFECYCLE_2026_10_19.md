# Ghost FTP — Windows/Linux credential lifecycle audit — 19 October 2026

**Scope:** only existing merged-PR-#214 branch `fix/winlinux-stability-quality-v09548`. Its PR is already closed and merged, so these new commits are branch-only, **not merged**, **not released**, with unreleased product metadata 0.95.49 following published v0.95.48.

**Source reviewed:** `credentials.rs`, `credentials/linux_secret_service.rs`, `commands.rs`, `profiles/mod.rs`, `oauth.rs`, `backup.rs`, `db.rs`, `grant.rs`, cloud/SSH credential call sites, React IPC, Settings, Tauri packaging, database migrations and CI/release rules.

## Findings and fixes

| Lifecycle | Verified finding | Mitigation on this branch | Acceptance still pending |
| --- | --- | --- | --- |
| Windows write/read | keyring v3 uses Windows-native Credential Manager; changing service/account would orphan existing data | Keep `keyring::Entry` IDs and `com.ghostftp.credentials` unchanged, preserve old profile-password and OAuth account identifiers | Actual Windows Credential Manager normal/denied/restart/concurrent OS tests |
| Linux write/read | `linux-native` uses kernel keyutils, not a durable persistent Secret Service | New Linux writes require `secret-tool`/user D-Bus; reads check Secret Service then keyutils; idempotent copy with read-back; preserve old secret during migration | GNOME and KDE KWallet (Secret Service enabled); locked wallet/no daemon; reboot; concurrent processes; no unverified automatic deletion |
| Linux subprocess | `secret-tool` piped secret is private from argv but limited by upstream libsecret input implementation | Write through private stdin; restrict input to 8191 bytes; exact attributes and CLI option separator; no credential in stdout logs | Physical install and locked/headless behavior; high-value SSH key exceeding size must not silently truncate |
| Profile create/rotation | Vault write preceded SQLite manifest; a partial failure could omit an existing secret from backup inventory | Manifest record first, then vault write; errors propagate; retain previous account IDs | Power-loss tests between manifest/keyring/profile JSON writes; rotation rollback remains non-atomic |
| Profile delete/duplicate | Best-effort delete could remove metadata/manifest but leave secrets | Checked vault delete, only then manifest forget; failed duplicate cleanup signaled; unresolved manifest deliberately retained | Deleted profile metadata can precede denied credential deletion; orphan repair and explicit reconciliation needed |
| API key | Locked keychain could be presented as credential absence; manifest could be forgotten despite failed removal | Checked status and removal; manifest-first writes | Vault unavailable/denied operations and UI status on real machines |
| OAuth authorization/revocation | Token write not in DB manifest; failed profile save could leave orphan OAuth token | Token inventoried before writing; checked cleanup and manifest removal; unified Linux Secret Service/Windows vault | Real Google/Microsoft/Dropbox/Box consent/refresh and post-restart access |
| Grant private SSH key | Grant import put private material in OS vault before manifest | Manifest-first; checked failed-import cleanup; retain original key identifiers | Real grant import with power loss and locked vault |
| Export/import/backup | Backup enumerates recorded vault and token IDs; Argon2id 64MiB x3 + AES256GCM v1, bounded file/decompression, integrity preflight; staging could fail after vault writes | Shared vault backend and read-back; refuse conflicting staged restore; rollback old credentials on config/SQLite stage failure | Still NOT atomic when promoting actual live multi-file restore/WAL/SHM and old/new keys; real power-loss scenarios required |
| SQLite / WAL | Plain metadata database uses WAL + `secure_delete` and private owner permissions; migration can fail | Keep existing non-destructive migrations + SQLite integrity checks; two-phase manifest repair removes only **verified absent** entries, retains real orphan secrets | Disk-full, permissions, WAL/SHM tamper/rollback; user recovery from restore snapshots |
| Uninstallation | Installer/uninstaller policy may retain user data; deleting credentials while OS vault locked is unsafe | No automatic installer-side destructive credential wipe added; prior data preserved | User-visible explicit removal vs retain-data policy acceptance on Windows and Linux |

## Negative/fault injection tests

- `credentials.rs::tests::injected_vault_failure_never_forgets_sqlite_manifest`: simulated locked backend prevents SQLite manifest delete; successful and SQLite-fault branches covered.
- `credentials.rs::tests::orphan_audit_retains_existing_secrets_and_prunes_only_proven_absent`: no orphan secret deleted; only definitively empty records removed.
- `credentials.rs::tests::locked_vault_does_not_prune_any_manifest_record`: fully non-destructive when any vault entry is unreadable.
- `credentials.rs::tests::clearing_missing_entry_is_idempotent_but_locked_vault_fails_closed`: unavailable vault denial vs absent value, no sentinel leak.
- `backup.rs::tests::staging_fault_can_restore_old_vault_values_and_remove_new_entries` and `rollback_error_is_reported_without_exposing_sentinel_secrets`.
- `credentials/linux_secret_service.rs::tests::invalid_secret_inputs_are_rejected_before_launching_any_process` validates rejection of empty and oversized values.
- Existing backup wrong-password/corruption/SQLite-invalid and migration tests must also run. No test proves behavior on a real locked session without OS-level execution.

## Required real-platform test matrix (release blocking)

| OS | Setup | Positive and failure cases |
| --- | --- | --- |
| Windows 10 and 11 | Normal and restricted account; existing published v0.95.48 credential entries | Install/update, existing profile read, new write, change secret, concurrent CLI+GUI rotation, denied write/delete, logout/restart, signed backup, uninstall retain/remove data, blocked recovery; no process arguments or logs with sentinel |
| GNOME Linux | GNOME Keyring + running `org.freedesktop.secrets` + `secret-tool` | Create v0.95.48 keyutils entry first, upgrade to 0.95.49 branch build without losing it, check verified Secret Service migration, unlock/lock, reboot/logout, denied delete, preserve old until verified, validate DEB/AppImage process |
| KDE Linux | KWallet providing Secret Service API + `secret-tool` | Same migration, 4/8/16 concurrent session operations, locked/unavailable wallet, session D-Bus failure, RPM dependency/resolution, no plaintext secret |
| Offline / headless Linux | No `secret-tool` or no user D-Bus | New writes fail closed; legacy secret can still be read where available, never silently claim persistent save; backup/restore must not falsely succeed |
| Windows and Linux | Disposable provider accounts and synthetic sentinel secrets | Import/export/inspect/corrupt backup, wrong password, SQLite WAL/SHM corruption/denied rename/disk-full, SIGKILL during restore, reboots and idempotent retry; instrument `ps`, `journalctl`, Crash Report files, Tauri IPC and browser devtools |
  
### Release gate and limitations

This report is **source-audit evidence**, not live OS acceptance. No live Windows or Linux keychain session was accessible during this implementation; ignored actual-OS regression tests are added but have not been executed on those targets. A passing GitHub Actions check is necessary but insufficient to certify provider vault persistence, OS wallet unlock behavior, power-loss atomicity, OAuth revocation or uninstall cleanup. Preserve published tags and all existing profiles, encrypted backup v1 schema, Brendigo identity and migrations. **Never merge or release this branch solely because unit tests are green.**
