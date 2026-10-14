# FTP / FTPS / SFTP End-to-End QA

Ghost FTP has a dedicated protocol E2E workflow at `.github/workflows/ghostftp-protocol-e2e.yml`.

The workflow starts real local protocol servers on an Ubuntu GitHub runner and exercises the same Rust backend used by the desktop application. It does not use fake transfer state or a browser simulation.

## 0.95.18 — real FTP data-channel IIS-format emulation

The 0.95.18 PR extends the existing **live socket E2E** with two additional authenticated loopback FTP daemons running pyftpdlib. They emit IIS/MS-DOS-format LIST *bytes* via the actual passive FTP data channel; the Ghost FTP production Rust session and RemoteFs code must parse the valid listing (Unicode filename, spaced directory, dotfile and omitted dot/dotdot), and must reject the second listing containing one invalid time alongside good entries.

- Local ports: 2123 valid MS-DOS-format LIST, 2124 malformed mixed LIST.
- Source: `ghostftp-desktop/src-tauri/tests/protocol_e2e.rs` and `.github/workflows/ghostftp-protocol-e2e.yml`.
- Important distinction: this is real FTP transport with **emulated IIS LIST formatting**, **not** Microsoft's IIS daemon. Windows IIS interoperability remains a separate UNVERIFIED acceptance item.
- On the development candidate, passing the gate for the **exact PR source SHA** is required before asserting E2E success. Never claim the fixture has passed merely because it was committed.

## Covered operations

For FTP and explicit FTPS:

- TCP connect and password login;
- passive data connection;
- create directory;
- upload;
- directory listing;
- rename;
- download with byte-for-byte content verification;
- delete file;
- delete directory.

For SFTP:

- unknown host-key rejection;
- password authentication;
- encrypted OpenSSH private-key authentication with passphrase;
- create directory;
- upload;
- directory listing;
- chmod with mode verification;
- rename;
- download with byte-for-byte verification;
- delete file;
- delete directory.

For FTPS certificate handling:

- the E2E runner creates a one-run CA and a server certificate valid only for `DNS:localhost`;
- the CA is installed into the runner trust store;
- connection to `localhost` must succeed;
- connection to `127.0.0.1` must fail because the certificate identity does not match.

This proves certificate validation is active rather than globally disabled.

## Scope

The workflow validates explicit FTPS because that is the FTPS mode implemented by the current native backend. It does not claim implicit FTPS support.

The protocol job runs on changes to the native Rust source and can also be started manually. Normal unit tests remain hermetic: `tests/protocol_e2e.rs` exits successfully without touching the network unless `GHOSTFTP_PROTOCOL_E2E=1` is explicitly set by the dedicated workflow.


## Release gate

A Ghost FTP release must not be published unless the real FTP, explicit FTPS and SFTP workflow completes successfully for the exact release source commit.
