# Ghost FTP — Windows/Linux transfer security and dead-code follow-up (2026-11-07)

## What changed in this branch

This is a **targeted implementation and source-level review** continuing the merged v0.99.34 audit. It is not a completed cross-provider certification, penetration test, or 1:1 pixel-parity signoff.

### Transfer integrity

- **SFTP downloads:** before reusing a staged partial download, compare **every previously committed byte** from the local staging file against the current remote source (bounded 64-KiB memory). If bytes differ, discard the old offset and restart the download. Keep existing byte-count, remote-size and staging durability checks.
- **SFTP uploads:** compare the entire existing remote upload prefix with the current local source before opening a writable resumable handle. Length equality is insufficient; an equal-length replacement now forces restart from byte zero. Preserve post-upload remote size verification.
- **Ghost FTP Agent downloads/uploads:** ranged daemon reads verify each byte of the local/remote committed prefix before allowing a resumed whole-file copy. A changed or shortened prefix forces zero-offset restart. Delta-mode full hash verification remains a separate existing mechanism.
- **FTP and explicit FTPS:** standard REST/APPE and SIZE do not provide a trusted content digest. A paused transfer now **restarts from zero** instead of appending to an unverified prefix. This favors integrity over bandwidth until a verified-prefix mechanism and server interoperability tests exist.
- **Regression coverage:** same-length remote replacement beyond the first 64-KiB block, valid prefix, zero-length prefix, short source and full match. Security gate checks source call sites and enforces the no-FTP-range-resume policy.

This does **not** establish a cryptographically immutable remote snapshot. A third-party server may change a same-length file *during* a live transfer. Such scenarios require server-side checksums or immutable revisions and live fault-injection tests.

## Protocol review matrix

| Adapter | Current intended behavior / checks | Outstanding acceptance work |
| --- | --- | --- |
| FTP | Binary RETR/STOR, SIZE when supported, staged downloads; restart interrupted copies from zero | Live Windows IIS/proftpd/vsftpd tests, SIZE unavailable, ABOR, proxy/NAT, non-UTF8 names, forced reset |
| Explicit FTPS | FTP operations with validated TLS certificate identity | TLS renegotiation, CA rotation, passive data channel, expiry, revoked server cert |
| SFTP/SSH | Pinned/approved host key, full resumed prefix comparison, staged download, size confirmation | Concurrent same-length remote edits, host-key rotation, permission errors, multi-GB transfers |
| Ghost FTP Agent | Ranged reads/writes, verified resumption, hashed delta on supported daemons | Daemon disconnect/restart, concurrent writers, same-length remote mutation under load |
| S3-compatible storage | Object GET, PUT/multipart | Real R2/B2/MinIO/cloud endpoints, ETags vs hash, partial multipart failures |
| WebDAV | Streaming GET/PUT | Auth challenge, MOVE/COPY conflicts, quota, lock and 412 response cases |
| HTTP/HTTPS | Download-only by design | Chunked/encoded bodies, redirects, 206 range ambiguity, interrupted downloads |
| Dropbox, OneDrive, Google Drive, Box | Provider-specific OAuth and upload/download adapters | Live scopes/token refresh, 429 and quota, multipart, large files, exact provider responses |
| Shopify, HubSpot, Dynamics | Product/media/API operations according to adapter permissions | Real tenants, API capabilities, size/rate limits, eventual consistency and idempotency |

Only advertise an operation in the UI when the adapter supports it. Upload over plain HTTP is **not** an implemented feature, and plaintext FTP is **not encrypted**.

## Security/dead-code verification checklist

1. Execute `node --test tools/audits/full-security-code-audit.test.mjs tools/audits/full-dead-code-audit.test.mjs` and both audit CLI scripts on the **exact head SHA**. Treat dead-code findings as candidate-only; do not delete reflection-/IPC-/Tauri-reachable functions based solely on text search.
2. Execute `cargo fmt --check`, `cargo check`, `cargo test`, and applicable Clippy jobs; then the dedicated `ghostftp-protocol-e2e.yml` real-server workflow.
3. Build/test on native Windows WebView2 and Linux WebKitGTK. Check keyrings, credential redaction, symlink/reparse-point staging, process cancellation, memory limits and release artifact signatures/checksums.
4. Execute protocol fault-injection: interrupted transfer at 0/64KiB/>4GiB, remote file replacement at identical length, byte-count mismatch, zero-length file, pause/resume, write permission denial and disconnect during finalization.
5. Collect genuine GUI screenshots from the two OS runtimes at reference viewport sizes and compare against available reference ZIPs. Never substitute a concept render for native evidence.
6. Report results per platform and protocol with pass/fail/untested and the exact tested commit. Do not mark 1:1 visual, third-party-cloud, security or production acceptance as complete without this evidence.

## Source-level findings / residual risk

- This change fixes **unverified prefix identity on resumed FTP/SFTP/Agent transfer paths**. It does not change other adapter implementations or imply that their E2E matrix passed.
- The existing automated dead-code and security audit tools remain heuristic/static. No arbitrary exported functions, commands or protocol adapters are deleted.
- A prior measured 0.99.34 baseline reports a **high strict RGB mismatch** for seven Windows/Linux screens; exact pixel parity cannot be claimed without new native runs, matching datasets, locale and capture dimensions.
- The working environment did not provide Windows/Linux GUI runners, protocol accounts or Rust toolchain, so local native builds, full compiler tests, external provider uploads and exact pixel acceptance remain **unverified** here. GitHub CI must be checked separately after the changes land.
