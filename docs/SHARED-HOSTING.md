# Shared-hosting compatibility

ByFTP is intentionally compatible with common shared-hosting layouts, chroot/virtual FTP roots and hosting-account credentials.

Use the full FTP username provided by the host, including forms such as `account@domain` when required. A web-root path such as `public_html` is resolved inside the authenticated account namespace rather than being rewritten as an arbitrary server filesystem root.

## Connection diagnostics

After a successful connection, ByFTP derives a small shared-hosting diagnostic result from the **initial remote root listing that the client already needed** to establish a usable session. Diagnostics do not create a second connection, scan ports, try alternative hosts or contact an external service.

The recognized web-root candidates use a deterministic priority:

1. `public_html`
2. `httpdocs`
3. `htdocs`
4. `www`
5. `web`
6. `html`

Only directory entries are candidates. The desktop diagnostic layer also rejects symlink candidates. A match is informational: ByFTP **does not automatically open the detected directory and does not save it into the connection profile**. Existing saved remote paths and explicit user navigation remain authoritative.

The diagnostic payload is deliberately non-secret. It can report whether the selected transport is secure, whether the current root represents an authenticated account root or an SFTP home, the detected web-root name and the initial entry count. It does not contain passwords, passphrases, private keys, usernames, SFTP fingerprints, certificate material or server banners.

A missing common web-root match does not mean the connection is broken. Hosting providers can use custom document-root names, nested domains, reseller layouts or application-specific paths. In those cases, navigate using the path supplied by the host rather than treating the diagnostic as authoritative configuration.

## Common connection failures

ByFTP keeps raw transport/tool output behind its safe localized error layer. Common shared-hosting failures include:

- FTP `421` connection-limit responses — the hosting account or source address has too many active sessions.
- FTP `425`/`426` data-channel failures — commonly associated with passive data-channel reachability, firewall/NAT behavior or a server-side transfer interruption.
- TLS/certificate failures — the FTPS certificate chain, endpoint identity or validity period cannot be verified safely.
- FTP `552` / quota failures — the account or destination filesystem has insufficient permitted space.
- authentication failures — verify the full hosting username, port, selected protocol and password rather than disabling transport security.

Diagnostics do not attempt unsafe automatic recovery such as certificate bypasses, trust-all TLS, SFTP host-key bypasses, arbitrary PASV destinations or port scanning. Server configuration and endpoint identity remain fail-closed security boundaries.

## Desktop

The desktop FTP/FTPS implementation uses login/home-relative control and transfer paths, supports **MLSD to LIST fallback** for older servers and remembers the usable listing mode for the session. **Passive connections** include compatibility handling that avoids trusting unsafe private PASV addresses in common NAT/shared-hosting scenarios.

The desktop connection engine reuses its existing first listing for diagnostics. Windows displays the resulting secure/plain transport and web-root/account-home status, but the selected profile path or user-entered path still determines the actual initial remote directory.

## Android

Android records the server `PWD` after FTP/FTPS login and treats that working directory as the UI `/`. For example, if the account logs in at `/home/example`, UI `/public_html` maps to `/home/example/public_html` rather than forcing `/public_html` outside the account home.

If the server cannot report `PWD`, Android uses login-relative paths (`public_html/...`). Traversal (`..`), duplicate separators, backslashes, NUL characters and other noncanonical UI paths are rejected before an FTP operation is sent.

Android derives diagnostics from the same initial `list("/")` used to populate the first file view. The result is shown in the connection summary only; it is not written into `ConnectionPresetStore` and does not trigger automatic navigation.

## iOS

The native iOS client follows the same account-root concept. It maps UI `/` to the authenticated FTP login working directory when the server reports `PWD`; otherwise it stays login-relative. `public_html` therefore remains inside the authenticated hosting account instead of being rewritten as an unrelated server filesystem path.

iOS rejects traversal, duplicate separators, dot components, backslashes, NUL/control characters and noncanonical server login roots. Passive mode prefers EPSV and falls back to PASV, but deliberately ignores the host address supplied inside a PASV response and reconnects the data channel only to the user-selected server host. This prevents a server response from redirecting file data to an unrelated address while remaining compatible with NAT/shared-hosting servers that advertise private PASV addresses.

The SwiftUI browser displays diagnostics from the existing initial listing in a **Shared hosting** section and explicitly states that detected paths are not opened or saved automatically. Diagnostics are cleared on failure/disconnect and are not part of the Keychain connection preset.

## Transport security

Plain FTP remains available for compatibility but is unencrypted. Prefer FTPS or SFTP where implemented. Desktop and Android support FTP, explicit/implicit FTPS and SFTP. iOS supports FTP and implicit FTPS; explicit FTPS and SFTP are intentionally not claimed on iOS until reviewed native implementations preserve the required trust and credential boundaries.

SFTP remains fail-closed for host-key verification where supported, while FTPS keeps platform certificate-chain and endpoint/hostname verification enabled. Do not disable those checks to accommodate a misconfigured host; verify or correct the server endpoint instead.
