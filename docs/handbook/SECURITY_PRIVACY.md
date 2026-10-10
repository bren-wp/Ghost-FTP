# Ghost FTP — cross-platform security, privacy and trust model

**Assessment scope:** inspected source and repository CI for the 0.93.0 development candidate. **Not** an external penetration test, data protection certification, guaranteed bug-free assessment or proof that every cloud provider API works on every platform. This document separates existing safeguards from gaps and acceptance criteria.

## 1. Security objectives and assets

### Primary assets

- User files and directories at rest, in staging areas and during transfer; include confidential business records, filenames, file metadata and modified timestamps.
- FTP/SFTP passwords, SSH private keys/passphrases, S3/Azure access keys, GCS service-account material, cloud OAuth access/refresh tokens, short-lived signed object URLs and any device/session credentials.
- Connection identity: protocol, normalized endpoint, port, account, TLS certificate, SSH host key, bucket/drive identity and access scope.
- Trusted configuration: saved profiles, UI settings, connection backups, synchronization plans, update metadata, manifests and app binaries.
- User intent: which account, target object/folder, operation, overwrite/delete policy, and whether a remote agent may execute a command.

### Threat actors and boundaries

| Threat | Trust boundary | Consequence | Control and next audit |
| --- | --- | --- | --- |
| Untrusted FTP/SSH server | Network protocol parser and remote filesystem response | Hostile names, size metadata, spoofed errors and data injection | Validate reply size/encoding/path; fuzz and property-test parsers; no shell interpolation |
| Malicious cloud URL | User-controlled signed URL, network destination and DNS | SSRF, data exfiltration, cross-host redirect of token | HTTPS/no userinfo/redirect; harden DNS rebinding and provider allowlist |
| Untrusted cloud provider response | HTTP API JSON/XML, redirects, pagination, response body | Wrong object, partial listing, memory exhaustion, token leakage | Bound parsing/streaming/paging, fail on unverified response, redact all errors |
| Local unprivileged process | OS profile directory, clipboard, temporary files, keychain | Credential theft and unintended access | Per-user file permissions, vault auth, no raw secret copies |
| Backup import | JSON/other format from potentially hostile external file | Credential re-binding, arbitrary path overwrite, DoS | Atomic validated import, record/size cap, UUID/identity collision rejection |
| Sync plan | Local/remote filesystem and asynchronous changes | Mass deletion and partial replacement | Preview, confirmations, dry-run, protected root, safe commit and recovery |
| Remote agent | Network pairing and tool/command invocation | Unapproved destructive command | Authenticated identity, pairing, per-command approval, scoped policy, audit metadata |
| Update pipeline | Artifacts/tags/actions secrets and downloaded updater | Supply-chain attack and wrong executable | Immutable tags, signed manifests, SHA-256, exact-source builds and asset verification |
| Unreliable network | File data channel, retries, timeout, disconnect | Partial/corrupt data, duplicate commits | Temporary stage, bounded retry, verify remote committed offsets and complete response |
| UI confusion | Similar providers, multiple sessions and duplicated profile names | Wrong account/bucket deletion, credential leak | Provider+account labels, scope confirmation and explicit destructive actions |

## 2. What is implemented today vs requiring verification

**Source-implemented safeguards:** Tauri CSP, protocol-specific TLS/SSH checks, local credentials/key references on desktop, macOS device Keychain, Android session-memory password and redacted error helper, import restore guards, no mandatory analytics, checked private/HTTP cloud endpoint patterns, Android FLAG_SECURE + disabled cleartext and OS backups, and signed-object links without persistent storage on macOS/Android.

**Not fully verified / needs additional work:** DNS rebinding resistance and fixed-IP binding for untrusted signed URLs, comprehensive cloud OAuth refresh rotation/revocation on every provider, complete secrets-on-disk inspection, malicious cloud JSON/remote filename fuzzing, physical device accessibility, complete macOS FTPS/SFTP, Android true atomic SAF target replacement, supply-chain dependency SBOM/signature attestations and privacy review across every bridge command.

Do not interpret "some hardening exists" as "all paths are safe."

## 3. Transport security specification

### FTP

FTP credentials and file content are transmitted without encryption. The application must display this warning prominently; do not call it private or secure. Prefer SFTP/FTPS for secrets. FTP path arguments must reject CR/LF injection and forbidden remote names. Handle ABOR/225 response order carefully: do not trust a local attempted upload offset as the remote committed prefix; connect fresh to verify before resuming.

### Explicit FTPS

Require AUTH TLS, verify full certificate chain, validity and hostname with OS roots, and protect data channels as appropriate (PBSZ/PROT). Never offer a silent downgrade to plain FTP. Test expired, self-signed, hostname-mismatched and replaced certificates. Native macOS FTPS is currently not a file-operation capability.

### SFTP

Use SSH host-key verification, reject mismatches before exchanging sensitive credentials, handle authorized trust prompts and known-host rotation explicitly. Private keys remain in protected storage or known user-controlled paths; passphrase handling must not enter logs. Native macOS SFTP is currently not a file-operation capability.

### HTTP/WebDAV

Require HTTPS for credential-bearing WebDAV with strict URL parsing, path/root normalization and no embedded credentials/query/fragment where configuration uses a bare endpoint. Verify that Authorization is not forwarded to redirected hosts; prefer no redirects for authenticated requests. Bound PROPFIND XML size/depth and prevent path traversal via decoded hrefs.

### Object storage

- S3 SigV4: secure endpoint selection, bucket and region matching, signing scopes, clock skew, path vs virtual-host addressing and session token support where available.
- Azure: account/container identity, SAS vs account key and operation-specific access, no secret in endpoint URL or diagnostics.
- GCS: distinguish a Google Cloud Storage service account from OAuth Google Drive. Secure private JSON key handling, rotation and least privilege.
- Object store rename is often copy+delete, not atomic. Require explicit indication of eventual consistency and partial failure.

## 4. Signed-object URL security requirements

Temporary signed URLs allow an operation without holding long-lived account credentials. This helps privacy but is **not equivalent to account integration**.

1. User generates an expiring, object-specific signed GET or PUT URL in a trusted provider. Links are bearer secrets, not harmless web links.
2. UI masks text and removes it on completion/cancel; no persistence in site profiles, backups, activity history or logs.
3. HTTPS, no URL userinfo, no fragments, no ordinary HTTP redirects, certificate/hostname validation.
4. Reject private/local endpoints and cloud metadata addresses. Android currently pre-checks DNS; macOS currently validates syntax/hostname shape. **P0 gap:** both need robust DNS rebinding defenses, IPv4/IPv6/IDNA canonicalization and potential HTTPS proxy policy.
5. Limit per-object input/output, total bytes, requests, timeouts, redirects and unexpected content. Current initial signed-transfer cap is 1 GiB.
6. Verify HTTP status after streaming; a 200 response is not a guarantee of the expected object ID/version/hash. Prefer provider ETag/checksum and expected key/size when available.
7. Avoid leaking URL in error/exception text, exception telemetry, clipboard, HTTP logs, stack traces or UI accessibility announcements.
8. Reject signing method mismatch and signed-header mismatch; document provider-specific Content-Type and Azure x-ms-blob-type requirements.
9. **Android known issue:** SAF cannot necessarily atomically replace destination. Cache staging reduces exposure but a canceled final copy can leave a partial document. Finalization needs recovery UX and integrity checks.
10. **macOS known issue:** ad hoc signed URL host checking alone is insufficient for DNS rebinding. Add secure host/IP binding or a trusted provider allowlist before sensitive production use.

## 5. Cloud OAuth, account and key management design

No third-party OAuth client secret should be embedded in a mobile or desktop app. Use **Authorization Code + PKCE** through the user's system browser and a verified redirect/custom URI scheme, not a shared static password. Bind state/nonce, validate issuer/audience where applicable, and protect local refresh tokens with OS facilities (Windows Credential Manager, suitable Linux secret backend, macOS Keychain, Android Keystore-backed encryption). Scope each connection to files/buckets/objects required; Drive drive.file rather than unrestricted Drive scope where functionality permits. Support account list, token expiry, rotation, 401/403 distinction, sign-out/revoke and device-only deletion. Never make an untrusted proxy a mandatory token receiver merely to simplify implementation.

S3 static keys should support rotation, least-privilege bucket policy, session credentials where possible, separate read/write scopes and consistent multi-account identity. Do not ship real AWS/GCP/Azure credentials, OAuth tokens or signed links as test fixtures.

## 6. Platform credential and data-retention matrix

| Data | Windows | Linux | macOS | Android |
| --- | --- | --- | --- | --- |
| FTP/SFTP passwords | OS credential integration/key refs where supported | Native keyring integration, availability varies | Keychain optional remember | Volatile Activity session, cleared on destroy/disconnect |
| Cloud long-lived keys | Desktop adapter plus OS secret integration subject to audit | Same Rust client; validate keyutils/keyring session | No full credential-based account browser in current Preview | No full credential-based account browser |
| Signed object URL | Existing account adapters; should never log bearer URL | Same desktop requirements | Masked transient input and ephemeral URLSession | Masked transient input and transient transfer thread |
| Connection settings | Per-user local profile files | Local profile with restrictive permissions | ProfileStore non-secret JSON with recovery | State/backup excludes password |
| File data | User selected paths + temporary staging | User selected paths + temporary staging | File picker paths + URLSession staging | SAF-selected URI + cache staging |
| Transfer/activity history | Local metadata; configurable controls | Local metadata | Bounded FTP history; clear action | Bounded visible rows |
| Cleanup/revocation | Explicit sign-out + vault clear recommended | Session vault purge recommended | Keychain deletion checked before site removal | Clear pending URI/link/session secrets on Activity destruction |

Never claim data is end-to-end encrypted merely because TLS is used; providers receive plaintext unless user-side file encryption is added. A cloud service can see objects and their metadata. Connection via S3/Drive is not a Ghost FTP hosted cloud service.

## 7. Secure file writes and destructive operations

Files must be staged with unpredictable names and user-only permissions. On upload, the existing target should remain untouched until verified content is ready for promotion, with a recovery route on failures. On download, do not overwrite previously good content until response body is completely and correctly received. Respect symlinks, case collisions, path separators and Unicode normalization. For user-visible rename/delete/mirror, show exact destination/account and impact before committing; require a second confirmation for root/mass operations, and record only non-sensitive audit metadata.

Synchronization is not inherently atomic. A provider copy+delete operation can partially succeed; retry must be idempotent and explicitly reconcile object versions, conditional headers or ETags. Unit tests must exercise failures between every step.

## 8. Secure updates and supply chain

- Pin and review third-party dependencies, GitHub Actions and lockfiles. Run SCA and license/security scans.
- Build release packages from one exact main commit; require six quality/native/Android/macOS/Windows-hardening/protocol gates before release.
- Never move an existing public tag to another SHA. Validate semantic release train and previous public release from GitHub API.
- Verify actual NSIS/MSI/AppImage/DEB/RPM/APK/AAB artifacts, their sizes and SHA-256 manifest. Do not equate uploaded artifact metadata to installer launch success.
- Updater metadata should be signed and bound to exact release version; do not install unsigned updates.
- Produce reproducibility/SBOM and provenance review; do not upload private signing credentials to PR artifacts.

## 9. Security testing methodology

| Class | Minimum cases | Expected result |
| --- | --- | --- |
| URL fuzz | HTTP, userinfo, fragment, missing/malformed signature, IPv4 hex/octal, IPv6, IDNA, encoded slash/query, DNS rebinding | No untrusted internal network access, no credentials in logs |
| TLS | Expired/wrong-host/self-signed/intercepted certificates, redirect, protocol downgrade | Secure error, no retry with verification disabled |
| Secrets | UI logs, Android Bundle, macOS Keychain error, exports, temporary file, crash trace | No plaintext secrets or bearer URL |
| File integrity | 0 bytes, 1 GiB, >limit, truncated response, length mismatch, disk full, read-only target | Prior good data preserved; clear and accurate failure |
| OAuth | PKCE/state mismatch, expiry, refresh rotation, revoked token, scope escalation | Abort or user consent; no scope expansion silently |
| Permissions | FTP root, SFTP symlinks, S3 bucket prefix, Drive shared folder, SaaS scope | Correct least-privilege failure rather than accidental delete |
| Dependency | Tauri/Swift/Kotlin/Rust package audit, action pinning | No unreviewed high-severity issue without explicit documented triage |
| Device | Windows/Linux/macOS/Android supported versions; resize/rotation/sleep/network loss | No crash, unauthorized file/secret persistence, or misleading success |
| Restore | Corrupt/oversize backups, UUID collision, unknown fields, malicious path | Atomic rejection, no credential rebind |

## 10. Release blocking criteria

**Block merge or release** for any known plaintext credential logging, invalid TLS bypass, untrusted private endpoint reachability, silent remote overwrite, wrong Keychain identity, broken NSIS/MSI smoke, failed Rust/Android/macOS test or unexpected source-version/tag reuse. Open security issues privately when exploitable; publish only after mitigation and verification. "All green" CI is necessary but not sufficient for unsupported real provider/device matrix.

## 11. Reporting vulnerabilities and data requests

See SECURITY.md for preferred disclosure route, and docs/legal/PRIVACY.md for current policies. Do not paste sensitive tokens, signed URLs, passwords or production customer logs into public issues. Redact diagnostic bundles before submitting. A privacy deletion request must distinguish local user-held data from information retained by the independent cloud provider.
