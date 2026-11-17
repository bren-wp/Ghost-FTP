# Ghost FTP — 0.99.69 source fingerprint and sparse 4 GiB assessment

Date: 17 November 2026. Status: **Scoped source change / awaiting exact-head CI and native acceptance.** Previous public Preview: v0.99.68.

## Finding: metadata equality did not prove content identity

`LocalUploadSnapshot` in `ghostftp-desktop/src-tauri/src/transfer/remote_upload_staging.rs` checked length, modification time and OS file identity. A local file can be rewritten in place with the same length and its original timestamp restored. The existing metadata guard could then accept modified content and promote the staged remote file. This is a risk for FTP, FTPS and SFTP staged uploads, not a demonstrated remote exploit.

## Mitigation

Capture a SHA-256 fingerprint of first/middle/last 64 KiB source windows while validating the same file handle and pathname's metadata before/after local sampling. Hash the entire source if no larger than 64 KiB. Verify the stored fingerprint at the start of each automatic retry and again before remote promotion. A mismatch refuses to promote. The extra sampling budget is at most 192 KiB per check regardless of source size, plus metadata checks and open/seek overhead. This is intentionally bounded to protect large-transfer responsiveness.

**Residual risk:** Samples cannot cover arbitrary mutations outside their ranges; neither a complete local hash nor a remote post-upload SHA-256 is present. A source or path may still be changed in a tight race outside the sampling/verification boundary. The file's mere metadata equality is not a sufficient security guarantee. FTP RNTO non-atomicity and provider-specific staging semantics remain unchanged.

## Regression expectations

| Test | Automated scope | Not proved |
| --- | --- | --- |
| Same-length rewrite with restored mtime | Local content fingerprint rejects metadata-preserving sampled mutation | All unobserved simultaneous writes |
| 4 GiB + 16 bytes sparse file (Linux) | 64-bit snapshot length and tail sampling, constant-memory scanning | 4 GiB real network upload/download |
| Existing FTP, FTPS, SFTP E2E | Real localhost protocol roundtrips | Internet reliability and arbitrary host behavior |
| Windows/Linux native build | Compilation and CI-host package smoke | Every target device, OS vault/reboot and accessibility |

## Remaining independent acceptance

Run disposable FTP, explicit FTPS, SFTP and cloud hosts. Exercise 0 B, 1 B, 64 KiB, 128 KiB, 1 GiB, 4 GiB+1 byte uploads/downloads, with concurrency 1/4/8 and forced network, disk, permission, pause, cancel, reconnect and process crashes. Record source/remote full SHA-256 and file sizes. Confirm original remote files survive failed overwrite staging, and inspect orphan temporary files. Independently verify Windows Credential Manager and Linux Secret Service across logout/reboot and missing/locked vaults. Conduct real native keyboard, scaling and first-frame UI acceptance.

The previous repository check found `main` without branch protection and empty rulesets; enforce reviews and required CI in repository settings before Stable. This document does not claim those settings were changed.

## Dead-code/security audit boundary

The repository's automated reachability and static security regression tools must pass on the exact PR SHA. These scans identify candidates and unsafe patterns, but cannot certify absence of vulnerabilities or justify deleting apparent dead code without proving compatibility and reachable call sites.
