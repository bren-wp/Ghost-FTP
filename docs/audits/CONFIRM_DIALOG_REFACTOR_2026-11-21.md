# Ghost FTP by Brendigo — Confirmation reliability and dead-code review

**Scope:** Ghost FTP 0.99.73 Preview source; targeted desktop and shared file-ui confirmation/prompt components. This is a **focused** audit, not certification that the entire 812-item repository tree is free of vulnerabilities or dead code.

## Verified source findings and changes

- Desktop and shared file-ui had separate, near-identical `ConfirmModal` implementations. The desktop implementation is now a thin adapter for the reusable file-ui modal, retaining the desktop toast error behavior; shared file-ui retains its inline error message.
- Confirm/prompt handlers previously checked only `useState(submitting)`, which can permit a second click or Enter event before React commits a re-render. Both now use a synchronous `useRef` single-flight guard, reset on settlement. Dismissal also respects the immediate guard.
- The prompt input previously handled Escape locally while the enclosing accessible-dialog hook also handled the bubbled Escape event. Escape dismissal now belongs exclusively to the dialog hook; Enter prevents native implicit submission.
- The existing interaction-hardening source regression checks were updated to require delegation and the new guards. No user-facing colors, layout rules or text were intentionally changed by the modal refactor.

## Review boundaries and next evidence

- Full repository dead-code tooling exists under `tools/audits/full-dead-code-audit.mjs` and `ghostftp-desktop/scripts/check-source-reachability.mjs`. Their advisory findings must not be converted to deletions without proving entrypoint, cross-platform, dynamic, protocol and CI reachability.
- Dependency changes require regenerated lockfiles and Rust/TypeScript/Go builds on Windows/Linux; do not bump manifests without proving the resolved versions and integration contracts.
- Verify Tauri/React build, mandatory CI, destructive operations, double-click and Enter behavior, Escape cancellation, failure toast/inline error, focus restoration and screen-reader announcement on **actual** native runtimes. CI runs and native screenshots are pending until externally reported.
- Reference pixel parity and screenshot freshness remain **unverified**. See [current visual acceptance](../qa/CURRENT_VISUAL_ACCEPTANCE.md).
- GitHub issue [#302](https://github.com/bren-wp/Ghost-FTP/issues/302) remains blocked while `main.protected=false`; only repository-admin branch/ruleset configuration and positive/negative merge acceptance can close it.

## Acceptance

No claim is made that all vulnerabilities, all duplicate code or all UI differences have been eliminated. This PR addresses a specific verified duplicate implementation and interaction race; remaining audit scope and external gates stay open.
