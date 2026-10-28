# Ghost FTP — Phase 1: fallback-state recovery safety (28 October 2026)

**Scope:** compatibility fallback runtime only: `tools/ghostftp-runtime/state.go` and `state_test.go`. This is not a full repository security certification. Audited after merged PR #241 (main `c78756525ac9dedd4fc5cd49bc0dffb85f9a276d`).

## Confirmed defect FS-001 — data-loss-prone empty-state fallback

Prior behavior: when `fallback-state.json` did not exist, `readPersistedState` tried `.bak` but returned an empty site list if **any** backup error occurred, even when the backup existed and was unreadable or contained corrupt JSON. A malformed `sites` or `settings` value in an existing file was silently replaced with `[]` or `{}` rather than triggering recovery. This risked displaying an empty workspace after a failed migration/interrupted write.

Remediation:
- Empty defaults are valid **only when both primary and backup do not exist**.
- Reject malformed root JSON, missing/invalid sites array, and missing/invalid settings object; recover from the last valid backup.
- Refuse symlinks, directories, and other non-regular persisted-state paths on read.
- When neither copy is usable, fail closed and retain the original bytes instead of silently resetting saved connections. API consumers receive an error rather than a successful empty state.
- Continue to sanitize sensitive fields in legacy saved-site data without recording them in new state writes.

## Confirmed defect FS-002 — repeated forensic snapshots

Previously, recovery used a timestamp-based corruption snapshot path and PR #241's follow-up used random names. The former could overwrite snapshots from the same second and the latter could accumulate identical copies every startup while the primary remained corrupt.

Remediation: a SHA-256 content-addressed corruption snapshot uses `O_CREATE|O_EXCL` and mode `0600`. Repeated startup with unchanged bad bytes produces one copy; genuinely different corrupted primary bytes produce distinct copies. Existing files are never truncated. Snapshots are best effort; they are not themselves backups of valid state.

## Regression coverage

- Two missing files return a fresh empty state without writing it.
- Missing primary with a corrupt backup returns an error, preserving the backup exactly.
- Valid backup recovers saved sites when primary is JSON null or has malformed sites/settings.
- Repeated recovery does not accumulate identical snapshots; distinct corrupt bytes stay separately preserved.
- Symlinked backups are rejected without modifying external files.
- Previous PR #241 tests still cover backup rotation, atomic replacement, directory and symlink destinations and predictable-temporary-file attacks.

**Verification:** these are implemented source tests; pass/fail must be recorded from the exact SHA GitHub Actions Go test results before merge. Cross-platform filesystems, power loss, malicious same-user concurrent mutation, and complete multi-store transactional consistency remain independent validation requirements.

## Release and platform limitations

- This hardening concerns the compatibility runtime, **not** an assertion that every Ghost FTP desktop, Android or macOS code path shares the same storage implementation.
- Native macOS FTPS/SFTP identity-verified implementations remain open in issue #223.
- The 75 screenshot concepts are not installed-app screenshots and have no complete real-UI 75/75 acceptance.
- `ghost-server/` is not yet implemented. There are no complete Ghost FTP Client ↔ Ghost FTP Server E2E results.
- Do not merge or announce a new complete release until the prior source edition is published and the exact-head quality/build/protocol/security gates pass.
