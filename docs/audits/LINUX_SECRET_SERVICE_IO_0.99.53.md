# Ghost FTP — Linux Secret Service process I/O hardening (0.99.53 Preview)

**Owner:** BRENDIGO — https://brendigo.com  
**Scope:** Phases 1–3: incremental dead-code review, credential helper security, reliability.

## Source findings and actual changes

- Before this change, `store()` wrote the secret synchronously before starting the child timeout supervisor; a locked or stalled service could block stdin writes indefinitely when an OS pipe filled.
- Before this change, `wait_bounded()` waited for child exit before draining piped stdout/stderr; a sufficiently large response could block the helper on an OS pipe.
- The input writer now runs concurrently with the child deadline; an unresponsive helper is terminated and reaped before the scoped writer is joined.
- Both output pipes drain concurrently with process execution. Stdout is capped at `MAX_SECRET_BYTES + 1` for oversize detection; stderr at 4,097 bytes. Secret and stderr content never appear in error messages.
- Credentials still travel only via private stdin, never argv/env. Existing read-back verification, legacy kernel keyutils migration and vault failure behavior are preserved.
- No production modules were removed as part of phase 1: the audit found no deletion sufficiently proven within this scoped change. Dynamic IPC, OS callbacks and migration functions cannot be treated as dead by a text-only search.

## Added regression tests

1. An unresponsive child which never reads more than the pipe capacity cannot disable the deadline.
2. A successful child consumes input and terminates when stdin closes.
3. A large response is drained and bounded, rather than deadlocking the child.
4. Existing timeout, invalid secret and byte-exact response tests remain.

**Release gate:** successful exact-commit Rust workspace tests, Clippy, security/protocol checks and Windows/Linux native builds. Source inspection alone does not prove runtime correctness.

## Open findings

- Review `secret-tool` PATH resolution and executable provenance without breaking package-manager paths.
- Test live GNOME Keyring/KWallet locked-vault semantics, restart/reboot, concurrent migration and failed writes.
- Assess helper descendant processes inheriting pipes: the supervisor terminates only the direct process.
- Independent pentest, cloud-provider/OAuth live E2E, signed stable installers and all 59 pixel parity checks remain open.

**Classification:** Preview; not certified stable.
