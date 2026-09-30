# Ghost FTP Transfer QA

## Implemented transfer engine scope

The authoritative Ghost FTP desktop source contains real FTP, FTPS and SFTP session/transfer paths, queue state, retries, pause/resume controls, bandwidth throttling, conflict handling and synchronization support.

The current transfer implementation also enforces:

- Completed rows do not expose Cancel.
- Skipped and Canceled are explicit terminal states.
- Retry All is available for failed transfers.
- Active queue filtering no longer duplicates terminal history.
- Progress is represented semantically.

## CI truth

Rust workspace tests and frontend production builds are release quality gates. Compatibility tooling does not simulate successful file transfers and is not used as proof of protocol correctness.

## 0.20.1 task lifecycle regression coverage

- Spawned transfer workers are registered through a race-safe helper that immediately drops an already-finished handle if completion won the spawn-to-registration race.
- The helper and `finalize()` share the same task-map lock, so a worker still finishing cleanup removes itself after registration releases the lock rather than leaving stale task state.
- A Tokio regression test covers the completed-before-registration interleaving.

## 0.20.0 restart/recovery contract

- Desktop transfer snapshots are persisted in SQLite without passwords, tokens or private-key material.
- Interrupted queued/transferring/paused rows are restored as explicit error/recovery rows after process restart rather than pretending to still be active.
- Recovered retries reconnect through the saved profile and existing OS credential store.
- A cross-process retry does not trust a historical byte offset unless file identity can be proven; the current safe behavior resets the first recovered retry to byte zero.
- Terminal history is pruned during runtime to a bounded budget while every active transfer remains retained.
- FTP/explicit-FTPS resumed uploads verify the final remote size and fall back to a safe full restart when a server accepts resume semantics but persists the wrong byte count.
- FTP/FTPS cancellation and chunk I/O failures issue ABOR/cleanup so the control connection remains synchronized for subsequent commands.

## Real-server acceptance still required

Before FINAL, execute against controlled test servers:

- FTP upload/download/rename/delete.
- FTP interrupted-transfer recovery/resume where supported.
- Explicit FTPS.
- Implicit FTPS where supported.
- Invalid/expired TLS certificate behavior.
- SFTP password authentication.
- SFTP private-key authentication.
- Unknown/changed host-key refusal and acceptance flows.
- Large-file transfer.
- Pause/resume/cancel/retry.
- Concurrent-transfer limits.
- Bandwidth limits.
- Overwrite/skip/rename conflict behavior.
- Timestamp preservation where supported.
- Optional checksum verification where both sides can calculate it.

Status: **production transfer implementation exists; real FTP/FTPS/SFTP end-to-end acceptance remains open before FINAL.**
