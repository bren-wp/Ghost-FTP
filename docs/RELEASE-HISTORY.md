# Ghost FTP release history

## 0.0.5 — 2026-09-12

Ghost FTP 0.0.5 focuses on lifecycle reliability, re-entry safety, Android connection ownership, truthful release documentation and optional browser companion source while preserving the established 0.0.4 desktop feature/security baseline.

### Windows reliability

- Added explicit in-flight ownership for encrypted profile persistence so duplicate save/delete commands cannot overlap and application close cannot terminate an active profile mutation mid-write.
- Preserved process shutdown across nested application-owned modal loops by reposting `WM_QUIT` rather than consuming it inside a dialog loop.
- Added local/remote file mutation guards around create-directory, rename, delete and remote permissions, with code-level protection behind UI state.
- Serialized Windows Remote Edit across open/save/reload/close continuations so asynchronous editor work cannot be re-entered into a parallel stale session.

### Android lifecycle stability

- Pending FTP/FTPS connection attempts are owned by the current Activity lifecycle.
- Activity destruction/recreation aborts the pending session through a non-blocking path.
- Late success/error callbacks are rejected so a destroyed Activity cannot be revived by an old connection result.
- Strict explicit FTPS, SAF-only storage and staged transfer safeguards remain unchanged.

### Browser companion source

- Added optional source packages for Chrome, Microsoft Edge, Opera, Brave, Vivaldi and Firefox.
- Supported `ftp://`, `ftps://` and `sftp://` link handling remains local and narrow, without telemetry, remote code, credential persistence, tab scraping or broad host permissions.
- Browser companion source is not part of the public Windows/Linux 17-file binary release.

### Documentation and release quality

- Reworked the root README around practical user value, downloads, privacy, Remote Edit, transfer control and security while retaining verifiable technical claims.
- Corrected generated release notes to the actual universal Windows + Debian/Ubuntu/Fedora/Portable Linux shape.
- Added a regression contract that rejects old architecture-specific Windows filenames and obsolete **12 platform artifacts / 15 public files** counts from current release notes.
- Maintains the canonical **14 platform artifacts / 17 public files** release shape, exact read-back, GHCR verification and latest-only retention.

### Security and privacy

- Preserves strict FTPS certificate/hostname validation, strict desktop SFTP host-key verification/pinning, trusted Linux AskPass provenance, rooted local path/transfer safeguards and protected-secret lifetime rules.
- Preserves no telemetry, analytics, advertising, fingerprinting, automatic crash upload, hidden backend or mandatory product account.

## 0.0.4 — 2026-09-11

Ghost FTP 0.0.4 focused on Windows/Linux parity, Android stability/security and release-evidence quality.

- Linux Light/Dark appearance parity and shared file sorting.
- Explicit Linux protected credential-save consent.
- Queued Top/Up/Down/Bottom priority and bookmark/profile-start workflows.
- Android bounded FTP/FTPS parsing and maintained strict TLS/SAF behavior.
- Read-only exact-head authentic UI evidence across Windows/Linux/Android.
- Canonical 14/17 Windows/Linux distribution with universal Windows files and distro-specific Linux packages.

## 0.0.3 — 2026-09-10

Ghost FTP 0.0.3 added real bandwidth-aware transfer controls and completed the public packaging transition.

- Independent upload/download bandwidth ceilings with conservative aggregate scheduling and real transport enforcement.
- Public Windows downloads reduced to universal Setup/Portable while retaining verified native x64/x86 payloads internally.
- Debian/Ubuntu/Fedora/Portable Linux packages promoted to canonical release artifacts.
- Canonical allow-list expanded to **14 platform artifacts / 17 public files**.
- Exact-head/exact-main gates, release read-back, GHCR verification and latest-only retention preserved.

## 0.0.2 — 2026-09-10

Ghost FTP 0.0.2 advanced reliability and navigation workflows.

- Non-destructive current-folder filtering, bounded recursive search and conservative directory comparison.
- Synchronized navigation only for safely proven paired ordinary directories.
- Stronger Windows comparison/session guards and safe legacy settings migration.
- Deterministic exact release-run/retention orchestration.
- Historical release shape: 12 platform artifacts / 15 public files.

## 0.0.1 — 2026-09-09

Ghost FTP 0.0.1 started the current public release line with native Windows/Linux FTP/FTPS/SFTP, Site Manager, dual-pane transfer workflow, Remote Edit, strict trust validation, rooted local safety, no application telemetry and exact-head release verification.

## Public history retention policy

Only the **latest public Ghost FTP version** is retained after successful publication and remote verification. Older releases, `ghostftp-v*` tags, superseded versioned release branches and obsolete package versions are removed by `.github/workflows/release-retention.yml` while current package/branch identities remain. Git commit history on `main` is not rewritten.
