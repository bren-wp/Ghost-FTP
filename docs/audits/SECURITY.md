# Ghost FTP Security Audit — RC3

Native saved-profile secrets are moved into the OS credential/keychain layer before profile metadata is written. Ephemeral Quick Connect now bypasses profile persistence entirely rather than briefly writing a profile and deleting it later.

The compatibility runtime uses a cryptographically random per-process 256-bit token for mutation requests, constant-time token comparison, same-origin checks, 1 MiB JSON limits, strict unknown-field rejection, loopback-only listening, security headers/CSP, protected-root destructive-operation guards and no persisted password/passphrase fields.

Executed fallback tests confirmed an unauthenticated mutation returns HTTP 401, SHA-256 is correct, normal local operations succeed with the session token, and deletion of `/` is refused.

The native update configuration remains pinned to `https://ghostftp.com/ghostftp-updates/latest.json` with Tauri updater signature verification. A final security sign-off still requires compiling the exact native release artifacts, dependency/security scanning of that resolved dependency graph and target-OS update/install testing.

## RC4 verification — 20 September 2026

- Local compatibility API mutation without the session token returned HTTP 401.
- Authenticated state write returned HTTP 200.
- Attempt to persist a `password` field returned HTTP 400 and was not written.
- HTTP surface returned CSP, `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY` and `Referrer-Policy: no-referrer`.
- Native profile-store recovery now preserves malformed primary/backup JSON and starts safely with an empty store rather than panicking.
- Native SQLite startup now quarantines a failed `ghostftp.db` plus WAL/SHM sidecars before recreating a working database.
