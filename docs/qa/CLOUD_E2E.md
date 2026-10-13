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

## 0.95.4 Android SAF read-back integrity (reopened #127)

The next candidate re-opens the newly created SAF output document **after** closing its output stream, re-reads all persisted bytes, and checks both length and SHA-256 against the temporary staged object. Extra bytes, truncation, silent same-length mutation and permission/provider read errors fail closed. An unreadable provider document cannot be counted as a successful download. The existing best-effort cleanup attempts to delete the newly created document; if denied, users are warned of possible incomplete output. New `CloudSavedDocumentIntegrityTest.kt` unit regressions cover exact/zero-length, truncated, mutated, extra, cancellation and invalid-checksum paths. Source tests alone are **not** acceptance for third-party SAF providers.

**Open acceptance:** Run on real Android 8–15 with an actual cloud-backed DocumentsProvider and revoked permissions, full disk, rotation, background death, zero-byte and 1 GiB objects. SAF providers can lie about both reads and writes and may not support atomic replacement. These are explicit limitations, not closed security items.

**Additional open items:** #126 requires controlled DNS/proxy/VPN + live provider proof; #128 requires complete mobile/macOS OAuth and provider E2E. The source-only evidence does not close either.

## 0.95.4 IPv6 and canonical DNS routing regression (reopened #126)

The Android `PublicCloudDns` hook, whose return values are used by OkHttp to establish the actual connection, now **also** validates the canonical hostname with the signed-link ASCII/provider policy (defense against future bypassing callers). Android and macOS IP filters both reject IANA's `3fff::/20` IPv6 documentation prefix (RFC 9637), `2001:2::/48` benchmarking and deprecated `2001:10::/28`. Their unit tests cover these cases. All other existing controls stay enabled: HTTPS certificate/hostname trust, no redirects, direct-only networking, and macOS curl `--resolve` peer pinning.

**Unmet acceptance remains**: no controlled proxy/VPN/split-horizon/rebinding lab and no disposable signed public-provider success evidence was provided in this cycle. Keep #126 open until that work is completed.

0.95.4 additional provider-reliability guard: the final readback also limits repeated zero-byte reads from contract-violating SAF providers. Isolated `CloudSavedDocumentIntegrityTest.rejectsNonProgressingDocumentProviderStream` covers this stall; real device/provider acceptance is still outstanding.

## 0.95.6 Android bounded cloud source reads (Issue #127 — partial)

Cloud SAF uploads, cloud GET staging and final SAF writes now share a size-limited and cancellation-aware stream copier. Broken streams that return zero bytes repeatedly are rejected after eight consecutive nonprogress returns, rather than spinning forever. Unexpected extra bytes or early EOF against provider-stated lengths fail closed. `CloudStreamProgressTest.kt` includes JVM tests for valid zero-byte/exact objects, truncation, expansion, zero-progress, recovery, cancellation and output errors.

**Remaining blocker:** A provider that blocks inside a single `read()` call may not respond to cancellation until it unblocks. Emulator/JVM tests cannot verify real Android DocumentsProvider lifecycle and kernel/device I/O behavior, storage exhaustion or large object transfer. Do not close #127 based only on these changes.

## 0.95.6 active SAF stream cancellation (Issue #127 — additional partial mitigation)

The Android cloud cancellation token now tracks live user-provider `InputStream`/`OutputStream` handles for signed upload, final SAF output write and persisted-document readback. Cancellation first aborts the OkHttp request, then attempts to close the provider handles from a dedicated daemon thread rather than blocking the Android UI. A stream registered after cancellation is immediately closed and rejected; closed/deregistered streams are not reused. `CloudProviderCancellationTest.kt` adds three JVM race and lifecycle tests. This is *best-effort* provider I/O abortion, not a guarantee that malicious, defective or disconnected DocumentsProviders unblock their own Binder transactions. Device acceptance of #127 remains open; cancellation still has to be verified against real SAF providers and Android 8–15 lifecycle states.


## 0.95.7 cancellation isolation follow-up (#127; source-level)

`CloudProviderCancellationTest.blockedProviderCloseDoesNotHoldUpAnotherProviderStream` covers two concurrently watched SAF streams, where a simulated provider's `close()` blocks. Cancellation dispatches independent daemon close attempts so the other stream still receives its close request. This JVM regression alone does not prove termination of real provider Binder calls, Android 8–15 lifecycle behavior, revoked permissions, disk-full/1 GiB edge cases or atomic SAF replacement. #127 remains open pending physical-device and real DocumentsProvider acceptance.

## 0.95.8 draft: orphaned app-private cloud stage cleanup (#127 — partial)

A new process can start after Android terminates a transfer without running Kotlin `finally`; this leaves an object body in `cacheDir/ghostftp-cloud-*.part`. The proposed `CloudStageCleanup` removes strictly named regular staging files *once* at app process startup, before any new transfer, and preserves unrelated cache content and SAF documents. The once-per-process guard prevents Activity recreation from deleting a file used by a live transfer. JVM tests cover a real `File.createTempFile` stage, unrelated files, a similarly named directory and absent cache directory.

This does not guarantee cleanup while the process remains dead, on disk-full, after Android revokes storage grants, or with a broken SAF provider. On-device Android 8–15 process-death/device-provider acceptance remains open in #127. This section documents a draft patch, **not** a published or physically validated feature.
