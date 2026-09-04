# GhostFTP Security

## Secure defaults

GhostFTP prioritizes FTPS with valid server certificates. Explicit FTPS is the default for new server profiles.

- TLS 1.2 and TLS 1.3 only for FTPS.
- Standard Windows/.NET certificate-chain and hostname validation.
- Certificate revocation uses the Windows offline cache so FTPS validation does not create hidden CRL/OCSP web requests.
- No "accept any certificate" setting.
- Passive data connections only.
- PASV host values are not trusted; data channels connect to the authenticated control host to reduce FTP bounce/NAT abuse.
- FTP command arguments reject CR, LF and NUL characters to block command injection.
- Control replies have line and size limits.
- Connection, command and transfer timeouts are enforced.
- Downloads use temporary `.ghostftp.part` files and are promoted only after a successful transfer.
- Uploads go to a unique temporary remote name and are renamed only after successful completion, reducing partial-file replacement risk.
- Deleting the FTP root directory is blocked.
- Remote filenames are sanitized before writing to Windows paths and are prevented from escaping the selected local directory.
- Saved passwords are optional and protected using Windows DPAPI.

## Plain FTP warning

Plain FTP provides no transport encryption. Use it only when required by a trusted server or isolated network. FTPS should be preferred whenever possible.

## Reporting security issues

Please use the project's GitHub issue tracker for non-sensitive issues. Do not post passwords, private server addresses, private keys, or other credentials in public issues.
