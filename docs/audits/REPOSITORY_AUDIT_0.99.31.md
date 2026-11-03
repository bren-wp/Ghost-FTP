# Ghost FTP 0.99.31 — Full-source dead-code and security audit (in review)

**Audit date:** 3 November 2026  
**Base:** published [v0.99.30](https://github.com/bren-wp/Ghost-FTP/releases/tag/v0.99.30)  
**Proposed changes:** [PR #260](https://github.com/bren-wp/Ghost-FTP/pull/260) (unmerged draft until latest checks succeed)

## What “full” covers—and does not

The repository-wide automated audit inventories every checked-in active source module and release script it understands; TypeScript/import reachability, Rust all-target compilation, Android Gradle lint/unit tests, native Swift compile/tests, Go tests/race/vet, protocol E2E and native release packaging are independently gated by CI. Heuristic orphan reports are **advisory**. It is not a penetration test or a proof that every visible control works on every device; remote provider credentials, physical hardware, OS migration and network failure scenarios remain separate.

## Baseline inventory and evidence

[Successful v0.99.30 full-repository quality run #41540800880](https://github.com/bren-wp/Ghost-FTP/actions/runs/41540800880) ran the full source inventory and static security regression suite, with downloadable audit JSON artifacts. The source inventory reported:

| Class | Files |
|---|---:|
| Tracked working-tree files | 884 |
| TypeScript/TSX | 117 |
| Rust | 84 |
| Kotlin production | 18 |
| Kotlin tests | 22 |
| Android resources | 150 |
| Swift production | 17 |
| Swift tests | 9 |
| Go | 13 |
| Operational scripts | 79 |

The report found zero candidate orphan Kotlin/Swift modules, zero candidate Android resources, zero single-reference nonpublic functions in each of the Kotlin, Swift, Rust, TypeScript and Go scans, and **one candidate orphan operational script** (`ghostftp-desktop/scripts/generate-funding-qr.mjs`). The npm production-dependency audit and development dependency high/critical policy also passed then. These values describe the *published baseline*, **not** a completed v0.99.31 head audit.

## Confirmed findings and changes in 0.99.31

1. **High — filesystem symlink redirection (Go compatibility runtime):** earlier destructive path checks protected root/home names but did not reject symlinked path ancestors. Earlier direct and recursive `chmod` could follow symlink targets and affect unrelated files. Added ancestor `Lstat` preflight; direct and recursive chmod reject links, and recursive preflight rejects nested symlinks *before* modifying the tree. Go regression tests cover linked file/parent and preservation of the external file's mode. **Residual:** preflight is not atomic; an attacker able to race directory replacement still requires OS-specific no-follow file descriptors for stronger guarantees.
2. **Medium — trailing JSON accepted (Go local API):** `decodeJSON` previously decoded the first object without requiring EOF. It now rejects appended values and malformed trailing bytes, matching the already stricter Go installer parser. Tests cover second objects, trailing booleans, unknown fields and valid whitespace.
3. **Medium — platform configuration regression blind spot:** security scanning checked known trust bypass patterns and text guards but did not independently validate Tauri updater/CSP/capabilities, Android manifest and data-extraction exclusions, cloud HTTP redirect policy, or macOS device-only Keychain configuration. Added `inspectPlatformSecurityConfig` with deliberately weakened configuration tests; the audit fails if these controls drift.
4. **Low — verified unreferenced operational script:** the sole source-inventory candidate, a legacy QR generator for donation addresses, had no other repository references, no generated funding assets, and no README links. Removed this unused, manually-invoked generator without touching user data, keyrings, transfer/restore code, live UI or migration paths.

## Matrix of protected surfaces

| Platform | Enforced/inspected | Remaining acceptance |
|---|---|---|
| Windows | Tauri CSP, Rust IPC and secret redaction, native credential storage, NSIS/MSI lifecycle, file/transfer operations, updater verification | Authenticode and installed-device upgrade/rollback, heavy FTP/SFTP fault injection, OS no-follow race hardening |
| Linux | Tauri CSP, package bundles, Secret Service interaction paths, local file permissions, protocol E2E, AppImage/DEB/RPM QA | Locked/missing keyring, multiple distribution versions and desktop portals, installer rollback, symlink-race scenarios |
| Android | Manifest backup/cleartext restrictions, export boundaries, Play release signing gate, cloud DNS/redirect protection, Gradle lint/tests | Physical devices, SAF provider failure tests, production Play signing and in-place migrations |
| macOS | Device-only non-synchronizing Keychain, saved profile restore, Swift tests, cloud signed URL peer security and package gates | FTPS/SFTP unsupported pending verified native peer trust; Developer ID/notarization, hardware tests |
| Go compatibility | Local API loopback/host/origin and token, bounded JSON, local filesystem operations, vet/race tests | Independent threat modelling and OS-specific atomic no-follow filesystem handles |

## Remaining mandatory verification

- Run `node tools/audits/full-dead-code-audit.mjs --json` and `node tools/audits/full-security-code-audit.mjs` against the **latest PR head**, with both respective test suites. Inspect advisory candidates; do not delete reflective, dynamically dispatched, feature-gated or migration code from textual searches alone.
- Verify exact-head quality, all platform builds, Windows hardening, Rust workspace, Go race/vet, protocol E2E, Android tests/lint and macOS Swift tests. Source-level checks never replace native installed-package and hardware acceptance.
- Never label plain FTP as encrypted, macOS FTPS/SFTP as implemented, or 75 image references as pixel-perfect 1:1. Signing/notarization, full backup/restore failure injection, large files and power-loss recovery are not certified by this audit.

**Status:** code review and automated regression extension have been committed to PR #260. Final all-platform approval and publication depend on latest-head CI and manual/device evidence; no universal security or functional guarantee is asserted.
