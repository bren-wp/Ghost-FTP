# Ghost FTP — parallel FTP/FTPS/SFTP and >4 GiB transfer acceptance

Date: 17 November 2026. Status: **automation added; run results and physical stress acceptance remain distinct**. Development PR: [#298](https://github.com/bren-wp/Ghost-FTP/pull/298).

## CI: real protocols, 1/4/8 workers

The `ghostftp-desktop/src-tauri/tests/protocol_e2e.rs` suite starts against authenticated localhost FTP, explicit FTPS and SFTP servers in `.github/workflows/ghostftp-protocol-e2e.yml`. The `real_concurrent_protocol_roundtrips` test uses **separate connections**, not a shared connection or mocked filesystem. For each protocol, it uploads and downloads a **1 MiB + 17 B** deterministic binary stream using concurrency 1, 4 and 8. Every worker checks full downloaded SHA-256 and byte length, removes its own remote payload, and closes its isolated connection. In total: 39 network roundtrips (13 per protocol).

`PatternReader` and `DigestSink` are bounded-memory generators/checkers. These tests exercise actual protocol read/write APIs, **not** the complete application transfer-queue UI, staging promotion, cancellation, or connection-pool scheduling.

The workflow continues to run the existing separate zero-byte, chunk-boundary, pause/resume, simulated disk-failure, invalid TLS hostname, key authentication, symlink and Windows security regressions.

## Optional physical large-transfer test

`opt_in_real_over_four_gib_parallel_stress` uses **two simultaneous uploads and downloads**, each **4,294,967,312 B** (`u32::MAX + 17`), sequentially for FTP, FTPS and SFTP. It checks complete SHA-256 and byte length for each real transfer; this is **not** the sparse 4 GiB metadata regression in `remote_upload_staging.rs`.

**Do not run this test on normal GitHub-hosted PR runners.** Provision a disposable Linux test host with substantial free disk space (**at least 20 GiB**, more if server staging/overhead requires it), sufficient network bandwidth, a generous time budget, test-only credentials, and localhost FTP, FTPS and SFTP services. The exact E2E variables and ephemeral fixture setup are defined in `.github/workflows/ghostftp-protocol-e2e.yml`. From `ghostftp-desktop/src-tauri`, with those environment variables exported:

```bash
cargo test -p ghostftp --test protocol_e2e --locked real_concurrent_protocol_roundtrips -- --nocapture

GHOSTFTP_PROTOCOL_E2E=1 GHOSTFTP_STRESS_4GIB=1 \
cargo test -p ghostftp --test protocol_e2e --locked opt_in_real_over_four_gib_parallel_stress -- --nocapture
```

**No large-network-transfer result may be reported as passing before the opt-in test has actually run and its evidence has been retained.** Both tests deliberately skip without the required environment gate. The optional stress case must not be confused with the default 1/4/8 matrix.

## Further acceptance required before a Stable designation

- Run full file sizes 0 B, 1 B, 64 KiB boundaries, 1 GiB, 4 GiB + 1 B, with concurrency 1/4/8 on **physical machines** against real network servers; collect SHA-256 and source/remote size evidence.
- Introduce deterministic network disconnect, reconnect, pause, cancellation, destination permission denial, disk exhaustion and process crash at multiple transfer offsets, including overwriting existing files.
- Assert remote original contents survive failed staging and uncommitted transfer; document cleanup and orphan staging behavior.
- Verify Windows Credential Manager and Linux Secret Service storage, missing/locked vault behavior, session persistence and native first-frame UI on real Windows/Linux installations.
- Confirm signed-distribution provenance, package install/uninstall and accessibility on target operating systems. Ordinary CI artifacts are not a substitute for this physical acceptance.

The Preview release remains gated on independently passing required CI checks, especially the npm development-dependency security policy tracked at [#299](https://github.com/bren-wp/Ghost-FTP/issues/299).
