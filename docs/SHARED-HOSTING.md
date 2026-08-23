# Shared-hosting compatibility

ByFTP is intentionally compatible with common shared-hosting layouts, chroot/virtual FTP roots and hosting-account credentials.

Use the full FTP username provided by the host, including forms such as `account@domain` when required. A web-root path such as `public_html` is resolved inside the authenticated account namespace rather than being rewritten as an arbitrary server filesystem root.

## Desktop

The desktop FTP/FTPS implementation uses login/home-relative control and transfer paths, supports MLSD with LIST fallback for older servers and remembers the usable listing mode for the session. Passive-mode compatibility avoids trusting unsafe private PASV addresses in common NAT/shared-hosting scenarios.

## Android

Starting with 1.1.1, Android records the server `PWD` after FTP/FTPS login and treats that working directory as the UI `/`. For example, if the account logs in at `/home/example`, UI `/public_html` maps to `/home/example/public_html` rather than forcing `/public_html` outside the account home.

If the server cannot report `PWD`, Android uses login-relative paths (`public_html/...`). Traversal (`..`), duplicate separators, backslashes, NUL characters and other noncanonical UI paths are rejected before an FTP operation is sent.

## Transport security

Plain FTP remains available for compatibility but is unencrypted. Prefer FTPS or SFTP. SFTP remains fail-closed for host-key verification, while FTPS keeps platform certificate-chain and endpoint/hostname verification enabled. Do not disable those checks to accommodate a misconfigured host; verify or correct the server endpoint instead.
