# Ghost FTP browser extensions

**Ghost FTP Connection Helper** is the official Ghost FTP browser companion for working with FTP, explicit FTPS and SFTP through the locally installed Ghost FTP engine.

The browser extension does not implement raw FTP/SFTP sockets itself. Browser sandboxes do not provide those protocol capabilities. Instead, the extension uses the browser's **Native Messaging** API to talk to a local Ghost FTP bridge process. That process reuses the same connection, host-key verification, protected profile storage, file-operation and transfer-engine code as the native Ghost FTP application.

## Official browser packages

The maintained source is split into one shared runtime plus browser-specific Manifest V3 metadata:

- `chrome/manifest.json` — Google Chrome package metadata and service worker;
- `edge/manifest.json` — Microsoft Edge package metadata and service worker;
- `firefox/manifest.json` — Mozilla Firefox metadata, background script and stable Gecko signing identity;
- `opera/manifest.json` — Opera Chromium package metadata and service worker;
- `shared/` — the maintained HTML, CSS, JavaScript, background relay and icon runtime;
- `cmd/ghostftp-native-host/` — the local Native Messaging host backed by the Ghost FTP Engine.

Run `python scripts/build_browser_extensions.py`. It produces exactly four deterministic ZIP archives in `dist/browser/`:

- `Ghost-FTP-<version>-Chrome-Extension.zip`
- `Ghost-FTP-<version>-Edge-Extension.zip`
- `Ghost-FTP-<version>-Firefox-Extension.zip`
- `Ghost-FTP-<version>-Opera-Extension.zip`

The official product name is **Ghost FTP** and the official extension name is **Ghost FTP Connection Helper**. `BRAND.json`, the package builder and regression suite enforce those identities. Brand drift, extra web permissions, browser storage, remote code or unsupported browser networking fail the maintained build contract.

## Browser workspace

The popup provides three production surfaces:

1. **Connection** — saved connections, FTP/FTPS/SFTP settings, protected profile saving, connect/disconnect and explicit SFTP host-key fingerprint verification.
2. **Files** — a two-pane local/remote workspace with folder navigation, create folder, rename, delete, upload and download.
3. **Transfers** — the real Ghost FTP transfer queue with progress, speed, cancellation, retry, pause/resume and finished-job cleanup.

A pasted `ftp://`, `ftps://` or `sftp://` address can still be used as a local form-filling shortcut. The parser rejects unsupported schemes, missing hosts, oversized input and literal or percent-decoded control characters. Any password embedded in a URL is deliberately discarded rather than copied into the connection form.

## Native bridge boundary

The extension requests exactly one browser permission: `nativeMessaging`. It requests no host permissions, tab access, browsing-history access or extension storage permission.

The background relay opens `com.ghostftp.bridge` only when the popup needs the local engine or a transfer is active. It keeps the native port alive for active transfers, polls only the local transfer event stream while work is active, and closes the native port when there are no clients, pending requests or active transfers. It does not use a permanent interval or hidden HTTP/WebSocket service.

The native host uses bounded length-prefixed Native Messaging frames and strict JSON decoding. It routes only a fixed operation set into the existing Ghost FTP Engine. Local file access is confined to a folder explicitly chosen with the operating-system folder picker; traversal and symlink escape attempts are rejected. Native errors pass through the existing user-safe Ghost FTP error mapping instead of returning raw curl, OpenSSH or operating-system diagnostics.

Server protocol traffic goes **directly** from the local Ghost FTP engine to the FTP/FTPS/SFTP server selected by the user. There is no Ghost FTP cloud proxy and no project-operated transfer relay.

## Credentials and saved connections

The extension **does not store credentials in browser storage**. Password and passphrase inputs live only in the popup's memory while the popup is open and are sent to the local Ghost FTP bridge only when the user explicitly connects or chooses to save a profile.

If the user elects to save credentials, the native Engine stores them through Ghost FTP's existing protected profile store. Public profile data returned to the extension exposes only non-secret fields and booleans such as whether a protected password/passphrase exists; plaintext saved secrets are not returned to the browser.

For SFTP, first contact remains fail-closed. The Engine obtains the server host-key fingerprint and requires explicit user confirmation. A remembered fingerprint is bound to the saved endpoint; a later mismatch blocks the connection rather than silently accepting a changed key.

## Privacy and security contract

- **No telemetry** and **no tracking**.
- **No remote code**; HTML, CSS, JavaScript and the icon are packaged locally.
- Exactly one browser permission: `nativeMessaging`.
- No host permissions, content scripts, tab permission, browsing-history permission or browser storage permission.
- The extension **does not read the active tab**, cookies or page content.
- The extension **does not store credentials in browser storage** or persist connection form values.
- No HTTP request, WebSocket, analytics SDK, advertising SDK or external crash reporter is used by the extension runtime.
- FTP/FTPS/SFTP connections are performed by the local Ghost FTP engine and go directly to the server selected by the user.
- Local file access is user-initiated and rooted to the explicitly selected local folder.
- Firefox declares `data_collection_permissions.required = ["none"]` and uses the stable ID `ghostftp-connection-helper@ghostftp.com` for signed distribution.

See `PRIVACY.md` for the concise data-handling statement.

## Local validation

```bash
go test ./cmd/ghostftp-native-host
python scripts/build_browser_extensions.py --check
python scripts/test_browser_extensions_contract.py
python scripts/build_browser_extensions.py
```

The dedicated GitHub Actions workflow validates the browser/native contract, builds all four deterministic ZIPs, verifies their contents and publishes them as CI artifacts.

## Development testing

A browser package alone is intentionally insufficient for live FTP/SFTP work: the matching local Native Messaging host must also be installed and registered for that browser. This avoids pretending that a browser extension can create raw FTP/SFTP sockets by itself.

Build the packages first and extract the browser ZIP you want to test. Chrome and Opera use their Chromium extension-development pages, Edge uses `edge://extensions`, and Firefox uses `about:debugging#/runtime/this-firefox` with the generated `manifest.json`.

Do not maintain browser-specific copies of `background.js`, `core.js`, `popup.js`, `popup.css`, `popup.html` or the Ghost FTP icon. All four packages are produced from `shared/`; only their manifests remain browser-specific.
