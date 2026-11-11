# Ghost FTP cloud storage security — Windows and Linux

**Active product:** Windows/Linux Rust/Tauri graphical client, Ghost FTP 0.99.47 Preview. Historical platform-specific functionality documented in earlier releases is not part of current distribution.

| Platform | Cloud connections | Required validation |
| --- | --- | --- |
| Windows | S3-compatible, GCS, Azure Blob, WebDAV, Google Drive, OneDrive, Dropbox, Box adapters | OAuth consent, refresh, key vault and real provider E2E |
| Linux | Same Rust desktop adapters | Secret Service persistence, consent and live provider E2E |

## Authorized account connections

In **Sites → Connect Cloud**, account-based providers use the browser's OAuth 2.0 authorization-code flow with PKCE S256, localhost callback and randomized anti-CSRF state. The provider receives the password directly; Ghost FTP must never request account credentials or persist OAuth client secrets in saved profiles.

Real public native desktop client IDs must be registered with providers. Without a configured ID, an authorized user may configure an appropriate public client ID and its redirect URL. Tokens belong in the operating-system credential service, not in SQLite or logs. Connection metadata alone is saved in the site record. Grant revocation, vault-lock behavior and upgrade migration require actual-device verification.

Google Drive and Google Cloud Storage are distinct services. S3-compatible services, Backblaze B2, GCS, Azure Blob and WebDAV/Nextcloud use their own endpoint and credentials; an arbitrary Drive share page is not a signed object grant.

## Transport and file security requirements

- FTP carries data in plaintext; prefer explicit FTPS with valid certificate/hostname verification or SFTP with a verified SSH host key.
- Never log bearer tokens, private keys, authorization headers or signed URLs, including query strings, on failure paths.
- Reject unsafe redirects and private/loopback destinations for network-grant and signed cloud transfer functionality; server peer validation must occur on the actual connection, not a disconnected preflight.
- WebDAV Basic/Bearer authentication requires valid HTTPS, no URL userinfo/query/fragment and no automatic 30x credential forwarding.
- Limit remote inventory sizes, response payloads and transfer resources. Downloads must not mark success on truncation or replace protected symlinks/special nodes.
- The signed in-app updater requires an authenticated manifest and signatures; a SHA-256 checksum alone is not an OS-publisher signature.
- Never invent connected provider states, successful transfers or screenshots for marketing.

## Outstanding production evidence

Execute credentialed E2E tests with authorized provider accounts, token expiry/revocation, invalid certificate/SSH host-key tests, transfer cancellation, multi-GiB integrity tests and target-Windows/Linux credential-vault restore/rollback. Static scans and offline mocks are **not** sufficient for production security certification.

See [Cloud provider matrix](../cloud/PROVIDERS.md), [cloud E2E](../qa/CLOUD_E2E.md), and [security/privacy handbook](../handbook/SECURITY_PRIVACY.md).
