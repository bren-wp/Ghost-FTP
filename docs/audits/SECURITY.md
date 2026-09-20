# Ghost FTP Security Audit — RC10

Native saved-profile secrets are kept outside ordinary profile metadata and use the OS credential/keychain layer where supported. Ephemeral Quick Connect bypasses saved-profile persistence rather than briefly writing a profile and deleting it later.

The developer compatibility runtime uses a cryptographically random per-process mutation token, same-origin checks, bounded JSON input, strict field validation, protected-root destructive-operation guards and no persisted password/passphrase fields. It is not the production desktop GUI.

The native Tauri application keeps a restrictive CSP, avoids analytics/telemetry SDKs by default, and uses signed Tauri updater verification. The current updater endpoint in the production configuration is:

`https://ghostftp.com/updates/latest.json`

The updater public key remains embedded in `ghostftp-desktop/src-tauri/tauri.conf.json`; unsigned or invalidly signed updates are not accepted by the updater path.

## RC10 hardening verified in source

- Saved secrets remain separate from normal profile JSON.
- Ephemeral Quick Connect does not save a site automatically.
- Corrupt profile metadata is preserved/recovered instead of causing an unconditional startup panic.
- Failed SQLite startup quarantines the database and WAL/SHM sidecars before recreating a working store.
- The production release path excludes the old browser-host GUI.
- Native release publication is gated on the current quality workflow and successful Windows/Linux native build.
- RC10 release validation now checks the version in `package.json`, `package-lock.json`, `Cargo.toml` and `tauri.conf.json`.

## Historical compatibility verification — 20 September 2026

- Local compatibility API mutation without the session token returned HTTP 401.
- Authenticated state write returned HTTP 200.
- Attempt to persist a `password` field returned HTTP 400 and was not written.
- HTTP surface returned CSP, `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY` and `Referrer-Policy: no-referrer`.
- Native profile-store recovery preserves malformed primary/backup JSON and starts safely with an empty store.
- Native SQLite startup quarantines a failed `ghostftp.db` plus WAL/SHM sidecars before recreating a working database.

## Remaining security gates before FINAL

- Compile and retain the exact Windows/Linux release artifacts.
- Dependency/security scan of the resolved release dependency graph.
- Real FTPS certificate-failure acceptance.
- Real SFTP unknown/changed host-key acceptance.
- Target-OS signed-update/install acceptance.
- Production Windows code-signing decision and validation.

Status: **RC10 source security controls are present; target-OS protocol/update/signing acceptance remains open before FINAL.**
