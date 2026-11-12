# Phase 4 — Object-store GET integrity and durable staging (v0.99.54)

**Scope:** `ghostftp-desktop/src-tauri/src/transfer.rs` — `run_object_download` and shared staging verification.

## Failure mechanism

A queue row may have a zero/unknown or stale remote size. Before this change, S3/object-store `GET` streamed bytes into staging and reported success after EOF without validating the response's own object size. The final manager verified only that the staging file matched reported bytes and the original queue estimate (when nonzero). An incomplete response could therefore be accepted when initial size metadata was unknown. A malicious oversized response could write beyond the advertised object size.

## Change

- Read the byte length from `get.meta.size` on the same GET response used to stream data.
- Reject counter overflow and data exceeding that declared size before writing each chunk.
- `flush`, `sync_all` and verify the on-disk bytes match both declared and received sizes.
- Preserve the existing atomic sibling-staging and recovery-backup promotion design; failed downloads never get promoted.
- Test zero, short, oversized and overflow conditions in Rust.

## Evidence and remaining work

This branch's tests and CI are authoritative for unit/build/format checks. Test sources do not prove that a third-party provider returned an authentic content checksum or immutable object version. Cloud-account live E2E, authorized fault injection and a full >4 GiB stress matrix are pending. No real performance measurements or UI pixel parity claims are made.

**Security classification:** Preview. Do not mark stable without independent acceptance.
