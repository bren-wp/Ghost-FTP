# Desktop protocol security verification matrix

This document tracks **verification requirements**, not claims that every adapter has passed.

Scope: `ghostftp-desktop/src-tauri/src/{session,remotefs}`, `oauth.rs`, `known_hosts.rs`, `grant.rs`, `transfer.rs`. Windows and Linux must both be exercised.

| Adapter | Required adversarial integration checks |
| --- | --- |
| FTP | Explicit plaintext warning; no silent FTPS downgrade; CR/LF command injection; malicious listings and traversal; concurrent transfers |
| Explicit FTPS | Untrusted/expired/wrong-host/rotated certificates; trust-chain validation; AUTH TLS; PBSZ 0 and PROT P; broken data-channel handshake; no plaintext retry |
| SFTP/SCP | Unknown and changed host keys; hashed known_hosts; file permissions and races; IPv4/IPv6; SSH agent and encrypted private keys; traversal |
| HTTP/HTTPS/WebDAV/Nextcloud | SSRF and DNS rebinding; redirect revalidation; HTTPS downgrade; Basic Auth leakage; response size limits; malformed XML; timeouts |
| S3/B2/GCS/Azure | Signed URL and token redaction; redirect policy; paginated response bounds; 429/5xx bounded backoff; truncated/corrupt responses |
| Google Drive/OneDrive/Dropbox/Box | OAuth refresh races and invalid grants; token redaction; cursor cycles; pagination caps; malformed JSON; throttling |
| Shopify/HubSpot/Dynamics | OAuth and tenant validation; rate limits; error-body redaction; untrusted next links; pagination bounds; malformed JSON |
| Other implemented adapters | Inventory actual adapter code and apply the same network/credential/path threat model |

## Required CI gate

A required `desktop-security` job should fail on any regression in TLS/SSH trust, plaintext downgrade, URL/redirect/SSRF protections, secret redaction, response bounds, traversal or injection. Run isolated service-backed FTP/FTPS/SFTP/HTTP adversarial integration tests, not only unit tests. Publish test evidence and platform matrix for Windows and Linux. Branch protection must require the job; adding a workflow alone does not enforce it.

## Evidence policy

- **Code inspection** establishes only the reviewed implementation behavior, not full runtime security.
- **Unit tests** establish only the cases executed by those tests.
- **Integration verified** requires an isolated server, an executed command, a result, and CI run reference.
- **Manual verification pending** includes real cloud accounts, provider-specific token expiry/rotation, certificate infrastructure, SSH agents, and Windows/Linux runtime compatibility not reproduced in CI.

## Initial source inspection (2026-10-20)

The repository's desktop root is `ghostftp-desktop/src-tauri/src`, not repository-root `src-tauri/src`. Existing `session/ftp.rs` uses distinct plain/TLS stream variants and a native TLS connector; `remotefs/ftp.rs` has a bounded reconnect case for a specific FTPS data-channel error. `known_hosts.rs` delegates OpenSSH known-host matching to russh. Existing regression tests are present for FTP command-path validation and selected FTPS errors. These observations **do not** establish that TLS data-channel validation, SSH host-key handling, or cloud adapter security is complete.

**Current status:** full audit incomplete; no integration test execution or security certification is claimed. Do not merge until the requirements above have test evidence and the required CI gate is configured.
