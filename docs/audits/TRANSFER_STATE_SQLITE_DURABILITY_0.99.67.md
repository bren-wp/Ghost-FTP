# Ghost FTP 0.99.67 — transfer state / SQLite durability security review

**Scope:** transfer manager source changes after published v0.99.66. **Status:** source-level design review, not a penetration test or proof of complete protection against every concurrent mutation, power failure, server behavior or hostile filesystem.

## Confirmed correctness defects addressed

| Severity | Root cause | Mitigation |
| --- | --- | --- |
| High | `finalize()` previously read the transfer status, then later wrote `Done` or `Error`; a concurrent Cancel could be overwritten. | A single guarded `finish_transfer()` transition refuses terminal rows under the same state mutex, and the UI emits the resulting committed snapshot. |
| High | Concurrent `update()` calls could persist an older `Transferring` snapshot after a more recent terminal SQLite write. This could cause incorrect crash recovery. | `ledger_serial` orders state mutation and `transfer_ledger_upsert`, including prune and insert, without holding the transfers mutex during SQLite or pruning. |
| High | A late progress/transport callback could mutate the byte count or status of a completed, skipped, canceled or failed row. | Ordinary updates cannot alter terminal rows. Only an explicit `reset_for_retry()` permits `Error`/`Canceled` → `Queued` with byte count reset. |
| Medium | Repeated Retry actions against a terminal row might enqueue the same transfer more than once or abort a newly started worker. | `retry_serial` orders user-triggered Retry requests and the terminal-to-queued reset checks status atomically. |
| Medium | Each automatic FTP/SFTP attempt previously captured a fresh source identity; a changed source between transient retries could silently change the uploaded revision. | Capture one local identity snapshot per logical automatic retry runner. Validate it on each attempt, the opened source handle, and before remote promotion. |

## Regression and release evidence

- Unit tests guard Cancel/Done and Cancel/Error interleavings, late progress after Done, durable SQLite status, rejected short successful transfers and manual requeue from failed/canceled only.
- Full-repository static security-code audit has required-source contracts for `ledger_serial`, immutable terminal guards and `finish_transfer`; a passing static scan does **not** replace manual security review.
- Exact-head Rust fmt/tests/Clippy, Windows filesystem/security, FTP/FTPS/SFTP E2E and native Windows/Linux build status must be taken from CI for this PR only.

## Remaining limits

1. FTP RNTO and competing writers do not offer a universal atomic rename-if-absent guarantee; remote server behavior must be tested.
2. Source file identity checks are metadata only; malicious same-size writes with forged timestamps need stronger validation (e.g. cryptographic readback) and cannot be claimed prevented.
3. A completed server-side rename can still race with a user attempting Cancel; once a server has committed a remote mutation the client cannot undo it safely. UI must never falsely claim a canceled worker was undone on the server.
4. Failed uploads can leave unique remote `.ghostftp-upload-*.part` staging files. Do not automatically delete unverified ownership.
5. OS/power-loss durability depends on SQLite configuration and filesystem guarantees beyond merely ordering application-level writes; prove separately with hard-kill/power-loss tests.
6. Full end-to-end `4 GiB + 1 byte`, 1/4/8 parallel, SHA-256 and native Windows Credential Manager/Linux Secret Service acceptance remains manual.

**Release rule:** A green CI is required but is not sufficient to claim a flawless production build, complete security audit, or true data-loss immunity.
