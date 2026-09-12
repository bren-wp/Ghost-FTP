# Ghost FTP browser extensions

Ghost FTP Connection Helper is a small, privacy-first browser extension for preparing FTP connection targets without sending connection data anywhere.

## Browser packages

- `chromium/` — one shared Manifest V3 package for **Google Chrome**, **Microsoft Edge**, **Opera**, **Brave**, and **Vivaldi**.
- `firefox/` — the equivalent Manifest V3 package for **Mozilla Firefox**.

Both packages use the same local runtime source and the same Ghost FTP product version. The separate folders make each package directly loadable in its browser without a build step.

## What it does

Paste an `ftp://`, `ftps://`, or `sftp://` address into the popup. The helper validates the scheme and host, then shows protocol, host, port, username, and path. The **Safe target** intentionally excludes all URL user information, including username and password, and also excludes query and fragment data. Individual displayed values can be copied only after an explicit user click.

The extension does not launch the desktop client directly because Ghost FTP does not currently expose a supported browser-to-desktop URI or native-messaging contract. A future handoff must be implemented and reviewed separately instead of inventing an unsafe or unsupported `ghostftp://` protocol.

## Privacy and security

- **No telemetry.**
- **No tracking.**
- **No remote code.** All HTML, CSS, JavaScript, and the icon are packaged locally.
- The extension **does not store** pasted targets, usernames, passwords, history, or parsed results.
- The extension **does not read the active tab** and requests no tab, host, history, storage, clipboard, or scripting permissions.
- The extension **does not connect to your FTP, FTPS, or SFTP server**. It only parses text locally in the popup.
- It makes no HTTP requests, opens no WebSocket, and sends no analytics or crash reports.
- Passwords are never rendered back to the page and never appear in the generated safe target.
- The safe target omits username, password, query, and fragment data to reduce accidental credential/token disclosure when copying.

See `PRIVACY.md` for the concise data-handling contract.

## Install for development / unpacked testing

### Google Chrome

1. Open `chrome://extensions`.
2. Enable **Developer mode**.
3. Choose **Load unpacked**.
4. Select `ekstenzije/chromium`.

### Microsoft Edge

1. Open `edge://extensions`.
2. Enable **Developer mode**.
3. Choose **Load unpacked**.
4. Select `ekstenzije/chromium`.

### Opera

1. Open `opera://extensions`.
2. Enable developer mode.
3. Choose **Load unpacked**.
4. Select `ekstenzije/chromium`.

### Brave

1. Open `brave://extensions`.
2. Enable **Developer mode**.
3. Choose **Load unpacked**.
4. Select `ekstenzije/chromium`.

### Vivaldi

1. Open `vivaldi://extensions`.
2. Enable **Developer mode**.
3. Choose **Load unpacked**.
4. Select `ekstenzije/chromium`.

### Mozilla Firefox

1. Open `about:debugging#/runtime/this-firefox`.
2. Choose **Load Temporary Add-on**.
3. Select `ekstenzije/firefox/manifest.json`.

Temporary/unpacked installation is intended for development and validation. Store publication and signing are deliberately outside this change.

## Usage

1. Open the Ghost FTP toolbar popup.
2. Paste a connection target, for example `sftp://user@example.invalid/home/user`.
3. Select **Analyze locally**.
4. Review the parsed fields and security note.
5. Use **Copy safe target** or one of the field-level copy buttons when needed.
6. Select **Clear** to remove the current popup contents immediately.

No example includes a real server or credential.
