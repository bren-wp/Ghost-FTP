# Ghost FTP v0.95.36 — Credential, continuous sync, script and editor audit

## Credential management

Ghost FTP already uses the Rust `keyring` package with `windows-native` (user-scoped Windows credential storage backed by the platform's protection model) and `linux-native` (desktop secrets service) rather than storing saved passwords in `profiles.json`. Only the native backend can read secrets; the UI cannot request a stored password in plaintext. The master-password-like workflow is **Argon2id encrypted portable backup**; it is not an alternate, persistent runtime master-password vault. Credential storage errors fail closed rather than silently writing unencrypted fallback data.

The 0.95.36 defense-in-depth change prevents even future CLI, migration or unreviewed code paths from serializing nonempty `AuthMethod::Password.password` or `AuthMethod::Key.passphrase` fields to JSON. Regression tests cover both secret types and clean metadata. Legacy profile JSON containing historical plaintext must still migrate via the existing command path; avoid claiming an all-device migration audit without testing historical snapshots and keyring-locked sessions.

## Continuous watched sync

Already implemented: recursive local `notify` watcher, 700ms burst debounce and a bounded 5s starvation window; a poll interval runs when events are lost or when syncing remote to local. Stronger protection in this version:

1. A watcher error triggers a rescan and still logs a warning.
2. The `sync_state` source index advances **only** after every transfer reports Done/Skipped. Error, Canceled, Missing and Timeouts refuse to acknowledge the pass.
3. Reserved staging files (`.ghostftp-download-*.part`, backups, delta temp, patch and new files) are never copied or Mirror-deleted.
4. Mirror retains existing empty-source and delete-count guards; additive remains the recommended web-design upload mode.

The app-level worker runs **while the application is open**; no Windows service or Linux daemon was installed. No file watcher can guarantee zero latency; debounce and network round-trip still apply.

## Generated scripts

In Settings → Sync & Backup, each normal pair offers PS/CMD/.NET watch buttons. Scripts use `ghostftp-cli sync` and saved profile references without embedded credentials. PowerShell single-quote doubling prevents injection. CMD rejects unsafe metacharacters and delayed expansion. Rejected: Mirror, exclusion patterns that the CLI does not reproduce, on-demand placeholders and invalid paths. The .NET FileSystemWatcher script is local-to-remote only, uses a 60-second poll fallback, and never issues Mirror deletion. A generated script is executable source: review the output before running it. The CLI needs to be in PATH and able to access the saved OS credentials.

## Editor and cloud

The currently implemented editor opens the user's external IDE/editor; syntax highlighting, bracket auto-closure and in-file search depend on that chosen application. In this increment only the upload watcher and staging security were improved: Unix private temp directories, bounded file-change signals, 10s retry check, source symlink refusal. Built-in syntax editor is a remaining feature, **not shipped**.

Windows/Linux adapter source already supports S3-compatible object storage, Backblaze B2 via S3, Google Cloud Storage, and HTTPS WebDAV. Implemented adapter source is distinct from validated provider login, large-object multipart behavior, API throttling, and vendor E2E acceptance. Confirm with real test accounts before claiming production-complete integrations.

## QA acceptance

Check exact-head CI for Cargo tests/Clippy and password safety regressions; TypeScript build, script quoting/injection checks and source-reachability; E2E FTP/FTPS/SFTP; Windows installer hardening; Linux lifecycle smoke; Android/macOS compatibility. Human acceptance still needs real cloud accounts, Windows locked/unavailable Credential Manager tests, Linux unavailable keyring tests, filesystem watcher reliability under IDE atomic saves, fast bulk-transfer benchmarking and 59 screen-by-screen visual comparisons.
