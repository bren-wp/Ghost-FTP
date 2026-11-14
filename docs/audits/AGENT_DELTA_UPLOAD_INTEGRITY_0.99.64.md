# Ghost FTP 0.99.64 — Agent delta upload integrity review

**Baseline:** published v0.99.63 Preview, main `3bb8f950fe9adbf17c0ca7cc55083d33fc16abaa`.  
**Scope:** Agent delta upload in `src/transfer.rs`, shared local staging at `src/transfer/download_staging.rs`, focused Rust regressions and static-security audit guards.

| Priority | Confirmed issue | Remediation |
| --- | --- | --- |
| P1 | Delta-upload planner used `std::fs::File::create`, potentially truncating an occupied output or following a local symlink; failed creation was followed by unconditional deletion. | Exclusive `OpenOptions::create_new(true)`, distinct unowned/owned failure paths, existing-file/symlink tests. |
| P1 | Delta patch transfer accepted any `Response::Written` regardless of reported byte count. | `verify_agent_delta_patch_ack` rejects short and excessive acknowledgements; tests cover boundaries. |
| P1 | Controller accepted `DeltaDone` without checking reported reused/written bytes or a fresh remote size. | `verify_agent_delta_upload_completion` and `agent_stat` verify all counts after assembly; tests cover mismatches. |
| P2 | Cleanup after patch planning and transmission unconditionally unlinked its chosen path. | `cleanup_owned_regular_staging_file` avoids unlinking an entry that is now a symlink or special file. |

**Persistence:** Existing saved profiles, application startup rehydration, profile schema and OS credential-store access remain unchanged from v0.99.63; no credential migration or user-data reset is part of this patch.

**Remaining risk:** A hostile local process able to race pathname replacement can still defeat check-then-use filesystem operations; use handle-relative OS APIs for stronger race resistance. Source-based audit is not a complete manual security audit. Test real OS-vault reconnection after restart, large interrupted transfers, provider auth flows, installers and upgrade/rollback on installed Windows/Linux devices before claiming complete production acceptance.
