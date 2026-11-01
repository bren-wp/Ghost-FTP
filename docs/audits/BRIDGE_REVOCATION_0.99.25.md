# Agent Bridge authorization review — v0.99.25 development

## Observed issue in v0.99.24

The native client still auto-started a previously enabled Agent Bridge and persisted allowed profiles and the approval policy, including `allow_all`. The client-only Settings UI no longer displayed the controls needed to stop and revoke those grants. An approval dialog alone does not restore those controls when the request is automatically approved.

## Proposed remediation

A dedicated Settings → Security card exposes the bridge master switch, existing live session access, and approval policy. A separate confirm-to-revoke-all command disables the listener, clears runtime and persistent grants (including disconnected profiles), and resets automatic approval. The command checks the persisted config before reporting success. Connections, credentials, saved commands and Skills are preserved.

## Tests and limitations

A Rust test exercises in-memory state transitions and a JavaScript source-level regression check ensures the UI/IPC/backend contract remains wired. These are not penetration tests. Device-specific checks, previously started in-flight request cancellation and full TLS/SSH/transfer security acceptance are not established by the source-level checks alone. Do not merge or publish before exact-head CI succeeds. The original v0.99.24 release remains immutable.
