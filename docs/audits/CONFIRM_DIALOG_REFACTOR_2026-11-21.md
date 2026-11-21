# Ghost FTP by Brendigo — Confirmation reliability and dead-code review

**Scope:** Ghost FTP 0.99.74 Preview source; targeted desktop and shared file-ui confirmation/prompt components. This is a **focused** audit, not certification that the entire maintained repository is free of vulnerabilities or dead code.

## Verified source findings and changes

- Desktop and shared file-ui had separate, near-identical `ConfirmModal` implementations. The desktop implementation is now a thin adapter for the reusable file-ui modal, retaining the desktop toast error behavior; shared file-ui retains its inline error message.
- Confirm/prompt handlers previously checked only `useState(submitting)`, which can permit a second click or Enter event before React commits a re-render. Both now use a synchronous `useRef` single-flight guard. Successful action completion keeps the guard until the dialog closes; failures release it for a safe retry. Dismissal also respects the immediate guard.
- The prompt input previously handled Escape locally while the enclosing accessible-dialog hook also handled the bubbled Escape event. Escape dismissal now belongs exclusively to the dialog hook; Enter prevents native implicit submission.
- The desktop and file-ui now share the same accessible focus-management hook, retaining `trapFocus: false` for the main workspaces and the dialog's Escape/focus restoration contract.
- The unused JavaScript imports `@tauri-apps/plugin-fs` and `@tauri-apps/plugin-shell` had no production source references; both dependencies and their npm lockfile records were removed. Tauri native filesystem and Windows Shell operations are separate and were not removed.
- The repository dead-code analyzer now handles Rust nested comments, raw strings and character literals inside `#[cfg(test)]` modules; tests cover production symbols after such modules.
- Native Agent Bridge tests were extracted to `bridge/tests.rs` without removing authorization/revocation coverage. A malformed Agent file preview now fails closed with a bounded decoded length and a verified byte count, and the daemon detects true truncation by reading at most the allowed size plus one byte.
- The existing interaction-hardening source regression checks were updated to require delegation and the new guards. No user-facing colors, layout rules or text were intentionally changed by the modal refactor.

## Review boundaries and next evidence

- Full repository dead-code tooling exists under `tools/audits/full-dead-code-audit.mjs` and `ghostftp-desktop/scripts/check-source-reachability.mjs`. Their advisory findings must not be converted to deletions without proving entrypoint, cross-platform, dynamic, protocol and CI reachability.
- Removed unused JS dependencies are reflected in `package-lock.json`; `npm ci`, Rust/TypeScript/Go and Windows/Linux build gates must validate this tree. Remaining dependency upgrades require regenerated lockfiles and compatibility checks, not blanket version bumps.
- Verify Tauri/React build, mandatory CI, destructive operations, double-click and Enter behavior, Escape cancellation, failure toast/inline error, focus restoration and screen-reader announcement on **actual** native runtimes. CI runs and native screenshots are pending until externally reported.
- Reference pixel parity and screenshot freshness remain **unverified**. See [current visual acceptance](../qa/CURRENT_VISUAL_ACCEPTANCE.md).
- GitHub issue [#302](https://github.com/bren-wp/Ghost-FTP/issues/302) remains blocked while `main.protected=false`; only repository-admin branch/ruleset configuration and positive/negative merge acceptance can close it.

## Acceptance

No claim is made that all vulnerabilities, all duplicate code or all UI differences have been eliminated. This PR addresses a specific verified duplicate implementation and interaction race; remaining audit scope and external gates stay open.
