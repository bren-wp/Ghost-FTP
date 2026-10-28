# Ghost FTP master development — verified phase ledger (28 October 2026)

This is an **execution record**, not a claim of full product acceptance. Source baseline: main merge `c78756525ac9dedd4fc5cd49bc0dffb85f9a276d` (v0.99.17). Current work: [PR #242](https://github.com/bren-wp/Ghost-FTP/pull/242) (v0.99.18 Preview).

| Phase | Source/implementation evidence | Acceptance status and next gate |
| --- | --- | --- |
| 1. Stabilization | PR #241 atomic backup rotation merged; seven exact-head PR workflows passed before merge. PR #242 adds fail-closed recovery tests, symlink rejection and corruption-snapshot deduplication. | **In progress.** v0.99.17 was published and verified. Re-run exact-SHA CI for #242, including Go race detector and concurrent-state stress tests. Hardware/Windows disk-fault injection and the native macOS protocol security issue #223 remain open. |
| 2. Client functionality | `ghostftp-desktop/`, `android/`, `macos/`; Rust `remotefs/ftp.rs`, `remotefs/sftp.rs`, native Android and Swift client implementations. | **Actively implementing in #242; not fully accepted.** Windows/Linux Rust FTP/SFTP mutating commands now reject unsafe paths; native Swift FTP upload inventory requires MLSD or LIST -a before STOR and rejects Windows-path filenames. Rust/Swift tests added. Real multi-server and installed-app results are required. Native Swift macOS FTPS/SFTP issue #223 is still a release blocker; do not downgrade to plaintext. |
| 3. Pixel-perfect 75 screens | `docs/qa/premium-75-screen-contract.json`, `PREMIUM_REFERENCE_EVIDENCE.json`, `tools/qa/compare_reference_v8.py`. The supplied (10) ZIP SHA-256 is `1cdaea3e60c0ec4b14c3e8fa515351ba0710ee9022060e7f107b8bb9fbf6e877`, identical to archived (8). | **0/75 human-accepted real installed captures.** 30 Windows, 29 Linux, 16 Android design references exist; preserve exact dimension/DPI/state and separate illustrative artwork from real captures. No approved macOS references. |
| 4. New Ghost FTP Server | No `ghost-server/` module in baseline tree. | **Not implemented.** Create a separate actual FTP/FTPS server engine, identity/auth/authorization, admin UI, Windows service and portable execution with security tests, not mock screens. |
| 5. Client ↔ Server integration | Existing `.github/workflows/ghostftp-protocol-e2e.yml` covers protocol fixtures. | **Blocked** for the new Ghost FTP Server until an actual server exists. All four clients need identity, transfer integrity, interrupts and authorization tests against it. |
| 6. Security, dead code and performance | Existing `tools/audits/`, `docs/audits/` and platform checks; scoped new finding `docs/audits/FALLBACK_STATE_RECOVERY_2026_10_28.md`. | **Incomplete for the full new scope.** Keep critical issues blocking, preserve user data, document exact-SHA findings and run security, migration, CVE, release chain and performance QA across each target OS. |
| 7. Distribution | v0.99.16 public preview carried 25 assets (client builds, source, checksums and QA); 0.99.17 release published with 25 GitHub assets. | **Not a complete Client+Server release.** Windows Server portable/setup do not yet exist; verify each client/OS package, signing status, checksum and installability separately. |
| 8. GitHub release | PR #241 merged; v0.99.17 source on main; PR #242 staged for v0.99.18. | **Not complete.** Public v0.99.17 is verified; PR #242 exact-SHA gates must pass before merge and any subsequent preview release. Never repoint immutable tags. |

## Deterministic work order

1. v0.99.17 is now publicly published (25 artifacts); keep its immutable tag and source provenance unchanged.
2. Run and fix all PR #242 tests, including Go race detector, cross-platform Rust mutations, native Swift macOS FTP hidden-file preflight and live protocol E2E. No bypassing failing gates.
3. Review open macOS issue #223 and ship protocol identity verification only with real positive/negative E2E.
4. Continue functional UI audit and capture true installed screen states before visual acceptance.
5. Implement Ghost FTP Server as an isolated product, then test platform client/server matrices.
6. Close release security/performance/localization and package gates before calling any full cross-platform release complete.

Evidence and remaining limitations must be updated after each source change; a passing static inventory alone is not feature parity, pixel parity or release readiness.
