# FTP / FTPS / SFTP End-to-End QA

Ghost FTP has a dedicated protocol E2E workflow at `.github/workflows/ghostftp-protocol-e2e.yml`.

The workflow starts real local protocol servers on an Ubuntu GitHub runner and exercises the same Rust backend used by the desktop application. It does not use fake transfer state or a browser simulation.

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


## RC11 release gate

A Ghost FTP RC11 release must not be published unless the real FTP, explicit FTPS and SFTP workflow completes successfully for the exact release source commit.
