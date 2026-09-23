# Ghost FTP Security

## Security model

Ghost FTP's native desktop implementation uses real protocol/session backends rather than a screenshot or simulated transfer layer. FTP, FTPS and SFTP are distinct protocol paths; SFTP/SSH identity and key handling, TLS certificate validation, filesystem errors, permission failures, cancellation, retry and reconnect logic must fail closed and surface actionable errors.

Credentials and private-key material must never be printed to routine logs. Native credential references should use OS-protected storage where supported. User-supplied remote paths and local paths must be treated as data, not command text. Any shell integration must quote paths defensively and avoid passing credentials through command lines.

## Diagnostic and terminal privacy

Terminal inline suggestions use process-memory-only history. Shell commands are not persisted to WebView storage, and commands that look credential-bearing are excluded from suggestion history.

Notification-center history is also session-only. User-facing diagnostics are filtered through credential redaction before they are stored in UI state or shown to the user. Upgrade startup removes legacy terminal/notification history keys written by older release candidates.

## Updates

Native update metadata is retrieved only from the configured official `https://ghostftp.com/updates/latest.json` endpoint. The Tauri updater verifies the configured signature before installing an artifact. Package SHA-256 values should also be published with releases. A failed download, checksum/signature verification or apply step must leave the currently installed version usable.

## Local runtime boundary

The Go compatibility runtime binds its local control server only to `127.0.0.1` on an ephemeral port, sets no-store/nosniff/referrer/CSP headers and exposes local filesystem utilities to its own embedded browser window. It is a fallback QA/runtime host, not the native protocol release.

## Reporting

Security reports should be submitted through the official Ghost FTP support/security route published at **https://ghostftp.com/**. Do not include production passwords, SSH private keys or customer data in a report. Include the Ghost FTP version, OS, reproducible steps and sanitized logs where possible.

No statement in this file claims that the software is vulnerability-free or that this package received an independent security certification.
