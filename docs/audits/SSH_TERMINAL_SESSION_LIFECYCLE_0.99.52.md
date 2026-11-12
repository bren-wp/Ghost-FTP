# Ghost FTP 0.99.52 — targeted SSH PTY lifecycle audit

**Baseline commit:** `a80a73bf7ac0e1696f6a2e6ac0eb6530bf1bce0f`  
**Source:** `ghostftp-desktop/src-tauri/src/terminal.rs`

## Source-verified findings

1. The original MAX_PTY_SESSIONS check inspected a map length before an awaited remote connection. Concurrent callers could all pass the check, exceeding 32. Replaced with an owned semaphore permit acquired before the first await.
2. The task ended without removing its map entry after remote EOF or errors, leaking terminal registry capacity. Added cleanup on task completion, with RAII permit release.
3. SSH ExitStatus emitted an exit event immediately; all tasks subsequently emitted a second event with a null code. The task now records the exit code and emits exactly once on completion.
4. Input and resize previously held a Tokio mutex guard across an awaited channel send. Both commands now clone the sender and release the guard before the await.

## Verification

The added capacity regression is source-level test coverage, not yet a measured runtime result. Native live-SSH concurrency, crash recovery and all 59 real-screen comparisons remain PENDING until independently executed. CI must verify the precise published source commit. The work does not constitute a complete security/dead-code certification.
