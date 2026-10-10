# Cloud Storage end-to-end quality plan

**Candidate:** Ghost FTP 0.93.0. This is an acceptance specification, **not a report of completed provider testing**. An automated unit-test result or mock HTTP endpoint must never be reported as a real third-party E2E result.

## Existing automated source-level evidence

| Test | Platform | Source | Verification boundary |
| --- | --- | --- | --- |
| Signed host and provider signature parsing | Android | `android/app/src/test/java/com/ghostftp/android/SignedCloudLinkTest.kt` | URL constructor behavior; no DNS or live provider access |
| Signed URL parsing and unsafe endpoint rejection | macOS | `macos/Tests/GhostFTPMacAppTests/SignedCloudLinkTests.swift` | Foundation URL parser behavior; no authenticated provider access |
| Android package and emulator checks | Android | `.github/workflows/ghostftp-android.yml` | Unit tests, lint, APK/AAB compilation and emulator UI, not live S3 acceptance |
| Protocol round trip | Desktop protocol engine | `.github/workflows/ghostftp-protocol-e2e.yml` | FTP/FTPS/SFTP; **not cloud provider E2E** |

Each must be re-evaluated on the **exact latest PR head SHA**. Listing a test file here does not assert it passed in the current run.

## Provider acceptance cases

| ID | Case | Expected result | Scope | Status |
| --- | --- | --- | --- | --- |
| CL-001 | Valid signed GET for exact object | Bytes match trusted SHA-256 | S3, R2, B2, GCS, Azure | NOT RUN with provider |
| CL-002 | Valid signed PUT, zero byte and Unicode names | Correct object at intended key | Same | NOT RUN |
| CL-003 | Expired or wrong-method signature | No upload/download; safe status | Same | NOT RUN |
| CL-004 | URL with wrong/duplicate/empty signature key | Rejected before local upload file opens | Android/macOS | SOURCE TEST ADDED; CI VERIFY |
| CL-005 | HTTP redirect with signed capability | Redirect refused; URL not leaked | Android/macOS | LIVE TEST PENDING |
| CL-006 | DNS rebinding / private IPv4 / IPv6 / DNS-over-proxy | Unsafe addresses never reach the HTTP client's connect path | Android/macOS | ANDROID guard + unit tests ADDED; DEVICE E2E PENDING; macOS BLOCKED |
| CL-007 | Response gzip / encoded content | Refused; no corrupted promotion | Android/macOS | SOURCE GUARD ADDED; CI VERIFY |
| CL-008 | Connection interrupted midway | Original good target survives if atomic | macOS; Android SAF may not support | PARTIAL, manual QA pending |
| CL-009 | Provider 401/403/404/429/5xx and retry limits | User-safe errors, no credential logs | All | NOT RUN |
| CL-010 | Restart/rotation/cancellation | No leaked task, stale URL, corrupt output | Android/macOS | ANDROID cancellation source ADDED; physical-device test NOT RUN |
| CL-011 | Token invalidation/disconnect | Revoked and locally erased | Authenticated cloud | BLOCKED mobile OAuth |
| CL-012 | Folder recursive transfer and conflicts | Safe, idempotent operation | Vendor APIs where offered | BLOCKED mobile workflows |

## Environment and proof collection

Use disposable least-privileged accounts and a private test bucket/container, with quotas, TTL and cleanup. Supply credentials only as scoped CI secrets. Record **vendor**, **API**, **region**, **scope**, **platform/device/OS**, **commit SHA**, **CI job URL**, **time**, **input size/hash**, **result size/hash**, and the failure mode. Redact full signed URLs (including the query) and token-bearing HTTP headers from all logs. Keep failed uploads for cleanup review when the provider reports uncertain completion.

Android package readiness requires inspecting unsigned AAB (`bundleRelease`) separately from the independently installable preview APK; an unsigned AAB is not a Play-ready signed production package.

See [provider compatibility](../cloud/PROVIDERS.md) and [production blockers](../status/PRODUCTION_READINESS.md).

### Additional Android source tests (0.94.0)

- `android/app/src/test/java/com/ghostftp/android/CloudNetworkSafetyTest.kt` covers public/private/CGNAT/IPv6/reserved DNS answers, blocked mixed records, missing DNS, direct-only proxy policy, redirect/retry restrictions and cancellation idempotence.
- Source tests do not prove live TLS routing, Android proxy/VPN behavior or provider-signed PUT/GET compatibility. The Android CI result for the exact candidate SHA is the authoritative compilation/unit result.
- The save-path behavior is deliberately **best-effort**: `ACTION_CREATE_DOCUMENT` creates the SAF target; the app attempts to delete it after any failed download, but informs the user when deletion is denied. A successful `OutputStream.close()` cannot prove a cloud object's cryptographic identity.

## 0.95.0 Android provider checksum verification — Issue #127 (partial)

When a signed GET returns the official `x-amz-checksum-sha256` response header, Android now computes SHA-256 while streaming to its private staging file, compares the base64-decoded 32-byte digest in constant time and **rejects an invalid/mismatched checksum before it opens the user-selected SAF output URI**. `CloudChecksumTest.kt` covers correct, missing, malformed, truncated and mismatched headers. A provider that omits the header cannot be claimed cryptographically verified, and multipart S3 `ETag` must **never** be interpreted as a SHA-256 object checksum. Check the exact-sha Android CI before considering this tested.

This closes neither the SAF provider-atomicity nor real-device part of [#127](https://github.com/bren-wp/Ghost-FTP/issues/127). The app still relies on provider-dependent best-effort deletion after incomplete final copy. Exercise revoked permissions, disk-full, device rotation, unexpected Content-Length and 0/1GiB objects on genuine providers before any 1.0 production readiness claim.


## macOS provider checksum and hostname policy (0.95.0 candidate, Issue #126)

Mac signed-object downloads now stream-verify S3's **optional** `x-amz-checksum-sha256` on the temporary file **before replacing any existing destination**. A malformed or mismatching header throws a generic error that does not leak the signed URL. An absent checksum does not authorize any claim of object authenticity. `SignedCloudLinkTests.swift` covers valid, absent, truncated and mismatched digests.

Additional macOS hardening rejects invalid/IDNA-punycode/underscore/trailing-dot/empty-label provider hostnames and disables automatic HTTP/HTTPS/SOCKS/proxy discovery on the signed URLSession configuration. These are layered safeguards, **not a transport-bound IP-pin**: URLSession can still independently resolve the hostname during connection. #126 remains **open/P0** until public-IP-only peer selection is guaranteed on the real macOS connect path, and split-horizon DNS, IPv6, VPN/PAC and real provider tests pass.
