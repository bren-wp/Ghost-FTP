# Ghost FTP v0.95.37 — SQLite, OS secret storage and in-app source editor

## Local-first architecture and security boundary

Ghost FTP's Windows/Linux UI is local-first Tauri + Rust + React; it does **not** require a custom hosted backend merely for saving sites or sync configuration. Existing `ghostftp.db` uses bundled SQLite, WAL and forward-only, versioned migrations. SQLite improves transactionality and structured querying compared with rewriting an entire JSON file; SQLite **does not automatically encrypt its contents**. Keep saved connection passwords and private-key passphrases in the OS credential service (Windows native keyring with user-scoped protection; Linux native Secret Service). Portable encrypted backup is a separate Argon2id-protected workflow.

## Watched sync settings migration

The new v8 SQLite migration adds one `folder_sync_config` record. On first app launch after upgrade, read the legacy `foldersync.json` only if the database has no record; parse, then write to SQLite in one transaction. Leave the JSON file untouched as a recovery snapshot. A private migration marker prevents stale Mirror-enabled JSON pairs from being silently resurrected after database recovery/quarantine. A corrupt legacy file or SQLite value is surfaced as an error instead of silently destroying settings or substituting an empty set. Future edits, enable/disable, and removes are transactionally persisted to SQLite before updating worker state.

Test: migrate a real legacy file, modify stale JSON (SQLite remains authoritative), refuse corrupt DB config and refuse corrupt legacy files. Critical follow-up: app database recovery/quarantine routines need human review to ensure restoring only old legacy settings cannot silently re-enable retired sync pairs. No claim of total filesystem crash consistency or full database encryption.

## Built-in remote code editor

Remote file menu adds an in-app source editing option; external editors remain supported. Rust creates a private temporary editing session and exposes only `editor_read_text(editId)` / `editor_write_text(editId,content)`: no arbitrary filesystem path is accepted via IPC. Binary/non-UTF-8 and over-2-MiB files fail closed. Syntax tokens are rendered as React-escaped text, never injected HTML; test source preservation with HTML/script-like input. Features: PHP, JavaScript, HTML, CSS syntax highlighting, Ctrl+F find, search previous/next, automatic bracket pairs, indentation, Ctrl+S.

A write IPC result means only that the local temporary copy was written. Existing notify watcher performs the actual remote write; only its `editor://saved` event should clear pending state. Errors remain visible and the watcher retries. Runtime limitations include editor saves with repeated mtime, simultaneous external edits, OS resource races, Linux keyring availability, large binary documents, file encoding support, and DPI-by-DPI visual verification.

## Protocol and automation support

Existing cross-platform adapter source supports FTP, FTPS, SFTP, S3-compatible targets (including B2 S3), Google Cloud Storage, and WebDAV. Existing keep-up-to-date watcher and credential-free PowerShell/CMD/.NET scripts work without a hosted backend while Ghost FTP is running (script watcher remains running separately). Vendor-authenticated cloud login, multipart >4 GiB, high-volume transfer resilience and outperforming FileZilla are **not** accepted without E2E benchmark evidence.

## QA and dead-code gates

Use exact-HEAD rustfmt / Cargo test / Clippy; TypeScript noUnusedLocals and React production build; the source reachability scanner; native Windows/Linux installation and real protocol fixtures. The GitHub CI provides reproducible source/build evidence but does not automatically visually validate all 59 Windows/Linux references.
