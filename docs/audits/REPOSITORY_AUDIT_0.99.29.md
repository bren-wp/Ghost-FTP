# Ghost FTP 0.99.29 — Windows/Linux security and dead-code audit

**Scope:** shared desktop React/TypeScript and Tauri/Rust, Windows Cloud Filter feature, Linux Secret Service, local filesystem, transfers, release/CI. Source-review evidence, not a penetration test or complete device acceptance.

## Findings and corrections

1. **High — VirtualFs journal information loss.** `VirtualFs::load` previously converted any read/parse error into `Settings::default()`. Existing Windows sync-root registrations could become untracked, breaking orphan cleanup. Revised to accept empty settings only when the file does not exist, return explicit errors otherwise. A focused Rust test covers missing, malformed and valid journals.
2. **Residual — journaling and OS operations.** Registration and unregister operations involve Windows Cloud Filter APIs; OS-specific race, power-loss and locked-file tests are still required. Do not infer successful removal merely from absence in a frontend list.
3. **Residual — Linux keyring operations.** The 0.99.28 helper timeouts and keyring migration must be verified against a locked GNOME Keyring/KWallet session on physical Linux installations, including missing libsecret-tools.
4. **Dead-code methodology.** Run `node tools/audits/full-dead-code-audit.mjs` and `node --test tools/audits/full-dead-code-audit.test.mjs`, plus Rust/TS build and lints. Textual references alone do not prove a callback, OS entrypoint, feature-gated function or manifest class is dead; remove only verified unreachable code. Preserve OS registration data and historical release provenance.
5. **Security methodology.** Run `node tools/audits/full-security-code-audit.mjs`, `node --test tools/audits/full-security-code-audit.test.mjs`, plus Windows hardening, dependency and real FTP/FTPS/SFTP E2E checks. This checks dangerous static patterns and required invariants, not cloud credentials or a penetration test.

## Baseline audit evidence (published 0.99.28)

The published v0.99.28 exact-source quality workflow [run #41362556207](https://github.com/bren-wp/Ghost-FTP/actions/runs/41362556207) passed the repository-wide static security and dead-code scanners and npm production dependency audit. Its inventory reports 866 tracked working-tree files, 116 TypeScript/TSX, 84 Rust, 18 Kotlin application sources plus 22 Kotlin tests, 17 Swift sources plus 9 Swift tests, 13 Go and 75 operational scripts. The advisory dead-code scanner reports **one potentially unreferenced operational script**, zero Android Kotlin, Swift and Android resource candidate files, and zero one-mention private symbols in Android, macOS, Rust desktop, TypeScript desktop and Go tooling. This is **not** sufficient evidence to delete the script without checking dynamic use and historical workflows.

The npm production dependency check found zero reported vulnerabilities at the time of that CI run; this does not prove absence of security defects. The new 0.99.29 head has to rerun the same gates and verify the new journal guards.

## Click-by-click acceptance boundary

See [click-through matrix](../qa/CLICK.md) for workspace/action coverage and explicit manual gaps. Source/CI checks do not replace installed GUI interaction on Windows 10/11 or Linux distributions and cannot guarantee pixel-for-pixel equivalence with visual references.

**Status:** Candidate changes require green checks on the PR's latest commit. Stable release and production security certification are not claimed.
