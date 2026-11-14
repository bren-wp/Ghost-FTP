# Ghost FTP — live cloud-provider acceptance

> **Active source:** Ghost FTP 0.99.64 Preview. **Previous canonical release:** v0.99.63. Supported platforms: Windows and Linux.

**Evidence, not certification.** The native client includes source-level cloud adapters, but compiling them and running real localhost FTP/FTPS/SFTP tests does not verify a third-party cloud account. Historical discontinued-platform QA evidence is preserved in [the archived plan](../audits/ARCHIVED_MOBILE_CLOUD_E2E_0.95.md), not presented as active desktop coverage.

## Required provider acceptance

| Area | Required real-provider test |
| --- | --- |
| Authentication | Disposable least-privilege account, token refresh, expiry, revocation and logout |
| Navigation | Pagination, duplicate names, Unicode paths, empty directories and permission errors |
| Transfer | Empty, small and multi-GiB upload/download with trusted input/output SHA-256 |
| Integrity | Correct Content-Length/range, resume prefix and no unintended overwrite |
| Failure | 401/403/404/409/412/429/5xx, network loss, quota and disk-full cleanup |
| Security | TLS, proxy, redirects, DNS rebinding, credential redaction and URL-safe logs |
| Reliability | Concurrent operations, cancellation isolation, retries, crash recovery and staging cleanup |
| Native UX | Actual Windows/Linux keyring, dialogs, responsive error states and accessibility |

Capture provider/region, permissions, operating system, exact commit, job or test report, file sizes/hashes, timestamp and redacted result. Secrets must remain in private CI storage; do not publish live tokens or signed URLs. Cloud adapters remain **unverified** until all applicable cases have recorded live evidence. Supported operations may vary by backend.

See [provider source inventory](../cloud/PROVIDERS.md), [protocol QA](PROTOCOL_E2E.md) and [Stable requirements](../release/STABLE_RELEASE_POLICY.md).
