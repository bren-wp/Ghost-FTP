# Ghost FTP Transfer QA — RC10

## Implemented transfer engine scope

The authoritative Ghost FTP desktop source contains real FTP, FTPS and SFTP session/transfer paths, queue state, retries, pause/resume controls, bandwidth throttling, conflict handling and synchronization support.

RC10 also corrects transfer-state UI behavior:

- Completed rows do not expose Cancel.
- Skipped and Canceled are explicit terminal states.
- Retry All is available for failed transfers.
- Active queue filtering no longer duplicates terminal history.
- Progress is represented semantically.

## CI truth

Rust workspace tests and frontend production builds are release quality gates. Compatibility tooling does not simulate successful file transfers and is not used as proof of protocol correctness.

## Real-server acceptance still required

Before FINAL, execute against controlled test servers:

- FTP upload/download/rename/delete.
- FTP interrupted-transfer recovery/resume where supported.
- Explicit FTPS.
- Implicit FTPS where supported.
- Invalid/expired TLS certificate behavior.
- SFTP password authentication.
- SFTP private-key authentication.
- Unknown/changed host-key refusal and acceptance flows.
- Large-file transfer.
- Pause/resume/cancel/retry.
- Concurrent-transfer limits.
- Bandwidth limits.
- Overwrite/skip/rename conflict behavior.
- Timestamp preservation where supported.
- Optional checksum verification where both sides can calculate it.

Status: **production transfer implementation exists; real FTP/FTPS/SFTP end-to-end acceptance remains open before FINAL.**
