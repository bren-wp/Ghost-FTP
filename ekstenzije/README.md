# Ghost FTP browser extensions

**Ghost FTP Connection Helper** is the official Ghost FTP browser companion for safely preparing FTP, FTPS and SFTP connection targets locally in the browser popup.

## Official packages

The source is deliberately split into one canonical runtime and three browser manifests:

- `shared/` — the only maintained HTML/CSS/JavaScript/icon runtime;
- `manifests/chrome.json` — Google Chrome package metadata;
- `manifests/edge.json` — Microsoft Edge package metadata;
- `manifests/firefox.json` — Mozilla Firefox package metadata and stable Gecko signing identity.

Run `python scripts/build_browser_extensions.py`. It produces exactly three deterministic ZIPs in `dist/browser/`:

- `Ghost-FTP-<version>-Chrome-Extension.zip`
- `Ghost-FTP-<version>-Edge-Extension.zip`
- `Ghost-FTP-<version>-Firefox-Extension.zip`

The official product name is **Ghost FTP** and the official extension name is **Ghost FTP Connection Helper**. `BRAND.json`, the packaging script and the regression suite enforce those names for official Ghost FTP builds. An altered manifest or popup brand fails the maintained build contract. As with any source-available project, a third party can modify its own fork; such a fork is not an official Ghost FTP build and is not covered by the Ghost FTP release contract.

## What the extension does

Paste an `ftp://`, `ftps://` or `sftp://` target into the popup. The helper validates the URL, rejects unsupported schemes, missing hosts, oversized input and literal or percent-decoded control characters, then displays protocol, host, port, username and remote path. A generated **Safe target** contains only scheme, host/port and encoded path: URL username/password, query and fragment data are never included.

The extension **does not launch the desktop client directly** and does not launch the Android app. Ghost FTP does not currently expose a supported browser-to-desktop URI or native-messaging contract. The browser package therefore remains a local parser/copy companion instead of inventing a privileged bridge.

## Privacy and security contract

- **No telemetry.**
- **No tracking.**
- **No remote code.** HTML, CSS, JavaScript and the icon are packaged locally.
- The extension **does not store** pasted targets, usernames, passwords, history or parsed results.
- The extension **does not read the active tab**.
- The extension requests **zero browser permissions and zero host permissions**.
- The extension **does not connect to your FTP, FTPS, or SFTP server**.
- No HTTP request, WebSocket, analytics SDK, crash reporter, browser storage or background/content script is present.
- Passwords are detected only to warn the user and are never rendered into result fields or copied into the Safe target.
- Firefox declares `data_collection_permissions.required = ["none"]` and uses the stable ID `ghostftp-connection-helper@ghostftp.com` for signed distribution.

See `PRIVACY.md` for the concise data-handling statement.

## Local validation

```bash
python scripts/build_browser_extensions.py --check
python scripts/test_browser_extensions_contract.py
python scripts/build_browser_extensions.py
```

The dedicated GitHub Actions browser workflow repeats the contract test, builds all three ZIPs, verifies their contents and publishes them as CI artifacts.

## Unpacked development testing

Build the packages first, extract the browser ZIP you want to test, then load the extracted directory with the browser's extension-development UI. For Chrome use `chrome://extensions`; for Edge use `edge://extensions`; for Firefox use `about:debugging#/runtime/this-firefox` and select `manifest.json`.

The generated packages are the canonical browser inputs. Do not maintain browser-specific copies of `core.js`, `popup.js`, `popup.css`, `popup.html` or the Ghost FTP icon; all three packages must be produced from `shared/`.
