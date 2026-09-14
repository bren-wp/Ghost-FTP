# Ghost FTP Web FTP

`web/ftp` is the browser-facing Ghost FTP workspace for FTP, explicit FTPS and fingerprint-pinned SFTP.

## Network boundary

A web browser cannot open raw FTP/FTPS/SFTP TCP sockets directly. The browser therefore sends a requested operation over HTTPS to `api.php`; the PHP server opens the FTP/FTPS/SFTP connection, performs that single operation and closes it. This is an **ephemeral server-assisted transport**, not a claim of browser-direct FTP.

The supplied application has no registration, user database, telemetry, analytics or application transfer-history storage. Credentials are kept in JavaScript page memory and sent with each request; the PHP application does not persist them. Hosting infrastructure can still observe runtime request data and may have its own access/error logs. Use trusted hosting and HTTPS only.

## Requirements

- PHP 8.2+
- cURL extension for FTP/explicit FTPS
- ssh2 extension for SFTP
- HTTPS
- outbound access to the target server ports

Private/reserved destination addresses are blocked by default to reduce SSRF risk. A trusted private deployment may explicitly set `GHOSTFTP_WEB_ALLOW_PRIVATE=1`; never enable that on an untrusted public deployment.

SFTP requires an expected `SHA256:...` host-key fingerprint and verifies it **before password authentication**. Unknown/mismatching host identity fails closed.

Upload, download and editor size ceilings are defined in `config.php`.
