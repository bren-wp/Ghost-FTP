# Signed cloud URL threat model — macOS and Android

**Feature scope:** Ghost FTP 0.93.0 signed-object foundation with Android 0.94.0 transport hardening. The signed URL is a **bearer secret**, not a general API session. Local file selection and cloud endpoint routing cross separate trust boundaries.

## Trust boundaries and threats

| Boundary | Threat | Present safeguards | Residual risk / action |
| --- | --- | --- | --- |
| Text input → URL parser | Arbitrary host with fake signature marker | Known provider-host families, HTTPS-only, no userinfo/fragments, limit 8192 UTF-8 bytes | Host spelling/IDNA ambiguity and other URL edge cases require fuzz testing |
| Query → provider authorization | Empty, duplicate or different vendor signature | Exact nonempty single X-Amz-Signature for accepted S3 hosts, X-Goog-Signature for GCS, sig for Azure | Query-shape checking cannot cryptographically validate signatures: provider must reject forgery |
| DNS → socket | Split DNS, rebinding, proxy, private metadata target | Android 0.94.0 uses OkHttp DNS injection to validate every address supplied to the actual connecting transport; forces direct (no proxy), disables redirects/retries, retains default TLS verification; macOS retains provider host allowlist only | **P0 macOS** peer-bound routing unresolved; **Android** controlled rebinding/VPN/real-provider E2E and URL-normalization acceptance still required |
| TLS → remote service | Hostname spoofing, certificate substitution | Default platform TLS certificate/hostname validation | P0 certificate failure and intercepting proxy live tests |
| HTTP response → local object | Redirect/token leak, partial 206, compressed body, oversized content | Redirect disabled, full GET 200 required, identity response encoding, 1 GiB ceiling | Test custom content-encoding, unusual content-length and network truncation |
| App RAM → log/backup | Bearer URL persisted or emitted | Masked inputs, transient state, limited high-level status strings | Examine network-library diagnostic paths, crash records and Activity recreation |
| Local document → network upload | Selected file exfiltrated through attacker-chosen link | SAF/file panel selection, provider allowlist, operation-limited signed URLs | P0 host/socket assurance and explicit destination disclosure; links are privileged capabilities |
| Download stage → destination | Partial target, symlink race, disk full or cancellation | macOS same-dir staging/promotion; Android private-cache stage then SAF output | **P0:** Android SAF output may be non-atomic; no guaranteed rollback after final copy |

## Implementation pointers

- Android: `android/app/src/main/java/com/ghostftp/android/SignedCloudTransfer.kt` and `SignedCloudLinkTest.kt`.
- macOS: `macos/Sources/GhostFTPMacApp/Services/SignedCloudTransfer.swift` and `macos/Tests/GhostFTPMacAppTests/SignedCloudLinkTests.swift`.
- Desktop authenticated cloud: `ghostftp-desktop/src-tauri/src/session/object.rs` and `webdav.rs`.

## P0 acceptance before treating arbitrary provider-signed endpoints as secure

1. Test the actual connected peer, not merely a previous DNS lookup; reject private, loopback, link-local, multicast and metadata routes after hostname resolution, including proxy and DNS-rebinding cases.
2. Forbid silent redirect and downgrade; confirm TLS verification and expected signing-host rules across providers.
3. Prove failed/canceled downloads do not replace valid user content; define a specific recovery strategy for SAF document outputs.
4. Validate content byte count and checksum or ETag where meaningful; distinguish provider ETag semantics from cryptographic SHA-256.
5. Make cancellation promptly interrupt active network I/O, clean stages and suppress stale UI updates after lifecycle events.
6. Run real-provider disposable-account tests and preserve redacted evidence with exact commit SHA.

Companion documents: [cloud security overview](../handbook/SECURITY_PRIVACY.md), [provider support](../cloud/PROVIDERS.md), [cloud E2E QA](../qa/CLOUD_E2E.md).

## Android 0.94.0 implementation delta

- `CloudNetworkSafety.kt`: the `PublicCloudDns` resolver blocks reserved/CGNAT/local IPv4, non-global IPv6 and mixed answers in the HTTP client's connecting DNS path. `CloudHttpClient` forces direct networking, TLS verification, no redirects and no automatic replay.
- `CloudTransferCancellation` cancels the underlying HTTP call. The Android UI exposes Cancel and invalidates stale callback results.
- `MainActivity.kt` attempts to delete a new SAF document after failed/canceled downloads; if the provider rejects deletion, the app explicitly warns that an incomplete file can remain.
- This is **not** a certified fix for custom VPN routing, all IDNA cases, provider URL canonicalization, macOS routing or complete atomicity on every SAF document provider. Keep #126 and #127 open until controlled device E2E closes those risks.
