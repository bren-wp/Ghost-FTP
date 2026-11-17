# Ghost FTP — 0.99.69 source fingerprint and sparse 4 GiB assessment

Date: 17 November 2026. Status: **Scoped source change / awaiting exact-head CI and native acceptance.** Previous public Preview: v0.99.68.

## Finding: metadata equality did not prove content identity

`LocalUploadSnapshot` in `ghostftp-desktop/src-tauri/src/transfer/remote_upload_staging.rs` checked length, modification time and OS file identity. A local file can be rewritten in place with the same length and its original timestamp restored. The existing metadata guard could then accept modified content and promote the staged remote file. This is a risk for FTP, FTPS and SFTP staged uploads, not a demonstrated remote exploit.

## Mitigation

Capture a SHA-256 fingerprint of first/middle/last 64 KiB source windows while validating the same file handle and pathname's metadata before/after local sampling. Hash the entire source if no larger than 64 KiB. Verify the stored fingerprint at the start of each automatic retry and again before remote promotion. A mismatch refuses to promote. The extra sampling budget is at most 192 KiB per check regardless of source size, plus metadata checks and open/seek overhead. This is intentionally bounded to protect large-transfer responsiveness.

**Residual risk:** Samples cannot cover arbitrary mutations outside their ranges; neither a complete local hash nor a remote post-upload SHA-256 is present. A source or path may still be changed in a tight race outside the sampling/verification boundary. The file's mere metadata equality is not a sufficient security guarantee. FTP RNTO non-atomicity and provider-specific staging semantics remain unchanged.

## Regression expectations

| Test | Automated scope | Not proved |
| --- | --- | --- |
| Same-length rewrite with restored mtime | Local content fingerprint rejects metadata-preserving sampled mutation | All unobserved simultaneous writes |
| 4 GiB + 16 bytes sparse file (Linux) | 64-bit snapshot length and tail sampling, constant-memory scanning | 4 GiB real network upload/download |
| Existing FTP, FTPS, SFTP E2E | Real localhost protocol roundtrips | Internet reliability and arbitrary host behavior |
| Windows/Linux native build | Compilation and CI-host package smoke | Every target device, OS vault/reboot and accessibility |


## Dependency-security blocker observed 17 November 2026

The v0.99.69 PR initially failed all `npm ci` native jobs because the upgraded `vite@8.3.8` and `@vitejs/plugin-react@6.3.0` package manifest was inconsistent with its older lockfile. The one-time GitHub Actions lock regeneration now produced a matching, integrity-bearing `package-lock.json`; the temporary workflow was subsequently removed from the PR.

**Separate blocker, not resolved by regenerating the lock:** the strict development-dependency audit reported the upstream `picomatch` ReDoS finding [GHSA-fc5q-39xc-vjpf](https://github.com/advisories/GHSA-fc5q-39xc-vjpf), propagated through Vite, tinyglobby, micromatch and other build dependencies. At the time of review, there was no independently confirmed patched picomatch version. It would be unsafe to suppress the finding or use an untested major Vite downgrade merely to produce a green build. Track the required remediation in [Issue #299](https://github.com/bren-wp/Ghost-FTP/issues/299). Retain the independent fail-closed development audit and do not publish v0.99.69 until all mandatory exact-head gates pass.

## CVE-2026-112366 emergency local mitigation (17 November 2026)

Ghost FTP's frontend build graph contains three development-only picomatch copies (`2.3.2`, `4.0.4`, `4.0.7`). The published package versions remain covered by the [GitHub advisory](https://github.com/advisories/GHSA-fc5q-39xc-vjpf). No official upstream repaired release was confirmed on 17 November 2026.

The application now applies [an explicit local patch](../../ghostftp-desktop/scripts/patch-picomatch-redos.mjs) **after npm verifies its lockfile tarball integrity and before frontend tools execute**. It modifies only the known parser `analyzeRepeatedExtglob` callsite in these three pinned versions, rejects repeated-extglob bodies with an otherwise unhandled inner `*` wildcard rather than constructing an unbounded nested regular expression, and removes the optional bypass of the recursion safety check. Previously supported patterns of this risky shape may now be interpreted literally; expected normal file globs must retain their behavior.

The patched file content, package topology, pinned versions and bounded real matcher regressions must pass in `--verify` mode. A change to package paths/versions/parser structure or a failed 3.5-second adversarial matcher timeout **fails closed**. The strict `check-npm-dev-audit.mjs` still executes npm audit, rejects any unrecognized root advisory or package chain, and recognizes the specific GHSA **only if the patched installed copies pass verification**. Runtime dependencies are separately audited without dev packages.

**Caveat:** npm's published tarball version is still flagged as vulnerable by standard `npm audit`, because npm cannot detect a post-install local source patch. This mitigation is therefore not an upstream fix or proof of globally clean dependencies. No claim of zero vulnerabilities should be made. Remove it in favor of a genuine independently verified upstream release when available; rerun the full npm audit and Windows/Linux functional and security gates before publishing.

## Parallel network test implementation (17 November 2026)

Added [parallel-transfer acceptance instructions](../qa/PARALLEL_TRANSFER_STRESS.md) and `protocol_e2e.rs` test cases for 1/4/8 independent FTP, explicit FTPS and SFTP sessions. Each of the 39 short real-network roundtrips uploads/downloads a 1 MiB + 17 B bounded-memory deterministic stream, verifying full source/destination SHA-256 and received byte count. A distinct, **opt-in only** case exercises two parallel **over-4-GiB** real network transfers per protocol on a properly resourced test host; it is not automatically executed on standard GitHub CI.

**Evidence boundary:** Source changes do not prove a passing CI result or any physical 4 GiB network transfer. The latter requires retained test logs and SHA-256 outcomes from a dedicated host. Full application-queue cancellation and promotion under concurrent fault injection remain outstanding.

## Remaining independent acceptance

Run disposable FTP, explicit FTPS, SFTP and cloud hosts. Exercise 0 B, 1 B, 64 KiB, 128 KiB, 1 GiB, 4 GiB+1 byte uploads/downloads, with concurrency 1/4/8 and forced network, disk, permission, pause, cancel, reconnect and process crashes. Record source/remote full SHA-256 and file sizes. Confirm original remote files survive failed overwrite staging, and inspect orphan temporary files. Independently verify Windows Credential Manager and Linux Secret Service across logout/reboot and missing/locked vaults. Conduct real native keyboard, scaling and first-frame UI acceptance.

The previous repository check found `main` without branch protection and empty rulesets; enforce reviews and required CI in repository settings before Stable. This document does not claim those settings were changed.

## Dead-code/security audit boundary

The repository's automated reachability and static security regression tools must pass on the exact PR SHA. These scans identify candidates and unsafe patterns, but cannot certify absence of vulnerabilities or justify deleting apparent dead code without proving compatibility and reachable call sites.
