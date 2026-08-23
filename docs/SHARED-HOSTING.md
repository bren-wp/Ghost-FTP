# Shared-hosting compatibility

ByFTP is intentionally compatible with common shared-hosting layouts and credentials.

Use the full FTP username provided by the host, including forms such as `account@domain` when required. A remote path such as `public_html` is resolved relative to the account login/home namespace rather than being rewritten as an arbitrary server-root path.

FTP and FTPS prefer modern listing behavior and can fall back from MLSD to LIST for legacy hosts. Passive connections are used without trusting an unsafe private PASV address in NAT scenarios.

SFTP remains fail-closed for host-key verification. Do not disable TLS certificate verification or SFTP host-key checks merely to accommodate a host; fix the server configuration or verify the correct endpoint instead.
