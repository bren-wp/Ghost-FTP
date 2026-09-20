# Transfer QA

The native source contains real FTP/FTPS/SFTP session and transfer paths and transfer-state stores. However, no native Rust/Tauri executable can be compiled in this sandbox, and no external test server is reachable from this isolated environment.

The Go compatibility fallback intentionally does **not** simulate uploads/downloads. Its former fake progress timer and fake "connected" success path were removed. Network Test Connection in that fallback is TCP reachability only.

Before FINAL, execute protocol-specific upload/download tests for FTP, explicit/implicit FTPS as supported, and SFTP; test pause/resume, cancel, retry-after-cancel, reconnect/resume where supported, overwrite/conflict behavior, timestamp preservation, checksum verification, concurrent-transfer limits, bandwidth limits, timeout/backoff and error recovery.

Status: **native transfer QA blocked; no false pass recorded.**
