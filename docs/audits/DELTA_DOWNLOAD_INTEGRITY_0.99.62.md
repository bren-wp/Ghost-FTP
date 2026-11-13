# Ghost FTP v0.99.62 — delta-download security and dead-code review

**Baseline:** published v0.99.61 Preview, `main` SHA `9f3a55442eb1f8e32be770f0c41580ba86eb8d91`.  
**Scope actually examined:** Agent delta reconstruction, controller delta download path and common local staging/promotion, with targeted Rust tests and whole-repository static audit gates.  
**Evidence limitation:** exact-HEAD CI, production/staging installations, live cloud providers and adversarial local-process fault injection are separate. This is not a completed line-by-line manual audit or penetration test of the entire repository.

## Confirmed source risks and changes

| Priority | Finding | Remediation |
| --- | --- | --- |
| P1 | Shared delta reconstruction used `File::create` and unconditional `remove_file(out)` on failure, potentially truncating or deleting a pre-existing output or symlink. | Exclusive `OpenOptions::create_new(true)`, explicit ownership tracking and deletion limited to created output. Added file/symlink preservation tests. |
| P1 | Controller delta download used direct `tokio::fs::rename` to replace the final destination without the common on-disk size, durability and rollback path. | `commit_verified_delta_output` enforces byte count + sync and delegates to safe promotion with old-data preservation. Added mismatch/symlink/normal replacement regressions. |
| P1 | Agent could return more bytes than requested by a delta range; client did not reject an oversized reply before writing or updating counters. | Nonzero, at-most-remaining byte check in `checked_agent_delta_chunk_length` and checked overall byte counter. Added boundary tests. |
| P1 | The daemon's reused-byte count used unchecked summation and its delta error paths unconditionally deleted a temporary output even when exclusive creation failed. | Checked `try_fold` arithmetic, ownership-aware cleanup, destination file-type validation and daemon regression tests. |
| P2 | Patch file used non-exclusive create; error cleanup removed temporary paths whether they were owned or not. | Exclusive patch open, ownership-gated patch and output cleanup. |

## Maintainability, audits and remaining work

- Extracted `transfer/download_staging.rs` separates durable temporary files, symlink-aware checks and no-replace/rollback promotion from transfer scheduling.
- Updated `tools/audits/full-security-code-audit.mjs` to require safety markers in the new module and shared delta engine. Existing `full-dead-code-audit.mjs` remains advisory and cannot justify automatic deletion of dynamically called Rust/Tauri/Go/TypeScript APIs.
- The daemon's normal-file `rename` still relies on OS-specific replacement behavior; Windows overwrite and crash recovery need a separate transactional destination promotion and fault-injection tests before calling it equivalent to desktop rollback-safe promotion.
- Remaining P1 work: handle-relative OS-level protection against hostile ancestor-directory swapping, disk-full/crash/power-loss/fault injection, multi-GiB and multiple-provider concurrency validation, credential/cloud OAuth acceptance and accessibility testing.
- No change to profile/database migrations, saved credentials, user accounts, externally visible protocol APIs or existing release tags/assets. New Preview release may publish only from an approved merged and exact-tested commit.
