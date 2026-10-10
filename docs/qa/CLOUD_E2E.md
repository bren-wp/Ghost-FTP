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
| CL-006 | DNS rebinding / private IPv4 / IPv6 / DNS-over-proxy | Connection refused at socket boundary | Android/macOS | BLOCKED: mitigation incomplete |
| CL-007 | Response gzip / encoded content | Refused; no corrupted promotion | Android/macOS | SOURCE GUARD ADDED; CI VERIFY |
| CL-008 | Connection interrupted midway | Original good target survives if atomic | macOS; Android SAF may not support | PARTIAL, manual QA pending |
| CL-009 | Provider 401/403/404/429/5xx and retry limits | User-safe errors, no credential logs | All | NOT RUN |
| CL-010 | Restart/rotation/cancellation | No leaked task, stale URL, corrupt output | Android/macOS | NOT RUN |
| CL-011 | Token invalidation/disconnect | Revoked and locally erased | Authenticated cloud | BLOCKED mobile OAuth |
| CL-012 | Folder recursive transfer and conflicts | Safe, idempotent operation | Vendor APIs where offered | BLOCKED mobile workflows |

## Environment and proof collection

Use disposable least-privileged accounts and a private test bucket/container, with quotas, TTL and cleanup. Supply credentials only as scoped CI secrets. Record **vendor**, **API**, **region**, **scope**, **platform/device/OS**, **commit SHA**, **CI job URL**, **time**, **input size/hash**, **result size/hash**, and the failure mode. Redact full signed URLs (including the query) and token-bearing HTTP headers from all logs. Keep failed uploads for cleanup review when the provider reports uncertain completion.

Android package readiness requires inspecting unsigned AAB (`bundleRelease`) separately from the independently installable preview APK; an unsigned AAB is not a Play-ready signed production package.

See [provider compatibility](../cloud/PROVIDERS.md) and [production blockers](../status/PRODUCTION_READINESS.md).
