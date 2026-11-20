# Ghost FTP 0.99.71 — Windows source security, stability and dead-code audit

**Date:** 20 November 2026. **Scope:** Existing Windows 10/11 Rust/Tauri desktop, React/TypeScript frontend, native helper launch paths, deep-link ingress, SSH terminal and shared release/test automation. **Status:** Scoped code review and automated regression gates added; full native acceptance requires independent evidence.

## Verified code-level findings and mitigations

| Surface | Finding before changes | Fix in this branch | Residual risk |
| --- | --- | --- | --- |
| External web links | `Command::new("rundll32.exe")` searches for a helper by name, potentially allowing a current-directory/PATH substitution in a portable install | Direct `ShellExecuteW`, initialized COM apartment, blocking worker; existing exact HTTPS allowlist stays | OS browser association and underlying COM shell extensions remain outside application control; test on Windows |
| Edit-in-place open | Same unqualified helper path | Shared `open_with_shell` with the document's native Windows file association | The chosen file association may itself be insecure; opening a user-selected remote file still requires care |
| Protocol handler | Arbitrarily long URL and count of query entries were accepted and decoded before per-value bounds | Reject over 16 KiB, over 64 parameters, decoded C0/C1 controls, URL userinfo/port/fragment | Accepted values still require user review and downstream profile/issuer validation |
| SSH terminal | Per-packet lossy UTF-8 conversion corrupts split multibyte codepoints | Stateful bounded decoder for stdout/stderr, final flush, regression tests and fair event loop | Real PTY shutdown, terminal emulator escape handling and throughput need physical acceptance |

## Full dead-code audit — methodology and limitations

The existing `tools/audits/full-dead-code-audit.mjs` recursively inventories checked-in TypeScript, Rust, Go and operational scripts, skips symlinks/generated outputs, reports potential unreachable or singly referenced nonpublic functions and requires key entrypoints. `ghostftp-desktop/scripts/check-source-reachability.mjs` follows TypeScript imports, Rust module edges and release script references; the quality gate also runs TypeScript and Rust/Go compiler checks. Candidate-only symbols are deliberately **not deleted** without proof for runtime callback, reflection, conditional compilation and compatibility paths. No claims of zero dead code can be made solely from textual search.

## Full security audit — methodology and limitations

`tools/audits/full-security-code-audit.mjs` inventories active source, checks obvious TLS/SSH bypasses and private-key/log patterns, and has required security invariants for Tauri capabilities/CSP, signed-updater configuration, FTP/SFTP staging, path safety, secrets, bridge and release tooling. The new `inspectWindowsShellSecurity` adds negative regression guards for Windows shell launch, `ghostftp://` bounds, UTF-8 terminal integrity and worker isolation. The npm dev/runtime advisory checks, Rust dependency validations, GitHub Actions and real protocol E2E gates are separate and mandatory.

**Limitations:** These are static guards and automated CI tests, not complete penetration tests, cryptographic integrity proofs or physical-OS verification. Credentials in the Windows Credential Manager, signed package upgrades, actual first-frame appearance and 1/4/8 parallel transfers at >4 GiB under network faults remain pending. The published Preview channel is not a Stable security certification.

## Required platform validation

Follow [the native Windows acceptance matrix](../qa/WINDOWS_SHELL_UTF8_ACCEPTANCE_0.99.71.md). Capture actual process tree, no unwanted helper executable, terminal Unicode output, memory/CPU, Windows file associations, screen states, crash diagnostics and regression evidence. Retain GitHub Actions audit JSON artifacts for each exact release SHA. Do not delete historical binaries, rewrite published tags, or silence a mandatory quality gate.
