# Shared-hosting compatibility

ByFTP is intentionally compatible with common shared-hosting layouts, chroot/virtual FTP roots and hosting-account credentials.

Use the full FTP username provided by the host, including forms such as `account@domain` when required. A web-root path such as `public_html` is resolved inside the authenticated account namespace rather than being rewritten as an arbitrary server filesystem root.

## Desktop

The desktop FTP/FTPS implementation uses login/home-relative control and transfer paths, supports **MLSD to LIST fallback** for older servers and remembers the usable listing mode for the session. **Passive connections** include compatibility handling that avoids trusting unsafe private PASV addresses in common NAT/shared-hosting scenarios.

## Android

Android records the server `PWD` after FTP/FTPS login and treats that working directory as the UI `/`. For example, if the account logs in at `/home/example`, UI `/public_html` maps to `/home/example/public_html` rather than forcing `/public_html` outside the account home.

If the server cannot report `PWD`, Android uses login-relative paths (`public_html/...`). Traversal (`..`), duplicate separators, backslashes, NUL characters and other noncanonical UI paths are rejected before an FTP operation is sent.

## iOS

The native iOS client follows the same account-root concept. It maps UI `/` to the authenticated FTP login working directory when the server reports `PWD`; otherwise it stays login-relative. `public_html` therefore remains inside the authenticated hosting account instead of being rewritten as an unrelated server filesystem path.

iOS rejects traversal, duplicate separators, dot components, backslashes, NUL/control characters and noncanonical server login roots. Passive mode prefers EPSV and falls back to PASV, but deliberately ignores the host address supplied inside a PASV response and reconnects the data channel only to the user-selected server host. This prevents a server response from redirecting file data to an unrelated address while remaining compatible with NAT/shared-hosting servers that advertise private PASV addresses.

## Transport security

Plain FTP remains available for compatibility but is unencrypted. Prefer FTPS or SFTP where implemented. Desktop and Android support FTP, explicit/implicit FTPS and SFTP. iOS 1.2.0 supports FTP and implicit FTPS; explicit FTPS and SFTP are intentionally not claimed on iOS until reviewed native implementations preserve the required trust and credential boundaries.

SFTP remains fail-closed for host-key verification where supported, while FTPS keeps platform certificate-chain and endpoint/hostname verification enabled. Do not disable those checks to accommodate a misconfigured host; verify or correct the server endpoint instead.
