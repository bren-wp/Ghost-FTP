# Ghost FTP by Brendigo — Windows and Linux transfer acceptance matrix

**Current development source:** 0.99.77 Preview (unpublished). **Latest published:** v0.99.76 Preview. This table records *automated test scope*, not an assertion that every user-facing feature has been certified. A green CI job proves only the cases that actually ran on its specified operating system.

| Feature / scenario | Linux automated coverage | Windows automated coverage | Required additional acceptance |
| --- | --- | --- | --- |
| FTP sign-in and real upload/download | Native Rust E2E against local pyftpdlib server | Native Rust E2E against loopback pyftpdlib (new blocking job; must pass) | Target servers, proxies and network interruptions |
| Explicit FTPS upload/download | Native E2E vs vsftpd, TLS data sockets, certificate hostname rejection | Not yet real-network tested | Native Windows real FTPS E2E with trusted and invalid certificates |
| SFTP password/key authentication and upload/download | Native OpenSSH E2E, changed host-key rejection, recovery, encrypted private key | Windows filesystem/credential tests only; no Windows real-network SFTP E2E | Native Windows OpenSSH SFTP, host-key and encrypted key cases |
| Zero-byte, UTF-8 names, binary boundary sizes | FTP/FTPS/SFTP protocol suite | FTP in new real-network test | FTPS/SFTP on Windows; different filesystems and encodings |
| Rename, collisions, delete, readonly paths | FTP/FTPS/SFTP network and protected-file regression tests | FTP network collision and readonly fixture plus Windows filesystem regressions | Permission matrix on NTFS/external storage and real production hosts |
| Concurrent 1/4/8 transfers and full-file SHA-256 | Repeated FTP/FTPS/SFTP real-network matrix | New native Windows FTP 1/4/8 network SHA-256 matrix (2 runs; pending latest-head CI confirmation) | Windows FTPS/SFTP concurrency, throughput, cancellation and failure injection under load |
| Interrupted transfers, restart/retry and saved profiles | Rust filesystem and transfer/reconnect regression suites | Windows local filesystem / credential regression suite | Manual real-network process-kill/resume test and disk-full conditions on both platforms |
| Cloud providers, scheduler, sync, terminal, Agent, upgrades, UI accessibility | Source/unit tests and selected workflow guards only | Source/unit tests and native packaging checks only | Connected-provider E2E with test accounts; manual native keyboard/screen-reader and installer acceptance |

## Evidence policy

- GitHub Actions check names are the proof of execution, not a README claim. The Linux real network test is `Real FTP / FTPS / SFTP roundtrip`; the Windows real network FTP test is `Real Windows FTP upload/download roundtrip`; the combined job is `Desktop security gate`.
- Both runners must use disposable, loopback-only servers and random per-run credentials. Server and client logs must not include raw passwords, tokens or private keys.
- A skipped, canceled, queued or failed test is **not passed**. Do not label the Windows FTP scenario or new parallel 1/4/8 Windows matrix verified before each reports success on the latest pull-request SHA.
- Build success on Windows and Linux is distinct from user-interface pixel parity, device acceptance, certificate trust and end-to-end functionality of external services.
- Native screenshots must come from running apps on the claimed version and operating system, with approval of the reference source recorded before declaring pixel parity.
- `main` branch protection remains an external owner-admin task (active issue #320; former #302 was deleted); source-only protection scripts do not authorize closing it.

## Reproduction

The workflow `.github/workflows/ghostftp-protocol-e2e.yml` provisions the Linux and Windows test servers and executes the checked-in Rust integration suite using `--locked`. Windows requires `GHOSTFTP_WINDOWS_FTP_E2E=1` and local per-run `GHOSTFTP_E2E_FTP_PORT`, `GHOSTFTP_E2E_USERNAME`, and `GHOSTFTP_E2E_PASSWORD` under a private testing environment. Its configured blocking job runs both `real_windows_ftp_roundtrip` and `real_windows_concurrent_ftp_roundtrips` twice, using disposable sessions and full-file SHA-256 checks for 1, 4 and 8 simultaneous transfers. This matrix measures integrity, not transfer speed or real-world device certification. Never use production FTP accounts for these tests.
