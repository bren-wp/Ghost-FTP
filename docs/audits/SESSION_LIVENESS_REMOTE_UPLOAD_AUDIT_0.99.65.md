# 0.99.65 source security audit — session liveness and remote uploads

**Scope:** Ghost FTP PR #293 vs. v0.99.64, 14 November 2026. This is a **scoped source audit**, not a complete penetration test, independent code verification or certification that all options work on all operating systems.

## Confirmed source findings and changes

| Priority | Root cause | Change |
| --- | --- | --- |
| High | FTP and SFTP whole-file overwrites used the final remote pathname during network write, risking an incomplete original on interruption. | Unique random sibling staging file; SFTP CREATE/EXCLUDE, FTP absence preflight, exact byte-length checks, then server-side rename only after transfer completion. |
| High | An otherwise live FTP tab could be left with a dead idle control socket and a failed NOOP. | Probe every opened FTP/FTPS/SFTP session while desktop is running; failed FTP/FTPS NOOP attempts one bounded re-authentication and re-probe. Failed reconnect leaves session Unhealthy, not falsely Healthy. |
| High | A concurrent liveness check or reconnect could outlive the user's Disconnect request. | One-way closed flag for FTP and SSH, checked before/after reauthentication; SSH disconnect serializes with the reconnect lock. |
| Medium | Upload retry persistence did not explicitly capture permission to overwrite after restart. | Add `allow_replace` serialized field (defaults to false for legacy data). |
| Medium | Heuristic unused-source discovery can flag legitimately dynamic Tauri IPC/UI entrypoints. | Preserve static TS reachability and Rust compiler/Clippy checks; don't delete source purely from filename heuristics. |

## Existing controls reviewed, preserved

- Desktop loads saved profiles from the durable native profile store at application startup, but does **not** persist connected TCP sessions across process exit.
- Stored credentials use platform credential vault APIs where available; profile JSON should not contain plaintext secrets. Linux Secret Service is a distinct persistent backend with legacy keyutils migration safeguards.
- Download staging is a separate code path; these changes do not edit existing user credentials, delete profiles or rewrite existing download history.

## Known gaps / security limits

1. **FTP atomicity:** FTP STOR has no exclusive create and FTP RNFR/RNTO is not atomic rename-if-absent. Server-side concurrent modifications and staging-path preflight races are not solved universally.
2. **Server compatibility:** FTP/SFTP rename-over-existing may fail. Never delete the original to force promotion. Verify old remote bytes remain intact and report the failure.
3. **Remote temporary files:** Cancel/failure may leave `.ghostftp-upload-*.part` on the remote host; there is no automatic deletion without reliable ownership and age checking.
4. **Other backends:** Cloud, Agent, WebDAV, object stores and SaaS uploads are not covered by the FTP/SFTP change. Each needs its own atomicity, resumability and cleanup assessment.
5. **Native credentials:** Windows Credential Manager and Linux Secret Service unlocked/reboot/revoked/locked-vault behavior requires real desktop-user sessions; unit mocks are insufficient.
6. **Size and concurrency:** Full >4 GiB, 4/8 concurrent transfers, SHA-256 verification, network resets, process-kill fault injection and simultaneous destination collisions remain release acceptance.
7. **Full security/dead-code audit is not yet verified:** Existing `npm run check:dead`, `npm run check:ui`, TypeScript, Rust formatting, Clippy and CI protocol gates provide the baseline. Source changes were checked for obvious race conditions, unsafe paths and credential exposure, but complete app-wide dynamic security and dead-code auditing needs those gates plus manual inspection and installed-platform testing. Do not claim a zero-vulnerability result.

**Release rule:** No stable-production assertion and no release publication based solely on this source review.
