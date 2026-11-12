# Ghost FTP 0.99.51 — targeted cloud credential and retry audit

## Confirmed source-level vulnerabilities and fixes

1. **Bearer destination integrity:** Shopify's `X-Shopify-Access-Token`, HubSpot's `Authorization: Bearer`, and Dynamics bearer tokens were added to requests built from URL strings without an explicit configured-origin check. HubSpot async task status links are returned by the provider and can be absolute. All three adapters now require `trusted_bearer_url` validation before attaching credentials. It rejects foreign origins, insecure downgrades, URL userinfo and fragments. Test fixtures may use explicitly configured loopback HTTP bases.
2. **Redirect forwarding:** Credential-bearing REST clients now reject redirect chains; a 3xx response is handled as an error instead of automatically forwarding tokens/headers. Genuine CDN signed-URL redirect behavior must be validated with authorized providers; avoid claiming all cloud edge cases work.
3. **Non-idempotent replays:** The shared HTTP throttle previously retried any 429/5xx, including writes, without knowing whether a remote mutation had already completed. It now retries only GET/HEAD/OPTIONS. POST, PUT, PATCH and DELETE return the provider's result immediately; callers can display errors and let users decide how to retry.
4. **Shopify token lifetime:** The untrusted `expires_in` input previously fed into an unchecked `Instant` addition. It is now capped at 86,400 seconds. Blank token responses are refused.
5. **Transfer Center load:** `recentTransferLog` previously filtered, cloned, sorted and sliced the whole backend queue. A stable bounded top-k selector holds at most k rows (default 12), avoiding a full log sort and preserving object identity. No measured native CPU/RAM benchmark is claimed.

## Verification

Mandatory exact-source CI includes repository-wide dead-code and security code audit scripts with test matrices and uploaded JSON evidence, Rust `cargo test --workspace --locked` and Clippy, TypeScript checks, Windows filesystem regressions, real localhost FTP/FTPS/SFTP E2E, Windows hardening, and Windows/Linux native package smoke checks. The release itself requires 15 exact SHA-256-checked assets. Any failed gate prevents publication.

## Remaining external/security acceptance

- Provider-authenticated cloud E2E and OAuth/credentials, signed URL behavior, rate-limit handling, renew/revoke and failures on production tenants
- Hostile local IPC/client, multi-GiB transfer recovery, symlink races and installer upgrade/rollback tests on real machines
- Independent penetration testing and code review, signed release binaries and exact pixel comparisons for all 59 Windows/Linux references

Historical macOS outputs are immutable archives only; the active product supports Windows and Linux.
