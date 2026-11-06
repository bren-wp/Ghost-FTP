# Ghost FTP v0.99.34 — Windows/Linux cross-protocol integrity, security and dead-code audit

**Status:** source inspection and targeted fixes committed; exact-SHA CI must be checked separately. A full static source audit is not a penetration test and cannot prove that every third-party server is compatible or that every UI control is perfect.

## Scope and evidence

- **Desktop UI:** Tauri/WebView2 Windows and Tauri/WebKitGTK Linux; React state stores; file list, upload/download actions, transfer queue, retries, pause/resume, Sync, Settings, profiles, host identity and UI failure handling.
- **Native Rust:** `src-tauri/src/transfer.rs`, 13 `Session` variants (SFTP/SSH, FTP/FTPS, S3-compatible object, WebDAV, HTTP read-only, Dropbox, OneDrive, Google Drive, Box, Shopify, HubSpot, Dynamics and Ghost FTP Agent), native `RemoteFs` safety, profile JSON durability, Credential Manager and Secret Service.
- **Release/packaging:** unsigned Windows portable/NSIS/MSI and Linux DEB/RPM/AppImage, exact-source CI, updater signature guard, immutable version/tag/checksums.
- **Repository-wide dead code:** `tools/audits/full-dead-code-audit.mjs --json` is the automated source/entrypoint inventory; candidates are advisory, not proof of unused exported code, dynamically called commands or public APIs. `npm run check:dead`, cargo check/test/clippy, Android/Swift/Go and CI/build-root references must pass before deletion.
- **Repository-wide security:** `tools/audits/full-security-code-audit.mjs` scans checked-in runtime sources and CI configurations, rejects obvious secret exposure, TLS/SSH bypass and broad Tauri capabilities, and now requires transfer completion and diagnostic-sanitization guards. Exact-head CI archives the JSON audit output.

## Transfer risk and verification matrix

| Transport | Download | Upload | Safety assertions and remaining gaps |
| --- | --- | --- | --- |
| FTP | Native binary RETR; staged local destination | Native STOR, resume, cancel/ABOR | Source SIZE where supported, counted bytes, staging fsync, no false Done; plaintext protocol remains **unencrypted** |
| FTPS (explicit TLS) | Same core FTP stream over TLS | Same core FTP stream over TLS | TLS trust verification and server SIZE when supported; no TLS bypass; real localhost FTPS E2E |
| SFTP / SSH | SFTP read, byte-offset support; staged file | SFTP write, byte-offset support | Source/destination metadata sizes, staging fsync, true transferred length, pinned host-key rejection; changes of same length and prefix identity need stricter verification |
| S3 / compatible object | Stream object GET | Put/multipart, abort on failure | Bounded upload memory; multipart source-length checks; real object-store outage and multipart fault injection still required |
| WebDAV | Streaming GET | PUT | Existing length guards on downloaded HTTP bodies; real WebDAV auth/lock/overwrite server acceptance required |
| HTTP / HTTPS | Read-only, staged download | Unsupported by design | Identity Content-Length validation; no claim of HTTP uploads |
| Dropbox / OneDrive / Google Drive / Box | Provider API/download streams | Provider-specific uploads/session/multipart | API identity, authorization refresh, 429, 5xx, range, quota, >4 GiB and partial commit require live provider fixtures |
| Shopify / HubSpot / Dynamics | Provider media/API read | Provider media/API write (bounded memory where needed) | Tenant scopes, payload quotas, API rate limits, conflicting remote metadata and live account E2E remain unverified |
| Ghost FTP Agent | Ranged reads; delta download when supported | Ranged writes; delta upload when supported | Native delta tests cover reused blocks/fallback; network interruption, daemon change of content and remote integrity require actual daemon integration |

**Design principle:** `Done` must not be emitted for a known-size partial transfer. A verified complete temporary download is promoted only after the transport reports success; existing files are not overwritten merely because a remote call returned HTTP 200. If server metadata is missing, status can only certify the bytes actually received, not that the remote server sent its entire logical object. Local size equality is not cryptographic end-to-end hashing.

## Confirmed defects addressed in 0.99.34

1. **Integrity:** SFTP/FTP/FTPS could return success without reconciling expected, streamed and persisted bytes. Added explicit guards; SFTP checks remote metadata before and after, uploads verify remote length.
2. **Incorrect progress:** a common successful transfer handler forced `transferred = size`, concealing short results and showing zero for unknown-size downloads. It now rejects mismatches and preserves the real byte count.
3. **Privacy:** raw remote-provided failure strings could enter the persistent transfer ledger and later surface in the UI. They are now replaced by safe classified user-facing errors before persistence or event emission.
4. **Regression:** added unit cases for zero-length, 64 KiB boundaries, truncation, oversize, >4 GiB arithmetic and secret-bearing server errors, plus repository security audit assertions.

## Remaining blockers / manual tests

- **SFTP and FTP resume:** current offset/size checks do not cryptographically verify the unchanged remote/local prefix or protect against a different file of equal size. Source replacement, mid-transfer mutation and power-loss recovery must be tested and hardened before claiming completely safe resume.
- **All cloud adapters:** no real third-party account/network test credentials were supplied. Do not represent source-level checks or mock HTTP fixtures as complete upload/download certification.
- **Crash/migration/keyrings:** test encrypted backup and restore, locked/unavailable Linux Secret Service, Windows Credential Manager and large transfer interruption/retry/overwrite with actual servers and target OS sessions.
- **File staging path replacement:** concurrent adversarial filesystem mutation requires dedicated device-level symlink/reparse-point race acceptance. Native no-replace promotion guards are present, but one-time lstat is not a full race-proof guarantee.
- **Pixel-perfect UI:** the design archive contains 30 Windows and 29 Linux concept screens, with only a subset captured by 0.99.33 native CI. Strict RGB differences are measured, not resolved; see visual QA baseline. No fabricated server files/transfers may be added to imitate a concept image.
- **Cross-platform release governance:** macOS FTPS/SFTP is explicitly not implemented and global stable signing/notarization gates are incomplete. Windows/Linux unsigned installation is allowed, but not a claim of universal platform stability.

## Dead-code decision

**No compiler-reachable module, route, protocol adapter or dynamic script was removed based on grep alone.** The full dead-code audit reports suspected unreferenced symbols and scripts with the evidence classification `advisory`; cleanup requires proof from platform-specific compiler, release packaging, public API and dynamic runtime tests. Delete only a confirmed dead module after the owner-facing feature/compatibility check. The exact version's CI must supply counts and candidate paths; this report does not invent them.

## Historical transfer-ledger privacy migration

The 0.99.34 source now sanitizes **previously persisted** untrusted failure diagnostics when the local transfer ledger loads, not only new failures. It retains the app's own fixed crash-interruption message and preserves retry linkage and committed progress. A regression fixture includes a signed URL/userinfo secret in an old failed transfer, verifies that the resumed UI row contains only a safe network category, and that the on-disk database row was updated without the secret. This is local data migration, not a claim that pre-existing copies/backups or logs have been erased.

## Cross-adapter staged-download durability guard

The 0.99.34 source now verifies **every non-Agent download adapter's local staging file** against the transfer manager's actual reported count and issues `sync_all()` before the staging file can replace the user destination. This is in the shared `run_download_task` path, not just the SFTP/FTP branches, so S3/WebDAV/HTTP/Dropbox/OneDrive/Drive/Box/SaaS cannot bypass this local integrity preflight when the adapter reports success. It does **not** prove the remote server sent its full logical object when it supplied no trustworthy source size; real cloud-provider integrity checks remain required. A Rust regression covers matched, shortened, oversized and empty local stages.
