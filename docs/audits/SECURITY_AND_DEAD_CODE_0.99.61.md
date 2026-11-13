# Ghost FTP 0.99.61 — security, dead-code and maintainability review

**Review baseline:** 0.99.60 Preview on `main` (commit `0aa555a10d94ed8a85c2032eec88b5f886fa8843`).  
**Scope examined:** Rust transfer and Agent Bridge request paths, their existing tests, repository static audit gates, active release/version policies, plus the previous 0.99.59 repository-wide inventory.  
**Evidence status:** Source-reviewed findings and committed fixes; **full exact-SHA CI, external penetration test, real provider acceptance, and complete line-by-line audit of all Rust/Go/TS code are not claimed**.

## Confirmed code defects and fixes

| Priority | Source and observation | Fix and evidence |
| --- | --- | --- |
| P1 — request validation hardening | `bridge.rs` used lossy header decoding, silently skipped malformed header lines and did not validate the HTTP request-line version/arity. A specific exploit chain has **not** been demonstrated. | `bridge/http_request.rs`: strict UTF-8, exact HTTP/1.0 or HTTP/1.1 framing, no folded or malformed headers, numeric-only Content-Length. New negative/positive parser tests; CI pending. |
| P1 — resource-exhaustion risk | `bridge.rs::serve` spawned a handler per accepted localhost client, with a 15-second request deadline but no concurrency bound. A local client could consume excessive tasks. | 32-connection semaphore with permits held through handler completion, rejection before spawning, deterministic permit-capacity test; CI pending. |
| P2 — concurrent directory race | `transfer.rs::ensure_download_subdirectory` treated `AlreadyExists` after an initial missing-path probe as an unconditional error, even when another valid transfer had just created the directory. | `transfer/path_safety.rs` now rechecks with `symlink_metadata` and permits only a genuine directory. Parallel-operation and Unix symlink tests; CI pending. |

## Unresolved security acceptance

- **P0 governance:** verify/enforce required branch protection/ruleset and exact-SHA checks. The ruleset list was empty during review; branch protection endpoint access was denied, so full branch policy cannot be certified here.
- **P0 Stable:** Authenticode, Linux artifact origin/trust, signed update identity, installed upgrade/rollback, and manual Windows/Linux installer acceptance require independent evidence.
- **P1 transfer TOCTOU:** path-by-path `symlink_metadata` and `create_dir` cannot eliminate an actively malicious local process swapping ancestor entries between checks. True adversarial parent-directory protection needs handle-relative OS-specific APIs and fault-injection tests.
- **P1 cloud/credentials:** authorized real-provider E2E, OAuth renewal/revocation, locked credential vault and migration tests remain unverified for this change set.
- **P1 protocol integrity:** concurrent multi-GiB transfers, crash/power-loss/disk-full, remote metadata changes and heterogeneous FTP/FTPS/SFTP servers require explicit acceptance.
- **P1 UI:** Windows/Linux DPI, keyboard and screen-reader matrix and real-reference comparison remain pending.

## Dead-code and architecture inventory

The existing CI scripts `tools/audits/full-security-code-audit.mjs` and `tools/audits/full-dead-code-audit.mjs` inspect source trees and emit reports. They are **static heuristics**, not proof of exploit absence or safe deletion. This PR updates source-level security contracts and retains the tests; use exact-HEAD CI output as the authoritative scan result. **No functions or files have been deleted solely due to a static unused-code indication.**

Prior `v0.99.60` successful full-dead-code CI reported **685 working-tree files** (117 TypeScript, 84 Rust, 13 Go, 74 operational scripts) with **zero unreferenced non-public function candidates** in its limited detector categories. This is historical baseline evidence, not a result for the current 0.99.61 head; unreferenced dynamically invoked APIs and platform-specific code may escape this heuristic. Credential-vault/export helpers have also been extracted into `commands/profile_security.rs`, with legacy key-identity tests. 

Maintained large-file hotspots remain `transfer.rs`, `bridge.rs`, `commands.rs`, `i18n.ts`, `styles.css`. The first two have now begun conservative extraction by domain while preserving existing caller contracts. Next candidate boundaries: local staging/promotion, transfer retry scheduling, bridge route handlers, OAuth commands, locale catalogs and CSS component scopes — each should have characterization tests *before* extraction.

## Acceptance checklist (not yet evidence of passed tests)

1. Exact PR SHA: Rust fmt/tests/Clippy, TypeScript/build and Go race/vet.
2. Full security and dead-code CI reports: review false positives, never delete dynamic Tauri/OS paths automatically.
3. FTP/FTPS/SFTP E2E and Windows hardening; native Windows/Linux builds and package checks.
4. No profile, vault secret, connection or user-data migration.
5. After authorized merge, verify new Preview assets, SHA-256, immutable tags and updater metadata.
