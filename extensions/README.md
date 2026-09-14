# Ghost FTP browser extensions

**Ghost FTP Connection Helper** is the official Ghost FTP browser companion for safely preparing FTP, FTPS and SFTP connection targets locally in a browser popup.

## Official browser packages

The maintained source is deliberately split into one shared runtime plus a browser-specific manifest directory for every supported package:

- `chrome/manifest.json` — Google Chrome package metadata;
- `edge/manifest.json` — Microsoft Edge package metadata;
- `firefox/manifest.json` — Mozilla Firefox metadata and stable Gecko signing identity;
- `opera/manifest.json` — Opera Chromium package metadata;
- `shared/` — the only maintained HTML, CSS, JavaScript and icon runtime.

Run `python scripts/build_browser_extensions.py`. It produces exactly four deterministic ZIP archives in `dist/browser/`:

- `Ghost-FTP-<version>-Chrome-Extension.zip`
- `Ghost-FTP-<version>-Edge-Extension.zip`
- `Ghost-FTP-<version>-Firefox-Extension.zip`
- `Ghost-FTP-<version>-Opera-Extension.zip`

The official product name is **Ghost FTP** and the official extension name is **Ghost FTP Connection Helper**. `BRAND.json`, the package builder and regression suite enforce those identities for official Ghost FTP builds. Brand drift, additional permissions or unsupported remote capabilities fail the maintained build contract.

## What the helper does

Paste an `ftp://`, `ftps://` or `sftp://` target into the popup. The helper validates the URL, rejects unsupported schemes, missing hosts, oversized input and literal or percent-decoded control characters, then displays the protocol, host, port, username and remote path. A generated **Safe target** contains only the scheme, host/port and encoded path. URL username/password, query and fragment data are never included.

The official helper has **no supported browser-to-desktop** custom URI, native-messaging bridge or privileged background service. It also does not launch a mobile application. It remains a local parser/copy companion rather than pretending to provide capabilities the browser sandbox does not grant.

## Privacy and security contract

- **No telemetry or tracking.**
- **No remote code.** HTML, CSS, JavaScript and the icon are packaged locally.
- **Zero browser permissions and zero host permissions.** Official manifests use an empty `permissions` list and define no host permissions.
- The extension **does not store** targets, usernames, passwords, history or parsed results.
- The extension **does not read the active tab**, browsing history, cookies or page content.
- The extension **does not connect to your FTP, FTPS or SFTP server**.
- There is no HTTP request, WebSocket, analytics SDK, crash reporter, browser storage, background script or content script.
- Passwords are detected only to warn the user and are never rendered into result fields or copied into the Safe target.
- Firefox declares `data_collection_permissions.required = ["none"]` and uses the stable ID `ghostftp-connection-helper@ghostftp.com` for signed distribution.

See `PRIVACY.md` for the concise data-handling statement.

## Local validation

```bash
python scripts/build_browser_extensions.py --check
python scripts/test_browser_extensions_contract.py
python scripts/build_browser_extensions.py
```

The dedicated GitHub Actions workflow repeats the contract test, builds all four ZIPs, verifies their contents and publishes them as CI artifacts.

## Unpacked development testing

Build the packages first and extract the browser ZIP you want to test. Chrome and Opera use their Chromium extension-development pages, Edge uses `edge://extensions`, and Firefox uses `about:debugging#/runtime/this-firefox` with the generated `manifest.json`.

The generated packages are the canonical browser inputs. Do not maintain browser-specific copies of `core.js`, `popup.js`, `popup.css`, `popup.html` or the Ghost FTP icon. All four packages are produced from `shared/`, while only their manifests remain browser-specific.
