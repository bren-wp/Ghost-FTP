# Installation and first connection

## Windows

Use the release asset matching the machine architecture:

- `ByFTP-<version>-Setup-x64.exe` for normal 64-bit Windows installation.
- `ByFTP-<version>-Portable-x64.exe` for a portable 64-bit run.
- x86 equivalents are available for supported 32-bit Windows systems.

The installer and portable executable are produced from the same tested source revision. Verify the release SHA-256 file before deployment when integrity matters.

### First connection

1. Select FTP, explicit FTPS, implicit FTPS or SFTP.
2. Enter the server, port and full username supplied by the hosting provider.
3. Enter the password. For SFTP, a private key can be selected where appropriate.
4. Click **Connect**.
5. For a new SFTP host key, verify the displayed fingerprint out of band before accepting it.

Saved profiles can retain connection details and, when explicitly approved by the user, protected credentials. Changing the server/account identity invalidates credentials that no longer belong to that identity.

## Linux

Install the architecture-appropriate `.deb` package from the release. The terminal client uses the same transfer engine. FTP/FTPS requires `curl`; SFTP requires OpenSSH tools. Run `byftp` and follow the prompts. Use `language <code>` to change the persisted UI language.

## macOS

Install `ByFTP-<version>-macOS-Universal.pkg`. The package contains a universal amd64/arm64 terminal build. FTP/FTPS uses the system curl path; SFTP uses OpenSSH.

## Common connection problems

- **Authentication rejected:** verify the full hosting username; shared-hosting usernames often include a domain.
- **Connection refused:** verify protocol and port.
- **FTP data connection failure:** passive FTP ports or a firewall may be blocked.
- **FTPS certificate error:** do not disable verification; correct the certificate/server configuration.
- **SFTP host key changed:** stop and verify the change with the server administrator.

ByFTP cannot override provider-side quotas, filesystem permissions, FTP session limits, TLS policy or firewall rules.