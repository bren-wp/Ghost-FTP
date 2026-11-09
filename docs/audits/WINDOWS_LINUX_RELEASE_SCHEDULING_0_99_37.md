# Ghost FTP 0.99.37 — Desktop Windows/Linux resilience audit

**Date:** 9 November 2026. **Status:** Source-level improvements; full acceptance depends on CI results for exact PR head SHA.

### Production issues addressed

1. GitHub release upload accepted 21 assets for v0.99.36 but a later HTTP 422 left a draft. Exact source commit, the checksum manifest and 21 remote SHA-256 digests were independently verified before the draft became public. The tag and already-uploaded binaries were never overwritten.
2. The existing publisher deleted draft assets on rerun. It now reuses matching content immutably, retries ambiguous upload responses and refuses conflicts rather than resetting an entire release.
3. A skipped downstream metadata workflow-run was incorrectly selected in place of successful exact-source push verification. The release gate now selects push events only.
4. Scheduled desktop transfers previously permitted malformed time strings to enter saved settings, and daily/weekly DST gaps could silently shift the intended hour. Strict clock and calendar validation now fail closed on nonexistent wall-clock times.
5. Editing an armed schedule could act on stale last-run state. Schedule edits now rebase the checkpoint.

### Source test inventory

- `ghostftp-desktop/scripts/check-transfer-schedule.mjs`: invalid dates/clocks, Europe/Zagreb DST gap/repeat, once/daily/weekly.
- `.github/scripts/check-release-governance.test.mjs`: immutable tags, digest checks, no draft asset purge, exact-source push gate.
- Existing suites: Rust transfer/staging/host-key guards; React/TypeScript interaction/IPC controls; Go tests; Linux/Windows packaging; macOS CI; FTP/FTPS/SFTP localhost E2E.
- Static safety/dead-code inventory is advisory and does not prove every external attack path or dynamic entrypoint.

### Remaining acceptance and limitations

- Real FTP servers (different implementations and TLS setups), real SFTP/Agent high-load interruption tests, and live OAuth/cloud tenants remain outside of mock/localhost assertions.
- Signed updater credentials, OS install/upgrade/rollback, and third-party penetration testing need independent evidence.
- README's existing native Windows and Linux screenshots are authentic captured binaries from older revisions, not 0.99.37 visual tests. All 59 Windows/Linux design concept screens remain unverified for pixel-perfect 1:1.
